package hashchain_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/davejduke/obvious/services/audit-trail/internal/hashchain"
	"github.com/davejduke/obvious/services/audit-trail/internal/model"
)

var (
	testOrgID      = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	testActorID    = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	testResourceID = uuid.MustParse("33333333-3333-3333-3333-333333333333")
	testTime       = time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
)

func baseReq() *model.AppendRequest {
	return &model.AppendRequest{
		OrgID:        testOrgID,
		ActorID:      &testActorID,
		ActorEmail:   "alice@example.com",
		Action:       "evidence.uploaded",
		EventType:    model.EventTypeEvidenceChange,
		ResourceType: "evidence",
		ResourceID:   &testResourceID,
		Changes:      map[string]interface{}{"status": "accepted"},
	}
}

// TestComputeReturnsNonEmptyHash verifies that Compute produces a non-empty
// hex string for a basic request.
func TestComputeReturnsNonEmptyHash(t *testing.T) {
	hash, err := hashchain.Compute(hashchain.GenesisHash, baseReq(), testTime)
	if err != nil {
		t.Fatalf("Compute error: %v", err)
	}
	if len(hash) != 64 {
		t.Fatalf("expected 64 hex chars (SHA-256), got %d: %q", len(hash), hash)
	}
}

// TestComputeIsDeterministic verifies that the same input always yields the same hash.
func TestComputeIsDeterministic(t *testing.T) {
	req := baseReq()
	h1, _ := hashchain.Compute(hashchain.GenesisHash, req, testTime)
	h2, _ := hashchain.Compute(hashchain.GenesisHash, req, testTime)
	if h1 != h2 {
		t.Fatalf("non-deterministic: %q vs %q", h1, h2)
	}
}

// TestChainLinks verifies that successive events link correctly.
func TestChainLinks(t *testing.T) {
	req1 := baseReq()
	h1, err := hashchain.Compute(hashchain.GenesisHash, req1, testTime)
	if err != nil {
		t.Fatalf("first Compute: %v", err)
	}

	req2 := baseReq()
	req2.Action = "evidence.reviewed"
	h2, err := hashchain.Compute(h1, req2, testTime.Add(time.Minute))
	if err != nil {
		t.Fatalf("second Compute: %v", err)
	}

	if h1 == h2 {
		t.Fatal("consecutive events must have distinct hashes")
	}
	if h2 == hashchain.GenesisHash {
		t.Fatal("second event hash must not be genesis hash")
	}
}

// TestVerifyValidChain verifies that Verify returns true for an intact event.
func TestVerifyValidChain(t *testing.T) {
	req := baseReq()
	occurredAt := testTime
	hash, _ := hashchain.Compute(hashchain.GenesisHash, req, occurredAt)

	event := &model.AuditEvent{
		ID:           1,
		OrgID:        req.OrgID,
		ActorEmail:   req.ActorEmail,
		Action:       req.Action,
		EventType:    req.EventType,
		ResourceType: req.ResourceType,
		ResourceID:   req.ResourceID,
		Changes:      req.Changes,
		PreviousHash: hashchain.GenesisHash,
		EventHash:    hash,
		OccurredAt:   occurredAt,
	}

	if !hashchain.Verify(event, hashchain.GenesisHash) {
		t.Fatal("Verify should return true for a correct event")
	}
}

// TestVerifyDetectsTamperedHash verifies that altering EventHash is detected.
func TestVerifyDetectsTamperedHash(t *testing.T) {
	req := baseReq()
	hash, _ := hashchain.Compute(hashchain.GenesisHash, req, testTime)

	event := &model.AuditEvent{
		OrgID:        req.OrgID,
		ActorEmail:   req.ActorEmail,
		Action:       req.Action,
		EventType:    req.EventType,
		ResourceType: req.ResourceType,
		ResourceID:   req.ResourceID,
		Changes:      req.Changes,
		PreviousHash: hashchain.GenesisHash,
		// Tamper the stored hash.
		EventHash:  hash[:63] + "x",
		OccurredAt: testTime,
	}

	if hashchain.Verify(event, hashchain.GenesisHash) {
		t.Fatal("Verify should return false when EventHash is tampered")
	}
}

// TestVerifyDetectsTamperedPreviousHash verifies that altering PreviousHash
// is detected.
func TestVerifyDetectsTamperedPreviousHash(t *testing.T) {
	req := baseReq()
	hash, _ := hashchain.Compute(hashchain.GenesisHash, req, testTime)

	event := &model.AuditEvent{
		OrgID:        req.OrgID,
		ActorEmail:   req.ActorEmail,
		Action:       req.Action,
		EventType:    req.EventType,
		ResourceType: req.ResourceType,
		ResourceID:   req.ResourceID,
		Changes:      req.Changes,
		// Wrong previous hash.
		PreviousHash: "aaaa",
		EventHash:    hash,
		OccurredAt:   testTime,
	}

	if hashchain.Verify(event, hashchain.GenesisHash) {
		t.Fatal("Verify should return false when PreviousHash is tampered")
	}
}

