// Package generate provides PDF, Excel, and evidence package generation.
package generate

import (
	"archive/zip"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"time"

	"github.com/davejduke/obvious/services/reporting/internal/template"
)

// ─── Evidence package types ──────────────────────────────────────────────────

// AuditTrailEntry represents a single entry in the audit trail excerpt.
type AuditTrailEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	Actor       string    `json:"actor"`
	Action      string    `json:"action"`
	ResourceID  string    `json:"resource_id,omitempty"`
	Description string    `json:"description"`
	Hash        string    `json:"hash,omitempty"`
}

// WorkingPaper represents a working paper document included in the evidence package.
type WorkingPaper struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Author      string    `json:"author"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"created_at"`
	Framework   string    `json:"framework,omitempty"`
	ControlRef  string    `json:"control_ref,omitempty"`
}

// EvidencePackageRequest holds all inputs needed to build an evidence package.
type EvidencePackageRequest struct {
	// EngagementID is the engagement these findings belong to.
	EngagementID string `json:"engagement_id"`
	// OrgName is the client organisation.
	OrgName string `json:"org_name"`
	// Findings is the full set of audit findings.
	Findings []template.Finding `json:"findings"`
	// Evidence is the evidence items linked to findings.
	Evidence []template.EvidenceItem `json:"evidence"`
	// WorkingPapers is the set of working papers for the engagement.
	WorkingPapers []WorkingPaper `json:"working_papers"`
	// AuditTrail is the audit trail excerpt.
	AuditTrail []AuditTrailEntry `json:"audit_trail"`
	// RSAPublicKeyPEM is the PEM-encoded RSA-2048 (or larger) public key used
	// to encrypt the package session key. If empty the ZIP is returned unencrypted
	// (useful for development/testing workflows that don't require encryption).
	RSAPublicKeyPEM string `json:"rsa_public_key_pem,omitempty"`
}

// FileChecksum holds a filename and its SHA-256 hex digest.
type FileChecksum struct {
	Filename string `json:"filename"`
	SHA256   string `json:"sha256"`
	SizeBytes int64 `json:"size_bytes"`
}

// PackageManifest lists all files in the evidence package with their checksums.
type PackageManifest struct {
	EngagementID string         `json:"engagement_id"`
	OrgName      string         `json:"org_name"`
	PackageID    string         `json:"package_id"`
	CreatedAt    time.Time      `json:"created_at"`
	Files        []FileChecksum `json:"files"`
	Encrypted    bool           `json:"encrypted"`
}

// EvidencePackageResult is the output of the evidence package generator.
type EvidencePackageResult struct {
	// Manifest describes the contents and checksums of the package.
	Manifest PackageManifest `json:"manifest"`

	// PackageZIP is the raw ZIP archive bytes. When Encrypted=true this field
	// is nil — use EncryptedPackage instead.
	PackageZIP []byte `json:"-"`

	// When encryption is requested the fields below are populated:

	// EncryptedPackage is the AES-256-GCM ciphertext of the ZIP archive.
	EncryptedPackage []byte `json:"-"`

	// EncryptedSessionKey is the RSA-OAEP ciphertext of the 32-byte AES key.
	// Decrypt with the corresponding RSA private key to recover the session key,
	// then use the session key to decrypt EncryptedPackage.
	EncryptedSessionKey []byte `json:"-"`

	// Encrypted indicates whether the package is encrypted.
	Encrypted bool `json:"encrypted"`
}

// ─── Generator ───────────────────────────────────────────────────────────────

// EvidenceZIPGenerator builds encrypted evidence packages from engagement data.
type EvidenceZIPGenerator struct{}

// NewEvidenceZIPGenerator creates an EvidenceZIPGenerator.
func NewEvidenceZIPGenerator() *EvidenceZIPGenerator { return &EvidenceZIPGenerator{} }

