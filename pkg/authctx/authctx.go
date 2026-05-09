package authctx

import (
	"context"

	"github.com/google/uuid"
)

// Context keys — must match what the identity module sets.
// These are the standard Spur context keys used across all modules.
type ctxKey string

const (
	keyTenantID    ctxKey = "tenant_id"
	keyUserID      ctxKey = "user_id"
	keyRoles       ctxKey = "roles"
	keyPermissions ctxKey = "permissions"
	keyAuthMethod  ctxKey = "auth_method" // "jwt" or "api_key"
)

// TenantID extracts the tenant UUID from context. Panics if missing.
// This is safe because the identity middleware guarantees it is set
// for all authenticated routes. If it panics, it means auth middleware
// was not applied — which is a wiring bug in app.go, not a runtime error.
func TenantID(ctx context.Context) uuid.UUID {
	v, ok := ctx.Value(keyTenantID).(uuid.UUID)
	if !ok {
		panic("authctx: tenant_id missing from context — auth middleware not applied")
	}
	return v
}

// UserID extracts the user UUID from context. Returns uuid.Nil for API key auth
// (API keys are not tied to a specific user, only to a tenant).
func UserID(ctx context.Context) uuid.UUID {
	v, _ := ctx.Value(keyUserID).(uuid.UUID)
	return v
}

// HasPermission checks if the authenticated user/key has a specific permission.
func HasPermission(ctx context.Context, permission string) bool {
	perms, ok := ctx.Value(keyPermissions).([]string)
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == permission || p == "*" { // wildcard = superadmin
			return true
		}
	}
	return false
}

// HasRole checks if the authenticated user has a specific role.
func HasRole(ctx context.Context, role string) bool {
	roles, ok := ctx.Value(keyRoles).([]string)
	if !ok {
		return false
	}
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

// AuthMethod returns "jwt" or "api_key" depending on how the request was authenticated.
func AuthMethod(ctx context.Context) string {
	v, _ := ctx.Value(keyAuthMethod).(string)
	return v
}
