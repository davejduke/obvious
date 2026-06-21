package signer_test

import (
	"testing"

	"github.com/davejduke/obvious/services/webhooks/internal/signer"
)

func TestSign_ProducesHexPrefixedSignature(t *testing.T) {
	sig := signer.Sign("my-secret", []byte(`{"type":"test"}`))
	if len(sig) == 0 {
		t.Fatal("expected non-empty signature")
	}
	if sig[:7] != "sha256=" {
		t.Fatalf("expected signature to start with 'sha256=', got %q", sig)
	}
	// hex portion must be 64 chars (32 bytes * 2)
	if len(sig) != 7+64 {
		t.Fatalf("expected signature length 71, got %d", len(sig))
	}
}

func TestSign_SameInputSameOutput(t *testing.T) {
	payload := []byte(`{"type":"evidence.intake.complete"}`)
	s1 := signer.Sign("supersecret", payload)
	s2 := signer.Sign("supersecret", payload)
	if s1 != s2 {
		t.Fatalf("expected identical signatures for same inputs, got %q vs %q", s1, s2)
	}
}

func TestSign_DifferentSecretsDifferentSignatures(t *testing.T) {
	payload := []byte(`{"type":"evidence.intake.complete"}`)
	s1 := signer.Sign("secret-a", payload)
	s2 := signer.Sign("secret-b", payload)
	if s1 == s2 {
		t.Fatal("expected different signatures for different secrets")
	}
}

func TestVerify_ValidSignature(t *testing.T) {
	payload := []byte(`{"type":"finding.status.changed"}`)
	sig := signer.Sign("signing-key", payload)
	if !signer.Verify("signing-key", payload, sig) {
		t.Fatal("expected Verify to return true for correct signature")
	}
}

func TestVerify_TamperedPayload(t *testing.T) {
	payload := []byte(`{"type":"finding.status.changed"}`)
	sig := signer.Sign("signing-key", payload)
	tampered := []byte(`{"type":"finding.status.changed","injected":true}`)
	if signer.Verify("signing-key", tampered, sig) {
		t.Fatal("expected Verify to return false for tampered payload")
	}
}

func TestVerify_WrongSecret(t *testing.T) {
	payload := []byte(`{"type":"evidence.intake.complete"}`)
	sig := signer.Sign("correct-secret", payload)
	if signer.Verify("wrong-secret", payload, sig) {
		t.Fatal("expected Verify to return false for wrong secret")
	}
}

func TestVerify_EmptySignature(t *testing.T) {
	payload := []byte(`{"type":"test"}`)
	if signer.Verify("secret", payload, "") {
		t.Fatal("expected Verify to return false for empty signature")
	}
}

func TestHashSecret_DeterministicAndNonEmpty(t *testing.T) {
	h1 := signer.HashSecret("my-webhook-secret")
	h2 := signer.HashSecret("my-webhook-secret")
	if h1 != h2 {
		t.Fatal("HashSecret must be deterministic")
	}
	if len(h1) != 64 { // SHA-256 = 32 bytes = 64 hex chars
		t.Fatalf("expected 64-char hex hash, got %d", len(h1))
	}
}

func TestHashSecret_DifferentSecretsProduceDifferentHashes(t *testing.T) {
	h1 := signer.HashSecret("secret-one")
	h2 := signer.HashSecret("secret-two")
	if h1 == h2 {
		t.Fatal("expected different secrets to produce different hashes")
	}
}

