# BAUM PDF OCR Web Servisi

Mersin Üniversitesi BAUM Yaz Staj 2026 kapsamında geliştirilen, yüklenen PDF/görüntü
dosyalarını Tesseract OCR ile metne çeviren, sonucu düz metin (.txt) ve aranabilir
PDF olarak sunan bir web servisi.

## Özellikler

- E-posta + kullanıcı adı ile kayıt, e-posta ile giriş (JWT tabanlı oturum)
- PDF / PNG / JPG / TIFF yükleme, dil seçimi (Türkçe, İngilizce, Türkçe+İngilizce)
- Arka planda kuyruklu OCR işleme (Tesseract + poppler-utils)
- Sonuçları .txt ve aranabilir PDF olarak indirme
- Belge listesi (sayfalama), belge silme
- Sunucu çökse/yeniden başlasa bile yarım kalan işlerin otomatik yeniden kuyruğa
  alınması (bkz. "Kuyruk dayanıklılığı" bölümü)
- "Tüm cihazlardan çıkış yap" — bir token'ı süresi dolmadan geçersiz kılabilme
- HTTPS (kendinden imzalı, yerel/demo amaçlı) + HTTP
- pgAdmin ile veritabanını görsel olarak inceleme

## Mimari

```
tarayıcı ──▶ nginx (frontend container, non-root, :8080/:8443)
                │  statik React build'i sunar
                │  /api/* isteklerini backend'e proxy'ler
                ▼
            Go backend (Gin, :8080, non-root, graceful shutdown)
                │  JWT auth (+ revocation), dosya yükleme, iş kuyruğu
                ▼
            PostgreSQL (kullanıcılar, belge metadata'sı, bağlantı havuzu sınırlı)
                │
            Tesseract + poppler-utils (OCR işleme, aynı container içinde binary)
```

## Kurulum

1. Bu depoyu klonla/aç.
2. `.env` dosyasını oluştur:
   ```
   cp .env.example .env
   ```
   İçindeki `POSTGRES_PASSWORD`, `JWT_SECRET`, `PGADMIN_DEFAULT_PASSWORD`
   değerlerini kendi belirlediğin güçlü değerlerle değiştir.
3. Servisleri ayağa kaldır:
   ```
   docker compose up --build
   ```
4. Tarayıcıda:
   - Uygulama (HTTP): `http://localhost:3001`
   - Uygulama (HTTPS, kendinden imzalı): `https://localhost:3443` — tarayıcı
     "bağlantı güvenli değil" uyarısı gösterir, bu beklenen bir durumdur
     (gerçek bir sertifika otoritesi tarafından imzalanmadığı için); "Gelişmiş
     > Yine de devam et" ile geçebilirsin.
   - pgAdmin: `http://localhost:5051`
   - Backend health check: `http://localhost:8082/health`

Portlar host makinende zaten kullanımdaysa `docker-compose.yml`'deki
`ports:` eşlemelerini (sol taraf host portu) değiştirebilirsin.

## API Uç Noktaları

| Metod  | Yol                        | Açıklama                                  | Auth |
|--------|----------------------------|--------------------------------------------|------|
| GET    | `/health`                  | Veritabanı bağlantısını da kontrol eder     | Hayır |
| POST   | `/api/auth/register`       | Kayıt (username, email, password)           | Hayır |
| POST   | `/api/auth/login`          | Giriş (email, password) → JWT               | Hayır |
| POST   | `/api/auth/logout-all`     | Bu kullanıcıya ait tüm token'ları geçersiz kılar | Evet |
| POST   | `/api/documents`           | Dosya yükle (multipart: file, lang)         | Evet |
| GET    | `/api/documents`           | Belge listesi (?page, ?page_size)           | Evet |
| GET    | `/api/documents/:id`       | Belge detayı + OCR metni                    | Evet |
| GET    | `/api/documents/:id/text`  | .txt indir                                  | Evet |
| GET    | `/api/documents/:id/pdf`   | Aranabilir .pdf indir                       | Evet |
| DELETE | `/api/documents/:id`       | Belgeyi ve dosyalarını sil                  | Evet |

