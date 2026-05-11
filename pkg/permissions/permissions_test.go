package permissions

import "testing"

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
