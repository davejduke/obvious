// Package crypto provides Argon2id password hashing for the identity service.
package crypto

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Params configures the Argon2id parameters.
type Params struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLen     uint32
	KeyLen      uint32
}

// DefaultParams are OWASP-recommended Argon2id settings.
var DefaultParams = &Params{
	Memory:      65536, // 64 MiB
	Iterations:  3,
	Parallelism: 2,
	SaltLen:     16,
	KeyLen:      32,
}

// HashPassword produces an encoded Argon2id hash string.
func HashPassword(password string) (string, error) {
	p := DefaultParams
	salt := make([]byte, p.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("crypto: generate salt: %w", err)
	}
	hash := argon2.IDKey([]byte(password), salt, p.Iterations, p.Memory, p.Parallelism, p.KeyLen)
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)
	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.Memory, p.Iterations, p.Parallelism, b64Salt, b64Hash)
	return encoded, nil
}

// VerifyPassword compares a plaintext password against an encoded hash.
// Returns nil on match, ErrPasswordMismatch on mismatch.
func VerifyPassword(password, encoded string) error {
	p, salt, hash, err := decodeHash(encoded)
	if err != nil {
		return fmt.Errorf("crypto: decode hash: %w", err)
	}
	other := argon2.IDKey([]byte(password), salt, p.Iterations, p.Memory, p.Parallelism, p.KeyLen)
	if subtle.ConstantTimeCompare(hash, other) != 1 {
		return ErrPasswordMismatch
	}
	return nil
}

// ErrPasswordMismatch is returned when the password does not match the stored hash.
var ErrPasswordMismatch = errors.New("crypto: password mismatch")

func decodeHash(encoded string) (*Params, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return nil, nil, nil, errors.New("invalid hash format")
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return nil, nil, nil, err
	}
	p := &Params{}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.Memory, &p.Iterations, &p.Parallelism); err != nil {
		return nil, nil, nil, err
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, nil, err
	}
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, nil, err
	}
	p.KeyLen = uint32(len(hash))
	p.SaltLen = uint32(len(salt))
	return p, salt, hash, nil
}
