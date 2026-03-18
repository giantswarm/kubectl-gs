package ociregistry

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/regclient/regclient/config"
)

func TestListTags(t *testing.T) {
	expectedTags := []string{"1.0.0", "1.1.0", "2.0.0", "latest"}

	srv := newMockRegistry(t, mockRegistryConfig{
		repoPath: "charts/test/myapp",
		tags:     expectedTags,
	})
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "http://")
	c := newClientForTest(ClientOptions{}, host)

	tags, err := c.ListTags(context.Background(), host, "charts/test/myapp")
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

	host := strings.TrimPrefix(srv.URL, "http://")
	c := newClientForTest(ClientOptions{Username: "user", Password: "pass"}, host)

	tags, err := c.ListTags(context.Background(), host, "charts/private/myapp")
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
		repoPath:             "charts/public/myapp",
		tags:                 expectedTags,
		requireTokenExchange: true,
	})
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "http://")
	c := newClientForTest(ClientOptions{}, host)

	tags, err := c.ListTags(context.Background(), host, "charts/public/myapp")
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

	host := strings.TrimPrefix(srv.URL, "http://")
	c := newClientForTest(ClientOptions{}, host)

	got, err := c.GetManifestAnnotations(context.Background(), host, "charts/test/myapp", "1.0.0")
	if err != nil {
		t.Fatalf("GetManifestAnnotations: %v", err)
	}

	for k, v := range annotations {
		if got[k] != v {
			t.Errorf("annotation %q = %q, want %q", k, got[k], v)
		}
	}
}

func TestGetManifestAnnotationsInvalidTag(t *testing.T) {
	c, err := NewClient(ClientOptions{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.GetManifestAnnotations(context.Background(), "example.com", "repo", "../evil")
	if err == nil {
		t.Fatal("expected error for invalid tag")
	}
}

func TestTagExists(t *testing.T) {
	srv := newMockRegistry(t, mockRegistryConfig{
		repoPath: "charts/test/myapp",
		tags:     []string{"1.0.0"},
	})
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "http://")
	c := newClientForTest(ClientOptions{}, host)

	exists, err := c.TagExists(context.Background(), host, "charts/test/myapp", "1.0.0")
	if err != nil {
		t.Fatalf("TagExists: %v", err)
	}
	if !exists {
		t.Error("expected tag to exist")
	}

	exists, err = c.TagExists(context.Background(), host, "charts/test/myapp", "99.99.99")
	if err != nil {
		t.Fatalf("TagExists: %v", err)
	}
	if exists {
		t.Error("expected tag to not exist")
	}
}

// newClientForTest creates a client configured for testing with a plain HTTP mock server.
func newClientForTest(opts ClientOptions, host string) Client {
	return &client{
		username: opts.Username,
		password: opts.Password,
		hostOverrides: map[string]config.Host{
			host: {TLS: config.TLSDisabled},
		},
	}
}

// mockRegistryConfig controls the behavior of the mock registry.
type mockRegistryConfig struct {
	repoPath             string
	tags                 []string
	annotations          map[string]string
	requireAuth          bool
	authUsername         string
	authPassword         string
	requireTokenExchange bool
}

// newMockRegistry creates a plain HTTP httptest server that mimics an OCI registry.
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
		w.Header().Set("WWW-Authenticate", fmt.Sprintf(
			`Bearer realm="http://%s/token",service="%s",scope="repository:%s:pull"`,
			r.Host, r.Host, cfg.repoPath))
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
	manifestPrefix := fmt.Sprintf("/v2/%s/manifests/", cfg.repoPath)
	mux.HandleFunc(manifestPrefix, func(w http.ResponseWriter, r *http.Request) {
		if !checkAuth(w, r) {
			return
		}
		// Check that the requested tag exists.
		tag := strings.TrimPrefix(r.URL.Path, manifestPrefix)
		found := false
		for _, t := range cfg.tags {
			if t == tag {
				found = true
				break
			}
		}
		if !found {
			http.Error(w, fmt.Sprintf("tag %q not found", tag), http.StatusNotFound)
			return
		}
		annotations := cfg.annotations
		if annotations == nil {
			annotations = map[string]string{}
		}
		manifestBody := map[string]any{
			"schemaVersion": 2,
			"mediaType":     "application/vnd.oci.image.manifest.v1+json",
			"annotations":   annotations,
			"config": map[string]any{
				"mediaType": "application/vnd.oci.image.config.v1+json",
				"size":      0,
				"digest":    "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			},
			"layers": []any{},
		}
		body, _ := json.Marshal(manifestBody)
		digest := fmt.Sprintf("sha256:%x", sha256.Sum256(body))
		w.Header().Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
		w.Header().Set("Docker-Content-Digest", digest)
		w.Write(body)
	})

	return httptest.NewServer(mux)
}
