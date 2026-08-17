-- Rol tabanlı yetkilendirme (admin/user) için. Var olan tüm kullanıcılar
-- varsayılan olarak 'user' rolüyle başlar; ilk kayıt olan kullanıcı
-- Register handler'ı tarafından ayrıca 'admin' yapılır.
ALTER TABLE users ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'user';
