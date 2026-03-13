package ociregistry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListTags(t *testing.T) {
	expectedTags := []string{"1.0.0", "1.1.0", "2.0.0", "latest"}

	srv := newMockRegistry(t, mockRegistryConfig{
		repoPath: "charts/test/myapp",
		tags:     expectedTags,
	})
	defer srv.Close()

	// Extract host from server URL (strip "https://").
	host := strings.TrimPrefix(srv.URL, "https://")

	c, err := NewClient(ClientOptions{HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	tags, err := c.ListTags(context.Background(), fmt.Sprintf("oci://%s/charts/test/myapp", host))
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}

	if len(tags) != len(expectedTags) {
		t.Fatalf("got %d tags, want %d", len(tags), len(expectedTags))
	}
	for i, tag := range tags {
		if tag != expectedTags[i] {
			t.Errorf("tag[%d] = %q, want %q", i, tag, expectedTags[i])
		}
	}
}

func TestListTagsWithAuth(t *testing.T) {
	expectedTags := []string{"1.0.0", "2.0.0"}

	srv := newMockRegistry(t, mockRegistryConfig{
		repoPath:     "charts/private/myapp",
		tags:         expectedTags,
		requireAuth:  true,
		authUsername: "user",
		authPassword: "pass",
	})
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "https://")

	c, err := NewClient(ClientOptions{Username: "user", Password: "pass", HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	tags, err := c.ListTags(context.Background(), fmt.Sprintf("oci://%s/charts/private/myapp", host))
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}

	if len(tags) != len(expectedTags) {
		t.Fatalf("got %d tags, want %d", len(tags), len(expectedTags))
	}
}

func TestListTagsAnonymousTokenExchange(t *testing.T) {
	expectedTags := []string{"1.0.0", "3.0.0"}

	srv := newMockRegistry(t, mockRegistryConfig{
		repoPath:          "charts/public/myapp",
		tags:              expectedTags,
		requireTokenExchange: true,
	})
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "https://")

	c, err := NewClient(ClientOptions{HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	tags, err := c.ListTags(context.Background(), fmt.Sprintf("oci://%s/charts/public/myapp", host))
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}

	if len(tags) != len(expectedTags) {
		t.Fatalf("got %d tags, want %d", len(tags), len(expectedTags))
	}
}

func TestGetManifestAnnotations(t *testing.T) {
	annotations := map[string]string{
		"io.giantswarm.application.values-schema": "https://example.com/schema.json",
		"org.opencontainers.image.title":          "myapp",
	}

	srv := newMockRegistry(t, mockRegistryConfig{
		repoPath:    "charts/test/myapp",
		tags:        []string{"1.0.0"},
		annotations: annotations,
	})
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "https://")

	c, err := NewClient(ClientOptions{HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	got, err := c.GetManifestAnnotations(context.Background(), fmt.Sprintf("oci://%s/charts/test/myapp", host), "1.0.0")
	if err != nil {
		t.Fatalf("GetManifestAnnotations: %v", err)
	}

	for k, v := range annotations {
		if got[k] != v {
			t.Errorf("annotation %q = %q, want %q", k, got[k], v)
		}
	}
}

func TestParseOCIURL(t *testing.T) {
	tests := []struct {
		input    string
		wantHost string
		wantPath string
		wantErr  bool
	}{
		{"oci://registry.example.com/charts/myapp", "registry.example.com", "charts/myapp", false},
		{"oci://gsoci.azurecr.io/charts/giantswarm/hello-world", "gsoci.azurecr.io", "charts/giantswarm/hello-world", false},
		{"https://example.com/foo", "", "", true},
		{"oci://registry.example.com", "", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			host, path, err := parseOCIURL(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got host=%q path=%q", host, path)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if host != tc.wantHost {
				t.Errorf("host = %q, want %q", host, tc.wantHost)
			}
			if path != tc.wantPath {
				t.Errorf("path = %q, want %q", path, tc.wantPath)
			}
		})
	}
}

// mockRegistryConfig controls the behavior of the mock registry.
type mockRegistryConfig struct {
	repoPath             string
	tags                 []string
	annotations          map[string]string
	requireAuth          bool
	authUsername          string
	authPassword         string
	requireTokenExchange bool
}

// newMockRegistry creates an httptest TLS server that mimics an OCI registry.
func newMockRegistry(t *testing.T, cfg mockRegistryConfig) *httptest.Server {
	t.Helper()

	const testToken = "test-bearer-token"

	mux := http.NewServeMux()

	// Token endpoint for bearer token exchange.
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if cfg.requireAuth {
			user, pass, ok := r.BasicAuth()
			if !ok || user != cfg.authUsername || pass != cfg.authPassword {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"token": testToken})
	})

	// checkAuth validates the bearer token if auth is required.
	checkAuth := func(w http.ResponseWriter, r *http.Request) bool {
		if !cfg.requireAuth && !cfg.requireTokenExchange {
			return true
		}
		auth := r.Header.Get("Authorization")
		if auth == "Bearer "+testToken {
			return true
		}
		// Return 401 with WWW-Authenticate challenge.
		scheme := r.URL.Scheme
		if scheme == "" {
			scheme = "https"
		}
		w.Header().Set("WWW-Authenticate", fmt.Sprintf(
			`Bearer realm="%s://%s/token",service="%s",scope="repository:%s:pull"`,
			scheme, r.Host, r.Host, cfg.repoPath))
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}

	// Tags endpoint.
	mux.HandleFunc(fmt.Sprintf("/v2/%s/tags/list", cfg.repoPath), func(w http.ResponseWriter, r *http.Request) {
		if !checkAuth(w, r) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"name": cfg.repoPath,
			"tags": cfg.tags,
		})
	})

	// Manifests endpoint.
	mux.HandleFunc(fmt.Sprintf("/v2/%s/manifests/", cfg.repoPath), func(w http.ResponseWriter, r *http.Request) {
		if !checkAuth(w, r) {
			return
		}
		annotations := cfg.annotations
		if annotations == nil {
			annotations = map[string]string{}
		}
		w.Header().Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
		json.NewEncoder(w).Encode(map[string]any{
			"schemaVersion": 2,
			"mediaType":     "application/vnd.oci.image.manifest.v1+json",
			"annotations":   annotations,
			"config": map[string]any{
				"mediaType": "application/vnd.oci.image.config.v1+json",
				"size":      0,
				"digest":    "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			},
			"layers": []any{},
		})
	})

	srv := httptest.NewTLSServer(mux)

	// Override the httpClient in the client to trust the test server's TLS cert.
	t.Cleanup(func() {
		srv.Close()
	})

	return srv
}
