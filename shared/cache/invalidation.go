package cache

import "context"

// InvalidationHandler provides event-driven cache invalidation helpers.
// Wire it up to your event bus (Redpanda, webhooks, etc.) to purge stale
// entries whenever the underlying data changes.
type InvalidationHandler struct {
	session *SessionCache
	scope   *ScopeCache
}

// NewInvalidationHandler creates an InvalidationHandler from an existing Client.
func NewInvalidationHandler(c *Client) *InvalidationHandler {
	return &InvalidationHandler{
		session: c.Session,
		scope:   c.Scope,
	}
}

// OnScopeChanged invalidates ALL cached scope DAGs for the given engagement.
// Call this whenever a scope definition (control selection, framework
// configuration, or scope metadata) is updated.
//
//	Example wiring (Redpanda consumer):
//	  if msg.Topic == "scope.changed" {
//	      handler.OnScopeChanged(ctx, event.EngagementID)
//	  }
func (h *InvalidationHandler) OnScopeChanged(ctx context.Context, engagementID string) error {
	return h.scope.Invalidate(ctx, engagementID)
}

// OnScopeVersionChanged invalidates the cache for a specific scope version.
// Use this for targeted invalidation when only one version is affected.
func (h *InvalidationHandler) OnScopeVersionChanged(ctx context.Context, engagementID string, version int) error {
	return h.scope.InvalidateVersion(ctx, engagementID, version)
}

// OnUserLogout invalidates the session cache for a user.
// Call this on explicit logout or when a JWT is revoked.
func (h *InvalidationHandler) OnUserLogout(ctx context.Context, userID string) error {
	return h.session.Delete(ctx, userID)
}

// OnUserDeactivated invalidates the session cache for a deactivated user.
// Identical effect to OnUserLogout but semantically distinct.
func (h *InvalidationHandler) OnUserDeactivated(ctx context.Context, userID string) error {
	return h.session.Delete(ctx, userID)
}

