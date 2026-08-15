package auth

import "testing"

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("gizli-sifre")
	if err != nil {
		t.Fatalf("HashPassword hata verdi: %v", err)
	}
	if !CheckPassword(hash, "gizli-sifre") {
		t.Error("doğru şifre için CheckPassword false döndü")
	}
	if CheckPassword(hash, "yanlis-sifre") {
		t.Error("yanlış şifre için CheckPassword true döndü")
	}
}

func TestGenerateAndParseToken(t *testing.T) {
	secret := "test-secret"
	userID := "user-123"

	token, err := GenerateToken(secret, userID, 1)
	if err != nil {
		t.Fatalf("GenerateToken hata verdi: %v", err)
	}

	claims, err := ParseToken(secret, token)
	if err != nil {
		t.Fatalf("ParseToken hata verdi: %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("beklenen user id %s, alınan %s", userID, claims.UserID)
	}
	if claims.TokenVersion != 1 {
		t.Errorf("beklenen token_version 1, alınan %d", claims.TokenVersion)
	}

	if _, err := ParseToken("baska-secret", token); err == nil {
		t.Error("yanlış secret ile ParseToken hata vermeliydi")
	}
}