// Generate builds the evidence ZIP package. If RSAPublicKeyPEM is non-empty the
// ZIP bytes are encrypted using AES-256-GCM with an RSA-OAEP wrapped session key.
func (g *EvidenceZIPGenerator) Generate(req EvidencePackageRequest) (*EvidencePackageResult, error) {
	packageID := newPackageID()

	// Marshal each section to JSON.
	findingsJSON, err := marshalPretty(map[string]interface{}{
		"engagement_id": req.EngagementID,
		"org_name":      req.OrgName,
		"package_id":    packageID,
		"generated_at":  time.Now().UTC().Format(time.RFC3339),
		"findings":      req.Findings,
	})
	if err != nil {
		return nil, fmt.Errorf("evidence_zip: marshal findings: %w", err)
	}

	evidenceJSON, err := marshalPretty(map[string]interface{}{
		"engagement_id": req.EngagementID,
		"package_id":    packageID,
		"evidence":      req.Evidence,
	})
	if err != nil {
		return nil, fmt.Errorf("evidence_zip: marshal evidence: %w", err)
	}

	papersJSON, err := marshalPretty(map[string]interface{}{
		"engagement_id": req.EngagementID,
		"package_id":    packageID,
		"working_papers": req.WorkingPapers,
	})
	if err != nil {
		return nil, fmt.Errorf("evidence_zip: marshal working papers: %w", err)
	}

	trailJSON, err := marshalPretty(map[string]interface{}{
		"engagement_id": req.EngagementID,
		"package_id":    packageID,
		"audit_trail":   req.AuditTrail,
	})
	if err != nil {
		return nil, fmt.Errorf("evidence_zip: marshal audit trail: %w", err)
	}

	// Build the file map; manifest.json is added last.
	files := map[string][]byte{
		"findings.json":      findingsJSON,
		"evidence.json":      evidenceJSON,
		"working_papers.json": papersJSON,
		"audit_trail.json":   trailJSON,
	}

	// Compute SHA-256 checksums and build manifest.
	checksums := computeChecksums(files)
	manifest := PackageManifest{
		EngagementID: req.EngagementID,
		OrgName:      req.OrgName,
		PackageID:    packageID,
		CreatedAt:    time.Now().UTC(),
		Files:        checksums,
		Encrypted:    req.RSAPublicKeyPEM != "",
	}
	manifestJSON, err := marshalPretty(manifest)
	if err != nil {
		return nil, fmt.Errorf("evidence_zip: marshal manifest: %w", err)
	}
	files["manifest.json"] = manifestJSON

	// Build ZIP archive.
	zipBytes, err := buildZIP(files)
	if err != nil {
		return nil, fmt.Errorf("evidence_zip: build zip: %w", err)
	}

	result := &EvidencePackageResult{
		Manifest:  manifest,
		Encrypted: req.RSAPublicKeyPEM != "",
	}

	if req.RSAPublicKeyPEM == "" {
		result.PackageZIP = zipBytes
		return result, nil
	}

	// Encrypt with RSA-OAEP + AES-256-GCM.
	encPkg, encKey, err := encryptPackage(zipBytes, req.RSAPublicKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("evidence_zip: encrypt: %w", err)
	}
	result.EncryptedPackage = encPkg
	result.EncryptedSessionKey = encKey
	return result, nil
}

// ─── ZIP builder ─────────────────────────────────────────────────────────────

// buildZIP assembles a ZIP archive from the given filename→bytes map.
// Files are written in deterministic sorted order.
func buildZIP(files map[string][]byte) ([]byte, error) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	// Ordered file list for deterministic output.
	order := []string{
		"findings.json",
		"evidence.json",
		"working_papers.json",
		"audit_trail.json",
		"manifest.json",
	}

	for _, name := range order {
		data, ok := files[name]
		if !ok {
			continue
		}
		f, err := w.Create(name)
		if err != nil {
			return nil, fmt.Errorf("zip create %s: %w", name, err)
		}
		if _, err := f.Write(data); err != nil {
			return nil, fmt.Errorf("zip write %s: %w", name, err)
		}
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("zip close: %w", err)
	}
	return buf.Bytes(), nil
}

