package authctx

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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
