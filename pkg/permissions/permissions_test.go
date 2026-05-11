package permissions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCatalogHasUniqueStableKeys(t *testing.T) {
	seen := make(map[string]struct{}, len(Catalog))
	for _, permission := range Catalog {
		if permission.Key == "" {
			t.Fatal("permission key must not be empty")
		}
		if permission.Description == "" {
			t.Fatalf("permission %q must have a description", permission.Key)
		}
		if _, ok := seen[permission.Key]; ok {
			t.Fatalf("duplicate permission key %q", permission.Key)
		}
		seen[permission.Key] = struct{}{}
	}

	for _, key := range Keys() {
		if _, ok := seen[key]; !ok {
			t.Fatalf("Keys returned unknown permission %q", key)
		}
	}
}

func TestRoleTemplatesReferenceCatalogPermissions(t *testing.T) {
	catalog := catalogKeys()
	seenCodes := make(map[string]struct{}, len(RoleTemplates))
	for _, template := range RoleTemplates {
		if template.Code == "" {
			t.Fatal("role template code must not be empty")
		}
		if template.Name == "" {
			t.Fatalf("role template %q must have a name", template.Code)
		}
		if _, ok := seenCodes[template.Code]; ok {
			t.Fatalf("duplicate role template code %q", template.Code)
		}
		seenCodes[template.Code] = struct{}{}
		if len(template.Permissions) == 0 {
			t.Fatalf("role template %q must include permissions", template.Code)
		}
		for _, permission := range template.Permissions {
			if _, ok := catalog[permission]; !ok {
				t.Fatalf("role template %q references unknown permission %q", template.Code, permission)
			}
		}
	}

	if !reflect.DeepEqual(RoleTemplates[0].Permissions, Keys()) {
		t.Fatalf("first role template must be the admin template with the complete catalog")
	}
}

func TestSpurManifestMatchesPermissionCatalogAndRoleTemplates(t *testing.T) {
	manifest := readSpurManifest(t)

	manifestPermissions := make([]Permission, 0, len(manifest.Permissions))
	for _, permission := range manifest.Permissions {
		manifestPermissions = append(manifestPermissions, Permission{
			Key:         permission.Key,
			Description: permission.Description,
		})
	}
	if !reflect.DeepEqual(manifestPermissions, Catalog) {
		t.Fatal("spur.json permissions must match pkg/permissions.Catalog")
	}

	if !reflect.DeepEqual(manifest.RoleTemplates, RoleTemplates) {
		t.Fatal("spur.json role_templates must match pkg/permissions.RoleTemplates")
	}
}

func catalogKeys() map[string]struct{} {
	keys := make(map[string]struct{}, len(Catalog))
	for _, permission := range Catalog {
		keys[permission.Key] = struct{}{}
	}
	return keys
}

func readSpurManifest(t *testing.T) struct {
	Permissions   []Permission   `json:"permissions"`
	RoleTemplates []RoleTemplate `json:"role_templates"`
} {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "spur.json"))
	if err != nil {
		t.Fatalf("read spur.json: %v", err)
	}
	var manifest struct {
		Permissions   []Permission   `json:"permissions"`
		RoleTemplates []RoleTemplate `json:"role_templates"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse spur.json: %v", err)
	}
	return manifest
}
