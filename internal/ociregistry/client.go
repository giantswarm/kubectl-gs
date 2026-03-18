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
	// ListTags returns all tags for the given registry and repository.
	ListTags(ctx context.Context, registry, repository string) ([]string, error)
	// TagExists checks whether a specific tag exists in the repository.
	// This is faster than ListTags when only validating a single tag.
	TagExists(ctx context.Context, registry, repository, tag string) (bool, error)
	// GetManifestAnnotations returns the annotations from the manifest
	// for the given registry, repository, and tag.
	GetManifestAnnotations(ctx context.Context, registry, repository, tag string) (map[string]string, error)
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

// ListTags returns all tags for the repository.
// Handles pagination automatically via regclient.
func (c *client) ListTags(ctx context.Context, registry, repository string) ([]string, error) {
	rc := c.newRegClient(registry)
	defer rc.Close(ctx, ref.Ref{})

	r, err := ref.New(registry + "/" + repository)
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

// TagExists checks whether a specific tag exists by performing a HEAD request
// for its manifest. This is faster than listing all tags.
func (c *client) TagExists(ctx context.Context, registry, repository, tag string) (bool, error) {
	if tag == "" || strings.ContainsAny(tag, "/:@") {
		return false, fmt.Errorf("invalid tag %q", tag)
	}

	rc := c.newRegClient(registry)
	defer rc.Close(ctx, ref.Ref{})

	r, err := ref.New(registry + "/" + repository + ":" + tag)
	if err != nil {
		return false, fmt.Errorf("parsing reference: %w", err)
	}

	_, err = rc.ManifestHead(ctx, r)
	if err != nil {
		return false, nil
	}

	return true, nil
}

// GetManifestAnnotations returns the annotations from the manifest for the
// given registry, repository, and tag.
func (c *client) GetManifestAnnotations(ctx context.Context, registry, repository, tag string) (map[string]string, error) {
	if tag == "" || strings.ContainsAny(tag, "/:@") {
		return nil, fmt.Errorf("invalid tag %q", tag)
	}

	rc := c.newRegClient(registry)
	defer rc.Close(ctx, ref.Ref{})

	r, err := ref.New(registry + "/" + repository + ":" + tag)
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
