package deploychart

import (
	"strings"
	"testing"
)

func TestDiffManifests_NoDiff(t *testing.T) {
	manifest := []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  namespace: default
data:
  key: value
`)
	diff, err := DiffManifests("test", manifest, manifest)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff != "" {
		t.Errorf("expected empty diff, got:\n%s", diff)
	}
}

func TestDiffManifests_WithDiff(t *testing.T) {
	existing := []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  namespace: default
data:
  key: old-value
`)
	desired := []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  namespace: default
data:
  key: new-value
`)
	diff, err := DiffManifests("test", existing, desired)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff == "" {
		t.Fatal("expected non-empty diff")
	}
	if !strings.Contains(diff, "-  key: old-value") {
		t.Errorf("diff should contain removed line, got:\n%s", diff)
	}
	if !strings.Contains(diff, "+  key: new-value") {
		t.Errorf("diff should contain added line, got:\n%s", diff)
	}
}

func TestDiffManifests_StripsServerFields(t *testing.T) {
	existing := []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  namespace: default
  uid: abc-123
  resourceVersion: "12345"
  creationTimestamp: "2024-01-01T00:00:00Z"
  generation: 1
  managedFields:
  - manager: kubectl
    operation: Apply
status: {}
data:
  key: value
`)
	desired := []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  namespace: default
data:
  key: value
`)
	diff, err := DiffManifests("test", existing, desired)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff != "" {
		t.Errorf("expected empty diff after stripping server fields, got:\n%s", diff)
	}
}