// ─── Encryption ──────────────────────────────────────────────────────────────

// encryptPackage encrypts plaintext with AES-256-GCM and wraps the session key
// with RSA-OAEP using the provided PEM public key.
// Returns (encrypted_package, encrypted_session_key, error).
func encryptPackage(plaintext []byte, pubKeyPEM string) ([]byte, []byte, error) {
	// Decode PEM block.
	block, _ := pem.Decode([]byte(pubKeyPEM))
	if block == nil {
		return nil, nil, fmt.Errorf("invalid PEM block for RSA public key")
	}

	var pub *rsa.PublicKey
	switch block.Type {
	case "RSA PUBLIC KEY":
		key, err := x509.ParsePKCS1PublicKey(block.Bytes)
		if err != nil {
			return nil, nil, fmt.Errorf("parse PKCS#1 public key: %w", err)
		}
		pub = key
	default:
		// Try PKIX (SubjectPublicKeyInfo) format used by most modern tools.
		keyIface, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, nil, fmt.Errorf("parse PKIX public key: %w", err)
		}
		var ok bool
		pub, ok = keyIface.(*rsa.PublicKey)
		if !ok {
			return nil, nil, fmt.Errorf("public key is not RSA")
		}
	}

	if pub.N.BitLen() < 2048 {
		return nil, nil, fmt.Errorf("RSA key must be at least 2048 bits, got %d", pub.N.BitLen())
	}

	// Generate a random 32-byte AES-256 session key.
	sessionKey := make([]byte, 32)
	if _, err := rand.Read(sessionKey); err != nil {
		return nil, nil, fmt.Errorf("generate session key: %w", err)
	}

	// Encrypt session key with RSA-OAEP (SHA-256 hash).
	encKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, sessionKey, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("rsa encrypt session key: %w", err)
	}

	// Encrypt package bytes with AES-256-GCM.
	block2, err := aes.NewCipher(sessionKey)
	if err != nil {
		return nil, nil, fmt.Errorf("aes new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block2)
	if err != nil {
		return nil, nil, fmt.Errorf("gcm new: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("generate nonce: %w", err)
	}

	// Prepend nonce to ciphertext so decryptors can extract it.
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	return ciphertext, encKey, nil
}

// DecryptPackage decrypts an evidence package using the RSA private key.
// It is provided here for test/validation purposes; production decryption
// is performed by the recipient using their private key.
func DecryptPackage(encryptedPackage, encryptedSessionKey []byte, privKey *rsa.PrivateKey) ([]byte, error) {
	// Recover the AES session key.
	sessionKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privKey, encryptedSessionKey, nil)
	if err != nil {
		return nil, fmt.Errorf("rsa decrypt session key: %w", err)
	}

	block, err := aes.NewCipher(sessionKey)
	if err != nil {
		return nil, fmt.Errorf("aes new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm new: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(encryptedPackage) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := encryptedPackage[:nonceSize], encryptedPackage[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("gcm decrypt: %w", err)
	}
	return plaintext, nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// computeChecksums returns a FileChecksum for each file in the map.
func computeChecksums(files map[string][]byte) []FileChecksum {
	order := []string{
		"findings.json",
		"evidence.json",
		"working_papers.json",
		"audit_trail.json",
	}
	out := make([]FileChecksum, 0, len(order))
	for _, name := range order {
		data, ok := files[name]
		if !ok {
			continue
		}
		h := sha256.Sum256(data)
		out = append(out, FileChecksum{
			Filename:  name,
			SHA256:    fmt.Sprintf("%x", h),
			SizeBytes: int64(len(data)),
		})
	}
	return out
}

// marshalPretty marshals v to indented JSON.
func marshalPretty(v interface{}) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

// newPackageID generates a random package identifier.
func newPackageID() string {
	b := make([]byte, 8)
	_, _ = io.ReadFull(rand.Reader, b)
	return fmt.Sprintf("pkg-%x", b)
}
