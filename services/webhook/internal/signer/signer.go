// Package signer provides HMAC-SHA256 webhook payload signing.
package signer

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// SignatureHeader is the HTTP request header carrying the HMAC-SHA256 signature.
const SignatureHeader = "X-Webhook-Signature"

// TimestampHeader carries the Unix epoch used during signing (replay protection).
const TimestampHeader = "X-Webhook-Timestamp"

// Sign produces an HMAC-SHA256 hex-encoded signature for the given payload.
// Format: HMAC-SHA256(secret, "<timestamp>.<body>")
// The timestamp is included so consumers can reject stale payloads.
func Sign(secret string, body []byte, ts time.Time) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = fmt.Fprintf(mac, "%d.", ts.Unix())
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify returns true when the provided signature matches a freshly computed one.
func Verify(secret string, body []byte, ts time.Time, sig string) bool {
	expected := Sign(secret, body, ts)
	// Constant-time comparison to resist timing attacks.
	return hmac.Equal([]byte(expected), []byte(sig))
}

