package ociregistry

import (
	"context"
	"fmt"
	"strings"

	"github.com/regclient/regclient"
	"github.com/regclient/regclient/config"
	"github.com/regclient/regclient/types/manifest"
	"github.com/regclient/regclient/types/ref"
)

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
}

type client struct {
	username string
	password string
	// hostOverrides is used in tests to configure TLS settings for specific hosts.
	hostOverrides map[string]config.Host
}

// NewClient creates an OCI registry client with the given authentication options.
// Credential resolution order: explicit username/password > Docker config > anonymous.
func NewClient(opts ClientOptions) (Client, error) {
	return &client{
		username: opts.Username,
		password: opts.Password,
	}, nil
}

// ListTags returns all tags for the repository at the given OCI URL.
// Handles pagination automatically via regclient.
func (c *client) ListTags(ctx context.Context, ociURL string) ([]string, error) {
	registryHost, repoPath, err := parseOCIURL(ociURL)
	if err != nil {
		return nil, err
	}

	rc := c.newRegClient(registryHost)
	defer rc.Close(ctx, ref.Ref{})

	r, err := ref.New(registryHost + "/" + repoPath)
	if err != nil {
		return nil, fmt.Errorf("parsing reference: %w", err)
	}

	tl, err := rc.TagList(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("listing tags: %w", err)
	}

	tags, err := tl.GetTags()
	if err != nil {
		return nil, fmt.Errorf("reading tags: %w", err)
	}

	return tags, nil
}

// GetManifestAnnotations returns the annotations from the manifest for the
// given OCI URL and tag.
func (c *client) GetManifestAnnotations(ctx context.Context, ociURL string, tag string) (map[string]string, error) {
	if tag == "" || strings.ContainsAny(tag, "/:@") {
		return nil, fmt.Errorf("invalid tag %q", tag)
	}

	registryHost, repoPath, err := parseOCIURL(ociURL)
	if err != nil {
		return nil, err
	}

	rc := c.newRegClient(registryHost)
	defer rc.Close(ctx, ref.Ref{})

	r, err := ref.New(registryHost + "/" + repoPath + ":" + tag)
	if err != nil {
		return nil, fmt.Errorf("parsing reference: %w", err)
	}

	m, err := rc.ManifestGet(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("fetching manifest for tag %q: %w", tag, err)
	}

	annotator, ok := m.(manifest.Annotator)
	if !ok {
		return map[string]string{}, nil
	}

	annot, err := annotator.GetAnnotations()
	if err != nil {
		return nil, fmt.Errorf("reading annotations for tag %q: %w", tag, err)
	}

	return annot, nil
}

// newRegClient creates a regclient instance configured for the given registry host.
func (c *client) newRegClient(registryHost string) *regclient.RegClient {
	opts := []regclient.Opt{
		regclient.WithDockerCreds(),
	}

	hostCfg := config.HostNewName(registryHost)

	// Apply test overrides if present.
	if override, ok := c.hostOverrides[registryHost]; ok {
		hostCfg.TLS = override.TLS
	}

	if c.username != "" && c.password != "" {
		hostCfg.User = c.username
		hostCfg.Pass = c.password
	}

	opts = append(opts, regclient.WithConfigHost(*hostCfg))

	return regclient.New(opts...)
}

// parseOCIURL parses an OCI URL like "oci://registry.example.com/path/to/repo"
// and returns the registry host and repository path separately.
func parseOCIURL(ociURL string) (registryHost, repoPath string, err error) {
	trimmed := strings.TrimPrefix(ociURL, "oci://")
	if trimmed == ociURL {
		return "", "", fmt.Errorf("OCI URL must start with oci://, got %q", ociURL)
	}

	registryHost, repoPath, ok := strings.Cut(trimmed, "/")
	if !ok || registryHost == "" || repoPath == "" {
		return "", "", fmt.Errorf("invalid OCI URL %q", ociURL)
	}

	return registryHost, repoPath, nil
}
