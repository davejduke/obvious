package crypto_test

import (
	"testing"

	"github.com/davejduke/obvious/services/identity/pkg/crypto"
)

func TestHashPassword_RoundTrip(t *testing.T) {
	password := "super-secret-password-123"
	hash, err := crypto.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if err := crypto.VerifyPassword(password, hash); err != nil {
		t.Fatalf("VerifyPassword failed: %v", err)
	}
}

func TestVerifyPassword_WrongPassword(t *testing.T) {
	hash, err := crypto.HashPassword("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	err = crypto.VerifyPassword("wrong-password", hash)
	if err == nil {
		t.Fatal("expected mismatch error, got nil")
	}
	if err != crypto.ErrPasswordMismatch {
		t.Fatalf("expected ErrPasswordMismatch, got: %v", err)
	}
}

func TestHashPassword_Uniqueness(t *testing.T) {
	h1, _ := crypto.HashPassword("same-password")
	h2, _ := crypto.HashPassword("same-password")
	if h1 == h2 {
		t.Fatal("expected unique hashes (different salts)")
	}
}
