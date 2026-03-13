package ociregistry

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/registry/remote/credentials"
)

const defaultHTTPTimeout = 30 * time.Second

// acceptHeader is the Accept header value for OCI registry requests.
var acceptHeader = strings.Join([]string{
	"application/vnd.oci.image.manifest.v1+json",
	"application/vnd.docker.distribution.manifest.v2+json",
	"application/json",
}, ", ")

// tagRe validates that a tag contains only allowed characters.
var tagRe = regexp.MustCompile(`^[a-zA-Z0-9_.\-]+$`)

// Client provides read-only access to an OCI registry for listing tags
// and reading manifest annotations.
type Client interface {
	// ListTags returns all tags for the given OCI URL.
	ListTags(ctx context.Context, ociURL string) ([]string, error)
	// GetManifestAnnotations returns the annotations from the manifest
	// for the given OCI URL and tag.
	GetManifestAnnotations(ctx context.Context, ociURL string, tag string) (map[string]string, error)
}

// ClientOptions configures how the OCI client authenticates.
type ClientOptions struct {
	Username string
	Password string
	// HTTPClient overrides the default HTTP client. Useful for testing.
	HTTPClient *http.Client
}

type client struct {
	httpClient *http.Client
	username   string
	password   string
	credStore  credentials.Store
}

// NewClient creates an OCI registry client with the given authentication options.
// Credential resolution order: explicit username/password > Docker config > anonymous.
func NewClient(opts ClientOptions) (Client, error) {
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}
	c := &client{
		httpClient: httpClient,
	}

	if opts.Username != "" && opts.Password != "" {
		c.username = opts.Username
		c.password = opts.Password
		return c, nil
	}

	// Try Docker config for credentials.
	store, err := credentials.NewStoreFromDocker(credentials.StoreOptions{})
	if err == nil {
		c.credStore = store
	}

	return c, nil
}

