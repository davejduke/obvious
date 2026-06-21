package generate_test

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"strings"
	"testing"
	"time"

	"github.com/davejduke/obvious/services/reporting/internal/generate"
	"github.com/davejduke/obvious/services/reporting/internal/template"
	"github.com/google/uuid"
)

// ─── Helpers ─────────────────────────────────────────────────────────────────

// generateTestKeyPair creates a fresh RSA-2048 key pair for tests.
func generateTestKeyPair(t *testing.T) (*rsa.PrivateKey, string) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	pubDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
	return priv, string(pubPEM)
}

func sampleRequest(pubKeyPEM string) generate.EvidencePackageRequest {
	return generate.EvidencePackageRequest{
		EngagementID: "eng-" + uuid.New().String()[:8],
		OrgName:      "Acme Corp",
		Findings: []template.Finding{
			{
				ID:             uuid.New(),
				Ref:            "NIS2-001",
				Title:          "MFA not enforced",
				Description:    "Privileged accounts lack MFA.",
				Severity:       template.SeverityCritical,
				Recommendation: "Enable MFA on all privileged accounts.",
			},
			{
				ID:          uuid.New(),
				Ref:         "NIS2-002",
				Title:       "Patch gap",
				Description: "Critical patches delayed.",
				Severity:    template.SeverityHigh,
			},
		},
		Evidence: []template.EvidenceItem{
			{
				ID:          uuid.New(),
				Title:       "Screenshot — admin console",
				Description: "Shows MFA is disabled.",
				SourceType:  "screenshot",
				CollectedAt: time.Now().UTC(),
				CollectedBy: "aiauditor",
			},
		},
		WorkingPapers: []generate.WorkingPaper{
			{
				ID:        "wp-001",
				Title:     "MFA Assessment",
				Author:    "Jane Auditor",
				Content:   "Tested 50 privileged accounts; 0 had MFA enabled.",
				CreatedAt: time.Now().UTC(),
			},
		},
		AuditTrail: []generate.AuditTrailEntry{
			{
				Timestamp:   time.Now().UTC(),
				Actor:       "aiauditor",
				Action:      "evidence.collected",
				Description: "Collected admin console screenshot.",
			},
		},
		RSAPublicKeyPEM: pubKeyPEM,
	}
}

// ─── Tests ───────────────────────────────────────────────────────────────────

func TestEvidenceZIPGenerator_Unencrypted(t *testing.T) {
	g := generate.NewEvidenceZIPGenerator()
	req := sampleRequest("")
	req.RSAPublicKeyPEM = "" // no encryption

	result, err := g.Generate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Encrypted {
		t.Error("expected Encrypted=false for unencrypted request")
	}
	if result.PackageZIP == nil {
		t.Fatal("expected PackageZIP to be populated")
	}
	if result.EncryptedPackage != nil {
		t.Error("expected EncryptedPackage to be nil for unencrypted request")
	}
}

func TestEvidenceZIPGenerator_ZIPContainsAllFiles(t *testing.T) {
	g := generate.NewEvidenceZIPGenerator()
	req := sampleRequest("")
	req.RSAPublicKeyPEM = ""

	result, err := g.Generate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r, err := zip.NewReader(bytes.NewReader(result.PackageZIP), int64(len(result.PackageZIP)))
	if err != nil {
		t.Fatalf("invalid ZIP: %v", err)
	}

	expected := map[string]bool{
		"findings.json":       false,
		"evidence.json":       false,
		"working_papers.json": false,
		"audit_trail.json":    false,
		"manifest.json":       false,
	}
	for _, f := range r.File {
		expected[f.Name] = true
	}
	for name, found := range expected {
		if !found {
			t.Errorf("missing file in ZIP: %s", name)
		}
	}
}

func TestEvidenceZIPGenerator_ManifestChecksums(t *testing.T) {
	g := generate.NewEvidenceZIPGenerator()
	req := sampleRequest("")
	req.RSAPublicKeyPEM = ""

	result, err := g.Generate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	manifest := result.Manifest
	if len(manifest.Files) == 0 {
		t.Fatal("manifest has no file checksums")
	}

	// Every checksum must be a 64-char hex SHA-256.
	for _, fc := range manifest.Files {
		if len(fc.SHA256) != 64 {
			t.Errorf("file %s: expected 64-char SHA-256, got %d chars", fc.Filename, len(fc.SHA256))
		}
		if fc.SizeBytes <= 0 {
			t.Errorf("file %s: expected positive size, got %d", fc.Filename, fc.SizeBytes)
		}
	}

	// Must include the core files.
	filenames := make(map[string]bool)
	for _, fc := range manifest.Files {
		filenames[fc.Filename] = true
	}
	required := []string{"findings.json", "evidence.json", "working_papers.json", "audit_trail.json"}
	for _, name := range required {
		if !filenames[name] {
			t.Errorf("manifest missing checksum for %s", name)
		}
	}
}

