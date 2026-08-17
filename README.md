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

### Başka bir makineye/sunucuya taşırken

Proje varsayılan olarak `localhost` üzerinden çalışacak şekilde ayarlı.
Kurumun kendi sunucusuna veya farklı bir makineye taşınırken sadece `.env`
dosyasındaki şu değerin güncellenmesi yeterlidir — kod değişikliği gerekmez:

```
FRONTEND_ORIGIN=http://<sunucunun-adresi>:3001
```

Bu adres, backend'in CORS için "hangi adresten gelen isteklere izin
vereceğini" belirler; `.env`'de bu satır güncellenmeden farklı bir adresten
açılırsa tarayıcıda CORS hatası alınır. Bunun dışında adım aynıdır: `.env`
oluştur → gerçek şifreler/`JWT_SECRET` gir → `docker compose up --build`.
Gerçek bir domain/TLS sertifikası (Let's Encrypt vb.) kullanılacaksa bu,
projenin önüne konacak bir reverse proxy (Caddy/nginx) ile ayrıca
yapılandırılır — istenirse bu kurulum için ayrı bir `docker-compose.prod.yml`
hazırlanabilir.

## API Uç Noktaları

| Metod  | Yol                        | Açıklama                                  | Auth |
|--------|----------------------------|--------------------------------------------|------|
| GET    | `/health`                  | Veritabanı bağlantısını da kontrol eder     | Hayır |
| POST   | `/api/auth/register`       | Kayıt (username, email, password)           | Hayır |
| POST   | `/api/auth/login`          | Giriş (email, password) → JWT               | Hayır |
| POST   | `/api/auth/logout-all`     | Bu kullanıcıya ait tüm token'ları geçersiz kılar | Evet |
| POST   | `/api/auth/forgot-password`| Şifre sıfırlama bağlantısı üretir (bkz. not aşağıda) | Hayır |
| POST   | `/api/auth/reset-password` | Token ile yeni şifre belirler                | Hayır |
| PATCH  | `/api/me`                  | Kullanıcı adı/e-posta günceller              | Evet |
| POST   | `/api/me/password`         | Şifre değiştirir (tüm oturumları geçersiz kılar) | Evet |
| DELETE | `/api/me`                  | Kendi hesabını ve tüm verilerini kalıcı siler (KVKK) | Evet |
| POST   | `/api/documents`           | Dosya yükle (multipart: file, lang)         | Evet |
| GET    | `/api/documents`           | Belge listesi (?page, ?page_size)           | Evet |
| GET    | `/api/documents/:id`       | Belge detayı + OCR metni                    | Evet |
| GET    | `/api/documents/:id/text`  | .txt indir                                  | Evet |
| GET    | `/api/documents/:id/pdf`   | Aranabilir .pdf indir                       | Evet |
| DELETE | `/api/documents/:id`       | Belgeyi ve dosyalarını sil                  | Evet |
| GET    | `/api/admin/users`         | Tüm kullanıcıları listeler                  | Evet (admin) |
| GET    | `/api/admin/documents`     | Tüm belgeleri listeler                      | Evet (admin) |
| DELETE | `/api/admin/users/:id`     | Bir kullanıcıyı siler                       | Evet (admin) |
| GET    | `/api/admin/audit-logs`    | Admin işlem geçmişini listeler              | Evet (admin) |

**Not (`/api/auth/forgot-password`):** `.env` dosyasında `SMTP_*` değişkenleri
doldurulmuşsa sıfırlama bağlantısı gerçekten e-postayla gönderilir
(`net/smtp`, Gmail dahil STARTTLS destekleyen sağlayıcılarla çalışır — bkz.
`.env.example` içindeki Gmail App Password notu). Boş bırakılırsa (varsayılan),
bağlantı e-posta yerine backend loglarına yazılır — `docker compose logs
backend` ile görülebilir (`[şifre sıfırlama] ... bağlantı: ...` satırını arayın).
Bu, SMTP kurulmadan da özelliği test edebilmek için kasıtlı bir fallback'tir.

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
- Rol tabanlı yetkilendirme (RBAC): `admin`/`user` rolleri, admin'e özel uç
  noktalar hem frontend'de (link gizleme) hem backend'de (`RequireAdmin`
  middleware) korunuyor — sadece linki gizlemek yeterli değildir.
- Hesap kilitleme: art arda 5 başarısız giriş denemesinden sonra hesap 15
  dakika kilitleniyor (brute-force koruması, IP bazlı rate limiting'e ek
  olarak hesap bazlı).
- Admin panelinden yapılan kullanıcı silme işlemleri `audit_logs` tablosuna
  kaydediliyor (kim, ne zaman, neyi sildi).
- Kullanıcılar kendi hesaplarını ve tüm verilerini (`DELETE /api/me`) kalıcı
  olarak silebiliyor (KVKK "unutulma hakkı"na karşılık gelir).

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
- **E-posta doğrulama (kayıt sonrası "e-postanı onayla" akışı) eklenmedi**:
  SMTP artık isteğe bağlı olarak kurulabiliyor (bkz. yukarısı), ama bu akış
  eklenmedi çünkü yanlış/erişilemez bir e-postayla kayıt olunması durumunda
  kullanıcıyı kalıcı olarak giriş yapamaz hale getirebilir — bu projenin
  kapsamında (öğrenciler/personel için dahili bir araç) riski faydasından
  yüksek görüldü. SMTP altyapısı (mailer paketi) zaten hazır olduğu için
  ileride eklenmesi kolay.
- **Refresh-token rotasyonu yok**: access token 72 saat geçerli ve
  `token_version` ile geri alınabilir (revocation). Bu, ayrı bir refresh
  token akışının sağladığı güvenliğin büyük kısmını (çalınan bir token'ı
  anında geçersiz kılabilme) zaten karşılıyor; tam refresh-token rotasyonu
  (kısa ömürlü access + ayrı refresh token, httpOnly cookie yönetimi) daha
  büyük bir mimari değişiklik olduğu için bu projenin kapsamı dışında
  bırakıldı.
- **Dosya taraması ve şifreleme yok**: yüklenen dosyalarda malware taraması
  (ör. ClamAV) ve diskte şifreleme (AES-256 at rest) bilinçli olarak
  eklenmedi — bir öğrenci/staj projesi için ek altyapı karmaşıklığının
  getirisi düşük görüldü.

## Testler

Backend:
```
cd backend
go test ./...                    # unit + router smoke testleri (sqlmock, DB gerekmez)
go test -tags=integration ./...  # gerçek tesseract gerektiren testler

# Migration'ları gerçek bir Postgres'e karşı test etmek için (opsiyonel):
docker compose up -d postgres
TEST_DATABASE_URL="postgres://baum:baum@localhost:5432/baum?sslmode=disable" \
  go test -tags=integration ./internal/db/...
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
