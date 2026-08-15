-- JWT'lerin geri alınabilmesi (revocation) için: token'ın içine bu kolonun
-- o anki değeri gömülür. Kullanıcı "tüm cihazlardan çıkış yap" dediğinde
-- veya hesabı şüpheli görüldüğünde bu sayı artırılır; artık eski token'lar
-- (içlerindeki eski sürüm numarasıyla) geçersiz sayılır, süresi dolmasını
-- beklemeye gerek kalmaz.
ALTER TABLE users ADD COLUMN IF NOT EXISTS token_version INT NOT NULL DEFAULT 1;