// ListTags returns all tags for the repository at the given OCI URL.
func (c *client) ListTags(ctx context.Context, ociURL string) ([]string, error) {
	registryHost, repoPath, err := parseOCIURL(ociURL)
	if err != nil {
		return nil, err
	}

	tagsURL := fmt.Sprintf("https://%s/v2/%s/tags/list", registryHost, repoPath)

	resp, err := c.doRegistryRequest(ctx, registryHost, repoPath, tagsURL)
	if err != nil {
		return nil, fmt.Errorf("listing tags: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("listing tags: %s: %s", resp.Status, string(body))
	}

	var result struct {
		Tags []string `json:"tags"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding tags response: %w", err)
	}

	return result.Tags, nil
}

// GetManifestAnnotations returns the annotations from the manifest for the
// given OCI URL and tag.
func (c *client) GetManifestAnnotations(ctx context.Context, ociURL string, tag string) (map[string]string, error) {
	if !tagRe.MatchString(tag) {
		return nil, fmt.Errorf("invalid tag %q: must contain only alphanumeric characters, dots, hyphens, and underscores", tag)
	}

	registryHost, repoPath, err := parseOCIURL(ociURL)
	if err != nil {
		return nil, err
	}

	manifestURL := fmt.Sprintf("https://%s/v2/%s/manifests/%s", registryHost, repoPath, tag)

	resp, err := c.doRegistryRequest(ctx, registryHost, repoPath, manifestURL)
	if err != nil {
		return nil, fmt.Errorf("fetching manifest for tag %q: %w", tag, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fetching manifest for tag %q: %s: %s", tag, resp.Status, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading manifest for tag %q: %w", tag, err)
	}

	var manifest ocispec.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parsing manifest for tag %q: %w", tag, err)
	}

	return manifest.Annotations, nil
}

// doRegistryRequest performs an HTTP GET with registry authentication.
// It handles the WWW-Authenticate challenge flow:
// 1. Try the request directly (may work for public registries without auth).
// 2. On 401, parse the WWW-Authenticate header and obtain a bearer token.
// 3. Retry with the bearer token.
func (c *client) doRegistryRequest(ctx context.Context, registryHost, repoPath, targetURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", acceptHeader)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}
	resp.Body.Close()

	// Parse WWW-Authenticate and get a token.
	token, err := c.obtainToken(ctx, resp.Header.Get("WWW-Authenticate"), registryHost, repoPath)
	if err != nil {
		return nil, fmt.Errorf("obtaining auth token: %w", err)
	}

	req, err = http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", acceptHeader)
	req.Header.Set("Authorization", "Bearer "+token)

	return c.httpClient.Do(req)
}

// obtainToken handles the OAuth2/Bearer token exchange.
// It uses GET for token requests (compatible with ACR anonymous access).
func (c *client) obtainToken(ctx context.Context, wwwAuth, registryHost, repoPath string) (string, error) {
	realm, service, scope := parseWWWAuthenticate(wwwAuth)
	if realm == "" {
		return "", fmt.Errorf("missing realm in WWW-Authenticate header: %s", wwwAuth)
	}

	// If no scope was provided in the challenge, construct one for pull access.
	if scope == "" {
		scope = fmt.Sprintf("repository:%s:pull", repoPath)
	}
	if service == "" {
		service = registryHost
	}

	// Build token request URL.
	tokenURL, err := url.Parse(realm)
	if err != nil {
		return "", fmt.Errorf("parsing realm URL %q: %w", realm, err)
	}
	q := tokenURL.Query()
	q.Set("service", service)
	q.Set("scope", scope)
	tokenURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL.String(), nil)
	if err != nil {
		return "", err
	}

	// Add basic auth if credentials are available.
	username, password := c.credentialsForHost(ctx, registryHost)
	if username != "" && password != "" {
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString(
			[]byte(username+":"+password)))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token request failed: %s: %s", resp.Status, string(body))
	}

	var tokenResp struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("decoding token response: %w", err)
	}

	// Some registries use "token", others use "access_token".
	if tokenResp.AccessToken != "" {
		return tokenResp.AccessToken, nil
	}
	return tokenResp.Token, nil
}

// credentialsForHost returns username/password for the given registry host.
// It checks explicit credentials first, then the Docker credential store.
func (c *client) credentialsForHost(ctx context.Context, registryHost string) (string, string) {
	if c.username != "" && c.password != "" {
		return c.username, c.password
	}

	if c.credStore != nil {
		cred, err := c.credStore.Get(ctx, registryHost)
		if err == nil && cred.Username != "" {
			return cred.Username, cred.Password
		}
	}

	return "", ""
}

// parseWWWAuthenticate extracts realm, service, and scope from a
// WWW-Authenticate header value like:
// Bearer realm="https://example.com/token",service="example.com",scope="repository:foo:pull"
// It handles commas inside quoted values correctly.
func parseWWWAuthenticate(header string) (realm, service, scope string) {
	header = strings.TrimPrefix(header, "Bearer ")

	// Parse respecting quoted strings — commas inside quotes are not delimiters.
	for len(header) > 0 {
		header = strings.TrimSpace(header)
		if header == "" {
			break
		}

		k, rest, ok := strings.Cut(header, "=")
		if !ok {
			break
		}
		k = strings.TrimSpace(k)

		var v string
		if len(rest) > 0 && rest[0] == '"' {
			// Find closing quote.
			end := strings.Index(rest[1:], `"`)
			if end < 0 {
				v = rest[1:]
				header = ""
			} else {
				v = rest[1 : end+1]
				header = rest[end+2:]
			}
		} else {
			// Unquoted value — delimited by comma.
			end := strings.Index(rest, ",")
			if end < 0 {
				v = rest
				header = ""
			} else {
				v = rest[:end]
				header = rest[end:]
			}
		}

		// Consume leading comma separator.
		header = strings.TrimPrefix(strings.TrimSpace(header), ",")

		switch k {
		case "realm":
			realm = v
		case "service":
			service = v
		case "scope":
			scope = v
		}
	}
	return
}

// parseOCIURL parses an OCI URL like "oci://registry.example.com/path/to/repo"
// and returns the registry host and repository path separately.
func parseOCIURL(ociURL string) (registryHost, repoPath string, err error) {
	ref := strings.TrimPrefix(ociURL, "oci://")
	if ref == ociURL {
		return "", "", fmt.Errorf("OCI URL must start with oci://, got %q", ociURL)
	}

	registryHost, repoPath, ok := strings.Cut(ref, "/")
	if !ok || registryHost == "" || repoPath == "" {
		return "", "", fmt.Errorf("invalid OCI URL %q", ociURL)
	}

	return registryHost, repoPath, nil
}
