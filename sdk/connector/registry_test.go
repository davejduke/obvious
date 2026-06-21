package connector_test

import (
	"context"
	"testing"

	connector "github.com/davejduke/obvious/sdk/connector"
)

// fakeConnector implements Connector for registry tests.
type fakeConnector struct{ name string }

func (f *fakeConnector) Connect(_ context.Context) error { return nil }
func (f *fakeConnector) Sync(_ context.Context, _ connector.SyncOptions) (<-chan connector.SyncRecord, error) {
	ch := make(chan connector.SyncRecord)
	close(ch)
	return ch, nil
}
func (f *fakeConnector) Transform(_ context.Context, req connector.TransformRequest) (connector.TransformResult, error) {
	return connector.TransformResult{RecordType: req.RecordType, Normalised: req.Raw}, nil
}
func (f *fakeConnector) Healthcheck(_ context.Context) connector.HealthStatus {
	return connector.HealthStatus{Healthy: true, Connector: f.name}
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := connector.NewRegistry()
	c := &fakeConnector{name: "test"}
	if err := r.Register("test", c); err != nil {
		t.Fatalf("Register: %v", err)
	}
	got, ok := r.Get("test")
	if !ok {
		t.Fatal("Get: connector not found")
	}
	if got != c {
		t.Error("Get returned wrong connector")
	}
}

func TestRegistry_DuplicateRegistration(t *testing.T) {
	r := connector.NewRegistry()
	_ = r.Register("dup", &fakeConnector{name: "dup"})
	if err := r.Register("dup", &fakeConnector{name: "dup2"}); err == nil {
		t.Fatal("expected error on duplicate registration")
	}
}

func TestRegistry_MustRegister_Panics(t *testing.T) {
	r := connector.NewRegistry()
	r.MustRegister("alpha", &fakeConnector{name: "alpha"})

	defer func() {
		if rec := recover(); rec == nil {
			t.Error("expected panic on duplicate MustRegister")
		}
	}()
	r.MustRegister("alpha", &fakeConnector{name: "alpha2"})
}

func TestRegistry_List(t *testing.T) {
	r := connector.NewRegistry()
	r.MustRegister("c", &fakeConnector{name: "c"})
	r.MustRegister("a", &fakeConnector{name: "a"})
	r.MustRegister("b", &fakeConnector{name: "b"})

	names := r.List()
	if len(names) != 3 {
		t.Fatalf("expected 3 names, got %d: %v", len(names), names)
	}
	// Should be alphabetically sorted.
	if names[0] != "a" || names[1] != "b" || names[2] != "c" {
		t.Errorf("expected [a b c], got %v", names)
	}
}

func TestRegistry_Unregister(t *testing.T) {
	r := connector.NewRegistry()
	r.MustRegister("x", &fakeConnector{name: "x"})
	r.Unregister("x")
	if _, ok := r.Get("x"); ok {
		t.Error("expected connector to be unregistered")
	}
	if r.Len() != 0 {
		t.Errorf("expected Len=0, got %d", r.Len())
	}
}
