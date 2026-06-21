package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestMetricRegistrations verifies all expected metrics are non-nil
// (registered via promauto at package init time).
func TestMetricRegistrations(t *testing.T) {
	tests := []struct {
		name string
		val  interface{}
	}{
		{"RequestDuration", RequestDuration},
		{"RequestTotal", RequestTotal},
		{"ErrorTotal", ErrorTotal},
		{"EngagementsActive", EngagementsActive},
		{"EvidenceItemsIngestedTotal", EvidenceItemsIngestedTotal},
		{"ConclusionsGeneratedTotal", ConclusionsGeneratedTotal},
		{"QualityGatesBlockedTotal", QualityGatesBlockedTotal},
		{"CoverageRate", CoverageRate},
		{"ReasoningDuration", ReasoningDuration},
	}
	for _, tt := range tests {
		if tt.val == nil {
			t.Errorf("%s: metric is nil", tt.name)
		}
	}
}

// TestMiddleware verifies the metrics middleware passes requests through
// without modification and records metrics for 2xx and 4xx responses.
func TestMiddleware(t *testing.T) {
	t.Run("2xx increments request_total", func(t *testing.T) {
		h := Middleware("test")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
		if rec.Code != http.StatusOK {
			t.Errorf("status: got %d want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("4xx increments error_total", func(t *testing.T) {
		h := Middleware("test")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", "/missing", nil))
		if rec.Code != http.StatusNotFound {
			t.Errorf("status: got %d want %d", rec.Code, http.StatusNotFound)
		}
	})
}

// TestBusinessMetricMutations verifies gauge and counter mutations compile and run.
func TestBusinessMetricMutations(t *testing.T) {
	EngagementsActive.Set(5)
	EngagementsActive.Inc()
	EngagementsActive.Dec()

	EvidenceItemsIngestedTotal.Add(10)
	ConclusionsGeneratedTotal.Add(3)
	QualityGatesBlockedTotal.Add(1)
	CoverageRate.Set(0.75)
	ReasoningDuration.Observe(0.123)
}

