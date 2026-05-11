package authctx

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestIdentityJWTBridgeCopiesIdentityClaims(t *testing.T) {
	token := unsignedToken(map[string]any{
		"sub":         "00000000-0000-0000-0000-000000000001",
		"tid":         "00000000-0000-0000-0000-000000000002",
		"roles":       []string{"Campaign Manager"},
		"permissions": []string{"messaging:messages:send"},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	var sawHandler bool
	IdentityJWTBridge("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawHandler = true
		if got := TenantID(r.Context()).String(); got != "00000000-0000-0000-0000-000000000002" {
			t.Fatalf("TenantID() = %s", got)
		}
		if got := UserID(r.Context()).String(); got != "00000000-0000-0000-0000-000000000001" {
			t.Fatalf("UserID() = %s", got)
		}
		if !HasRole(r.Context(), "Campaign Manager") {
			t.Fatal("expected role to be copied")
		}
		if !HasPermission(r.Context(), "messaging:messages:send") {
			t.Fatal("expected permission to be copied")
		}
		if got := AuthMethod(r.Context()); got != "jwt" {
			t.Fatalf("AuthMethod() = %s", got)
		}
	})).ServeHTTP(rec, req)

	if !sawHandler {
		t.Fatal("handler was not called")
	}
}

func TestIdentityJWTBridgeAllowsSuperAdminWildcard(t *testing.T) {
	token := unsignedToken(map[string]any{
		"sub": "00000000-0000-0000-0000-000000000001",
		"tid": "00000000-0000-0000-0000-000000000002",
		"sa":  true,
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	IdentityJWTBridge("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !HasPermission(r.Context(), "messaging:campaigns:execute") {
			t.Fatal("expected super admin wildcard permission")
		}
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestWithAPIKeySetsMessagingContext(t *testing.T) {
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	ctx := WithAPIKey(context.Background(), tenantID, []string{"messaging:messages:send"})

	if !IsAuthenticated(ctx) {
		t.Fatal("expected context to be authenticated")
	}
	if got := TenantID(ctx); got != tenantID {
		t.Fatalf("TenantID() = %s", got)
	}
	if got := UserID(ctx); got != uuid.Nil {
		t.Fatalf("UserID() = %s", got)
	}
	if !HasPermission(ctx, "messaging:messages:send") {
		t.Fatal("expected API key scope to be accepted as permission")
	}
	if got := AuthMethod(ctx); got != "api_key" {
		t.Fatalf("AuthMethod() = %s", got)
	}
}

func TestIdentityJWTBridgeSkipsAlreadyAuthenticatedContext(t *testing.T) {
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	handler := IdentityJWTBridge("spur_sso")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := TenantID(r.Context()); got != tenantID {
			t.Fatalf("TenantID() = %s", got)
		}
		if got := AuthMethod(r.Context()); got != "api_key" {
			t.Fatalf("AuthMethod() = %s", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(WithAPIKey(req.Context(), tenantID, []string{"messaging:messages:send"}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rec.Code)
	}
}

func unsignedToken(claims map[string]any) string {
	header := map[string]any{"alg": "none", "typ": "JWT"}
	return encodePart(header) + "." + encodePart(claims) + "."
}

func encodePart(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(data)
}
