package authctx

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

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

// IdentityJWTBridge copies claims from a JWT already verified by the identity
// module into the context shape used by messaging handlers. It does not
// authenticate requests by itself; use it after identity.AuthMiddleware().
func IdentityJWTBridge(cookieName string) func(http.Handler) http.Handler {
	if cookieName == "" {
		cookieName = "spur_sso"
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token == "" {
				if cookie, err := r.Cookie(cookieName); err == nil {
					token = cookie.Value
				}
			}
			claims, err := parseJWTClaims(token)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			ctx, err := contextFromClaims(r.Context(), claims)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
}

func parseJWTClaims(token string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return nil, http.ErrNoCookie
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}
	return claims, nil
}

func contextFromClaims(ctx context.Context, claims map[string]any) (context.Context, error) {
	if sub, _ := claims["sub"].(string); sub != "" {
		userID, err := uuid.Parse(sub)
		if err != nil {
			return ctx, err
		}
		ctx = context.WithValue(ctx, keyUserID, userID)
	}

	tid, _ := claims["tid"].(string)
	if tid == "" {
		tid, _ = claims["tenant_id"].(string)
	}
	tenantID, err := uuid.Parse(tid)
	if err != nil {
		return ctx, err
	}
	ctx = context.WithValue(ctx, keyTenantID, tenantID)

	roles := claimStrings(claims["roles"])
	ctx = context.WithValue(ctx, keyRoles, roles)

	permissions := claimStrings(claims["permissions"])
	if isSuperAdmin(claims) {
		permissions = append(permissions, "*")
	}
	ctx = context.WithValue(ctx, keyPermissions, permissions)
	ctx = context.WithValue(ctx, keyAuthMethod, "jwt")
	return ctx, nil
}

func claimStrings(raw any) []string {
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	values := make([]string, 0, len(items))
	for _, item := range items {
		if value, ok := item.(string); ok && value != "" {
			values = append(values, value)
		}
	}
	return values
}

func isSuperAdmin(claims map[string]any) bool {
	for _, key := range []string{"sa", "is_super_admin"} {
		if value, ok := claims[key].(bool); ok && value {
			return true
		}
	}
	return false
}
