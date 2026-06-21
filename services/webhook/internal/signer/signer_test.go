package signer

import (
	"testing"
	"time"
)

var (
	testSecret  = "s3cr3t"
	testBody    = []byte(`{"event":"test.created","payload":{"id":"abc"}}`)
	testTime    = time.Unix(1720000000, 0)
)

// TestSignDeterministic verifies the same inputs always produce the same signature.
func TestSignDeterministic(t *testing.T) {
	s1 := Sign(testSecret, testBody, testTime)
	s2 := Sign(testSecret, testBody, testTime)
	if s1 != s2 {
		t.Errorf("Sign not deterministic: %q vs %q", s1, s2)
	}
	if len(s1) != 64 {
		t.Errorf("expected 64-char hex, got %d", len(s1))
	}
}

// TestVerifyValid checks a valid signature passes.
func TestVerifyValid(t *testing.T) {
	sig := Sign(testSecret, testBody, testTime)
	if !Verify(testSecret, testBody, testTime, sig) {
		t.Error("expected Verify to return true for valid signature")
	}
}

// TestVerifyWrongSecret checks that a different secret fails verification.
func TestVerifyWrongSecret(t *testing.T) {
	sig := Sign("wrong-secret", testBody, testTime)
	if Verify(testSecret, testBody, testTime, sig) {
		t.Error("expected Verify to return false for wrong secret")
	}
}

// TestVerifyWrongTimestamp checks that a different timestamp fails verification.
func TestVerifyWrongTimestamp(t *testing.T) {
	sig := Sign(testSecret, testBody, testTime)
	if Verify(testSecret, testBody, testTime.Add(1*time.Second), sig) {
		t.Error("expected Verify to return false for different timestamp")
	}
}

// TestVerifyTamperedBody checks that modifying the body fails verification.
func TestVerifyTamperedBody(t *testing.T) {
	sig := Sign(testSecret, testBody, testTime)
	tampered := append(testBody, '!')
	if Verify(testSecret, tampered, testTime, sig) {
		t.Error("expected Verify to return false for tampered body")
	}
}

// TestSignDifferentTimestampsProduceDifferentSigs verifies replay protection.
func TestSignDifferentTimestampsProduceDifferentSigs(t *testing.T) {
	s1 := Sign(testSecret, testBody, testTime)
	s2 := Sign(testSecret, testBody, testTime.Add(1*time.Second))
	if s1 == s2 {
		t.Error("different timestamps should produce different signatures")
	}
}