Auth gerektiren uç noktalarda `Authorization: Bearer <token>` başlığı gerekir.

## Kuyruk Dayanıklılığı

İş kuyruğu bellek içi bir Go kanalıdır — süreç yeniden başladığında kuyruk
sıfırlanır. Sunucu her başladığında `status IN ('queued','processing')` olan
tüm belgeleri veritabanından bulup otomatik olarak yeniden kuyruğa alır
(`internal/ocr/reconciler.go`).

## Güvenlik Notları

- Şifreler bcrypt ile hashleniyor.
- JWT'ler geri alınabilir (revocation): her kullanıcının `token_version`'ı
  var, token içine gömülüyor ve her istekte DB'den doğrulanıyor. "Tüm
  cihazlardan çıkış yap" bu sayıyı artırıp eski token'ları anında geçersiz
  kılar — normalde JWT'ler süresi (72 saat) dolana kadar iptal edilemez,
  burada edilebiliyor.
- Yüklenen dosyalar magic-byte (dosya imzası) kontrolünden geçiyor.
- `/api/auth/*` uç noktaları IP başına dakikada 10 istekle sınırlı.
- CORS sadece `FRONTEND_ORIGIN` ile tanımlı origin'e izin veriyor.
- Backend release modda çalışıyor, sadece Docker'ın iç ağından gelen
  `X-Forwarded-For` başlığına güveniyor.
- Container'lar non-root kullanıcıyla çalışıyor (backend: `baum`,
  frontend: `nginx`/nginx-unprivileged).
- nginx güvenlik header'ları (`X-Content-Type-Options`, `X-Frame-Options`,
  `Referrer-Policy`) ekleniyor.
- HTTPS desteği var (yerel/demo için kendinden imzalı sertifika).
- CI'da `govulncheck` (backend) ve `npm audit` (frontend) ile bağımlılık
  güvenlik açığı taraması yapılıyor.
- Tüm sırlar (`.env`) `.gitignore`'da, repoya gitmiyor.

## Operasyonel Notlar

- **Graceful shutdown**: backend `SIGTERM`/`SIGINT` yakalar, devam eden HTTP
  isteklerinin bitmesini bekler (en fazla 10 saniye) sonra kapanır.
- **Bağlantı havuzu**: Postgres bağlantıları `SetMaxOpenConns(20)` ile
  sınırlı, sınırsız bağlantı açıp DB'yi boğmayı engelliyor.

## Bilinen Sınırlamalar

- **Rate limiter ve iş kuyruğu tek instance için**: bellek içi tutuluyor,
  birden fazla backend replikası çalıştırılırsa paylaşımlı bir store
  (Redis) gerekir.
- **Dosya depolama**: S3/MinIO yerine Docker volume kullanılıyor.
- **Postgres yedekleme**: otomatik bir yedekleme stratejisi yok.
- **Satır seviyesi güvenlik (RLS)**: Postgres'te aktif değil, izolasyon
  uygulama kodundaki `WHERE user_id = $X` filtrelerine dayanıyor.
- **TLS sertifikası kendinden imzalı**: gerçek bir ortamda Let's Encrypt
  gibi bir CA'dan alınmış sertifika kullanılmalı.

## Testler

Backend:
```
cd backend
go test ./...                    # unit + router smoke testleri
go test -tags=integration ./...  # gerçek tesseract gerektiren testler
```

Frontend:
```
cd frontend
npm install
npm test
```

CI (`.github/workflows/ci.yml`) her push/PR'da backend + frontend testlerini,
derlemesini ve bağımlılık güvenlik taramasını otomatik çalıştırır.

## Proje Yapısı

```
backend/    Go API (Gin), OCR kuyruğu, migration'lar
frontend/   React + TypeScript + Vite arayüzü
pgadmin/    pgAdmin sunucu ön ayarı
docker-compose.yml  Tüm servisleri (postgres, pgadmin, backend, frontend) ayağa kaldırır
```