func TestEvidenceZIPGenerator_ManifestEngagementID(t *testing.T) {
	g := generate.NewEvidenceZIPGenerator()
	req := sampleRequest("")
	req.RSAPublicKeyPEM = ""
	req.EngagementID = "eng-test-123"

	result, err := g.Generate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Manifest.EngagementID != "eng-test-123" {
		t.Errorf("expected engagement_id=eng-test-123, got %s", result.Manifest.EngagementID)
	}
	if result.Manifest.PackageID == "" {
		t.Error("expected non-empty package_id")
	}
}

func TestEvidenceZIPGenerator_EncryptedResult(t *testing.T) {
	priv, pubPEM := generateTestKeyPair(t)
	g := generate.NewEvidenceZIPGenerator()
	req := sampleRequest(pubPEM)

	result, err := g.Generate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Encrypted {
		t.Fatal("expected Encrypted=true")
	}
	if result.PackageZIP != nil {
		t.Error("expected PackageZIP=nil for encrypted request")
	}
	if len(result.EncryptedPackage) == 0 {
		t.Error("expected non-empty EncryptedPackage")
	}
	if len(result.EncryptedSessionKey) == 0 {
		t.Error("expected non-empty EncryptedSessionKey")
	}

	// Verify round-trip: decrypt the package, extract ZIP, read files.
	plainZIP, err := generate.DecryptPackage(result.EncryptedPackage, result.EncryptedSessionKey, priv)
	if err != nil {
		t.Fatalf("decrypt package: %v", err)
	}

	r, err := zip.NewReader(bytes.NewReader(plainZIP), int64(len(plainZIP)))
	if err != nil {
		t.Fatalf("invalid decrypted ZIP: %v", err)
	}

	fileNames := make([]string, 0, len(r.File))
	for _, f := range r.File {
		fileNames = append(fileNames, f.Name)
	}
	if len(fileNames) < 5 {
		t.Errorf("expected at least 5 files in ZIP, got %d: %v", len(fileNames), fileNames)
	}
}

func TestEvidenceZIPGenerator_ManifestEncryptedFlag(t *testing.T) {
	_, pubPEM := generateTestKeyPair(t)
	g := generate.NewEvidenceZIPGenerator()

	// Encrypted request → manifest.Encrypted = true
	req := sampleRequest(pubPEM)
	result, _ := g.Generate(req)
	if !result.Manifest.Encrypted {
		t.Error("expected manifest.Encrypted=true")
	}

	// Unencrypted request → manifest.Encrypted = false
	req.RSAPublicKeyPEM = ""
	result, _ = g.Generate(req)
	if result.Manifest.Encrypted {
		t.Error("expected manifest.Encrypted=false")
	}
}

func TestEvidenceZIPGenerator_InvalidPublicKey(t *testing.T) {
	g := generate.NewEvidenceZIPGenerator()
	req := sampleRequest("this-is-not-a-valid-pem-key")

	_, err := g.Generate(req)
	if err == nil {
		t.Fatal("expected error for invalid public key")
	}
	if !strings.Contains(err.Error(), "PEM") && !strings.Contains(err.Error(), "encrypt") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestEvidenceZIPGenerator_RSAKeyTooSmall(t *testing.T) {
	// Generate an RSA-1024 key (below minimum).
	priv, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	pubDER, _ := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	pubPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}))

	g := generate.NewEvidenceZIPGenerator()
	req := sampleRequest(pubPEM)

	_, err = g.Generate(req)
	if err == nil {
		t.Fatal("expected error for RSA-1024 key")
	}
	if !strings.Contains(err.Error(), "2048") {
		t.Errorf("expected error message mentioning 2048, got: %v", err)
	}
}

func TestEvidenceZIPGenerator_EmptyFindings(t *testing.T) {
	g := generate.NewEvidenceZIPGenerator()
	req := generate.EvidencePackageRequest{
		EngagementID: "eng-empty",
		OrgName:      "Empty Corp",
	}

	result, err := g.Generate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.PackageZIP == nil {
		t.Fatal("expected PackageZIP even for empty request")
	}
}

func TestEvidenceZIPGenerator_PackageIDsAreUnique(t *testing.T) {
	g := generate.NewEvidenceZIPGenerator()
	req := generate.EvidencePackageRequest{EngagementID: "eng-1", OrgName: "Org"}

	r1, _ := g.Generate(req)
	r2, _ := g.Generate(req)
	if r1.Manifest.PackageID == r2.Manifest.PackageID {
		t.Error("expected unique package IDs across runs")
	}
}
