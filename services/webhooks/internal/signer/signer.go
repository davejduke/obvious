// Package signer implements HMAC-SHA256 webhook payload signing per §4.5.
//
// The signature is computed as:
//
//	 HMAC-SHA256(secret, payload_bytes)
//
// and delivered in the X-AIAUDITOR-Signature header as:
//
//	 sha256=<lowercase_hex_digest>
//
// Recipients verify by recomputing the HMAC with their stored secret and
// comparing in constant time to prevent timing attacks.
package signer

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

const (
	// SignatureHeader is the HTTP header that carries the HMAC-SHA256 signature.
	SignatureHeader = "X-AIAUDITOR-Signature"

	// signaturePrefix is the algorithm prefix in the header value.
	signaturePrefix = "sha256="
)

// Sign computes the HMAC-SHA256 of payload using secret and returns the
// full header value in the form "sha256=<hex>".
func Sign(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return signaturePrefix + hex.EncodeToString(mac.Sum(nil))
}

// Verify reports whether the provided signature matches the HMAC-SHA256 of
// payload computed with secret. Comparison is constant-time to prevent
// timing side-channels.
func Verify(secret string, payload []byte, signature string) bool {
	expected := Sign(secret, payload)
	if len(signature) < len(signaturePrefix) {
		return false
	}
	return hmac.Equal([]byte(expected), []byte(signature))
}

// HashSecret returns the SHA-256 hex-encoded hash of the secret. This is
// stored in webhook_subscriptions.secret_hash for breach-safe auditing; the
// plaintext is stored separately in secret_value.
func HashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return fmt.Sprintf("%x", sum)
}