// TestVerifyDetectsTamperedPayload verifies that altering a payload field
// (simulating data tampering after insert) is detected.
func TestVerifyDetectsTamperedPayload(t *testing.T) {
	req := baseReq()
	hash, _ := hashchain.Compute(hashchain.GenesisHash, req, testTime)

	event := &model.AuditEvent{
		OrgID:        req.OrgID,
		ActorEmail:   req.ActorEmail,
		Action:       "TAMPERED_action", // payload modified
		EventType:    req.EventType,
		ResourceType: req.ResourceType,
		ResourceID:   req.ResourceID,
		Changes:      req.Changes,
		PreviousHash: hashchain.GenesisHash,
		EventHash:    hash,
		OccurredAt:   testTime,
	}

	if hashchain.Verify(event, hashchain.GenesisHash) {
		t.Fatal("Verify should return false when payload is tampered")
	}
}

// TestVerifyMultiEventChain verifies a 3-event chain in sequence.
func TestVerifyMultiEventChain(t *testing.T) {
	type stored struct {
		event *model.AuditEvent
		hash  string
	}

	chain := make([]stored, 0, 3)
	prevHash := hashchain.GenesisHash

	actions := []string{"evidence.uploaded", "finding.created", "engagement.status_changed"}
	for i, action := range actions {
		req := baseReq()
		req.Action = action
		occurredAt := testTime.Add(time.Duration(i) * time.Minute)

		hash, err := hashchain.Compute(prevHash, req, occurredAt)
		if err != nil {
			t.Fatalf("Compute[%d]: %v", i, err)
		}

		ev := &model.AuditEvent{
			ID:           int64(i + 1),
			OrgID:        req.OrgID,
			ActorEmail:   req.ActorEmail,
			Action:       req.Action,
			EventType:    req.EventType,
			ResourceType: req.ResourceType,
			ResourceID:   req.ResourceID,
			Changes:      req.Changes,
			PreviousHash: prevHash,
			EventHash:    hash,
			OccurredAt:   occurredAt,
		}
		chain = append(chain, stored{event: ev, hash: hash})
		prevHash = hash
	}

	// Walk the chain and verify each link.
	prev := hashchain.GenesisHash
	for i, s := range chain {
		if !hashchain.Verify(s.event, prev) {
			t.Fatalf("Verify failed at event index %d", i)
		}
		prev = s.hash
	}
}

// TestComputeDifferentTimesYieldDifferentHashes ensures that same payload at
// different timestamps produces different hashes (temporal uniqueness).
func TestComputeDifferentTimesYieldDifferentHashes(t *testing.T) {
	req := baseReq()
	h1, _ := hashchain.Compute(hashchain.GenesisHash, req, testTime)
	h2, _ := hashchain.Compute(hashchain.GenesisHash, req, testTime.Add(time.Millisecond))
	if h1 == h2 {
		t.Fatal("different timestamps must produce different hashes")
	}
}

// TestComputeChangesFieldAffectsHash verifies the changes map is in the hash.
func TestComputeChangesFieldAffectsHash(t *testing.T) {
	req1 := baseReq()
	req1.Changes = map[string]interface{}{"field": "before"}

	req2 := baseReq()
	req2.Changes = map[string]interface{}{"field": "after"}

	h1, _ := hashchain.Compute(hashchain.GenesisHash, req1, testTime)
	h2, _ := hashchain.Compute(hashchain.GenesisHash, req2, testTime)
	if h1 == h2 {
		t.Fatal("different changes payloads must produce different hashes")
	}
}

// TestComputeNilChangesIsStable ensures nil Changes marshals consistently.
func TestComputeNilChangesIsStable(t *testing.T) {
	req := baseReq()
	req.Changes = nil

	h1, err1 := hashchain.Compute(hashchain.GenesisHash, req, testTime)
	h2, err2 := hashchain.Compute(hashchain.GenesisHash, req, testTime)
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected errors: %v, %v", err1, err2)
	}
	if h1 != h2 {
		t.Fatal("nil Changes must be deterministic")
	}
	// Also verify the hash length is still correct.
	if len(h1) != 64 {
		t.Fatalf("expected 64 hex chars, got %d", len(h1))
	}
}

// TestJSONMarshalRoundTrip ensures the canonical payload is valid JSON.
func TestJSONMarshalRoundTrip(t *testing.T) {
	req := baseReq()
	hash, err := hashchain.Compute(hashchain.GenesisHash, req, testTime)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	// The hash itself should be valid hex-encoded bytes; verify length.
	if len(hash) != 64 {
		t.Fatalf("SHA-256 hex should be 64 chars, got %d", len(hash))
	}
	// Verify that an event reconstructed from JSON still round-trips.
	ev := &model.AuditEvent{
		EventHash: hash,
	}
	data, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	var ev2 model.AuditEvent
	if err := json.Unmarshal(data, &ev2); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	if ev2.EventHash != hash {
		t.Fatalf("round-trip hash mismatch")
	}
}

