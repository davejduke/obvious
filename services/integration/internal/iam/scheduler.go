package iam

import (
	"context"
	"sync"
	"time"
)

// SyncResult holds the outcome of a single scheduled sync cycle.
type SyncResult struct {
	Provider string
	Snapshot *IAMSnapshot
	Evidence []EvidenceItem
	Error    error
	SyncedAt time.Time
}

// SyncCallback is invoked after each sync cycle (success or failure).
type SyncCallback func(result SyncResult)

// SchedulerConfig holds tuning parameters for the sync scheduler.
type SchedulerConfig struct {
	// Interval controls how often each connector is synced.
	Interval time.Duration
	// OnSync is called after each sync (may be nil).
	OnSync SyncCallback
}

// DefaultSchedulerConfig returns safe defaults: sync every 15 minutes.
func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		Interval: 15 * time.Minute,
	}
}

// Scheduler runs periodic syncs for all IAM connectors in the registry.
// It mirrors the rate-limiting / circuit-breaker reuse requirement from
// Phase 1: circuit breakers are applied at the adapter level; the scheduler
// just drives Sync() calls on whatever connectors are registered.
type Scheduler struct {
	registry *IAMRegistry
	config   SchedulerConfig
	mu       sync.Mutex
	results  map[string]SyncResult
	cancel   context.CancelFunc
}

// NewScheduler creates a Scheduler backed by the given registry.
func NewScheduler(reg *IAMRegistry, cfg SchedulerConfig) *Scheduler {
	return &Scheduler{
		registry: reg,
		config:   cfg,
		results:  make(map[string]SyncResult),
	}
}

// Start begins background sync loops for every registered connector.
// Lifecycle is controlled via ctx; call Stop() to halt gracefully.
func (s *Scheduler) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.cancel = cancel
	s.mu.Unlock()

	for _, name := range s.registry.List() {
		go s.runLoop(ctx, name)
	}
}

// Stop halts all running sync loops.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
}

// LastResult returns the most recent sync result for a named connector.
func (s *Scheduler) LastResult(name string) (SyncResult, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.results[name]
	return r, ok
}

// AllResults returns the most recent sync result for every connector.
func (s *Scheduler) AllResults() map[string]SyncResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]SyncResult, len(s.results))
	for k, v := range s.results {
		out[k] = v
	}
	return out
}

// runLoop syncs a single connector on every tick until ctx is cancelled.
// It performs one immediate sync, then waits for the configured interval.
func (s *Scheduler) runLoop(ctx context.Context, name string) {
	s.sync(ctx, name)
	ticker := time.NewTicker(s.config.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sync(ctx, name)
		}
	}
}

// sync runs one sync cycle for the named connector and stores the result.
func (s *Scheduler) sync(ctx context.Context, name string) {
	conn, ok := s.registry.Get(name)
	if !ok {
		return
	}

	snap, err := conn.Sync(ctx)
	result := SyncResult{
		Provider: name,
		Error:    err,
		SyncedAt: time.Now().UTC(),
	}
	if err == nil && snap != nil {
		result.Snapshot = snap
		result.Evidence = MapSnapshotToEvidence(snap)
	}

	s.mu.Lock()
	s.results[name] = result
	s.mu.Unlock()

	if s.config.OnSync != nil {
		s.config.OnSync(result)
	}
}
