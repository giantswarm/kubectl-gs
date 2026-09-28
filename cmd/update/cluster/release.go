package cluster

import (
	"context"
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/giantswarm/microerror"
	releasev1alpha1 "github.com/giantswarm/releases/sdk/api/v1alpha1"
	"go.yaml.in/yaml/v3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// releaseVersionPath is where the cluster chart's values carry the release version.
var releaseVersionPath = []string{"global", "release", "version"}

// infrastructureKindProviders maps a Cluster's infrastructureRef kind to the
// provider prefix of its Release resources (aws-34.5.0, eks-34.5.0, …).
var infrastructureKindProviders = map[string]releasev1alpha1.Provider{
	"AWSCluster":             releasev1alpha1.ProviderAws,
	"AWSManagedCluster":      releasev1alpha1.ProviderEKS,
	"AzureCluster":           releasev1alpha1.ProviderAzure,
	"AzureManagedCluster":    releasev1alpha1.ProviderAKS,
	"AzureASOManagedCluster": releasev1alpha1.ProviderAKS,
	"VSphereCluster":         releasev1alpha1.ProviderVsphere,
	"VCDCluster":             releasev1alpha1.ProviderCloudDirector,
	"ProxmoxCluster":         releasev1alpha1.ProviderProxmox,
}

// releaseProvider returns the Release provider of a Cluster API cluster, from
// the kind of its infrastructure reference.
func releaseProvider(capiCluster *unstructured.Unstructured) (releasev1alpha1.Provider, error) {
	kind, _, _ := unstructured.NestedString(capiCluster.Object, "spec", "infrastructureRef", "kind")
	provider, ok := infrastructureKindProviders[kind]
	if !ok {
		return "", microerror.Maskf(notAllowedError, "Cannot tell the release provider of cluster '%s' from its infrastructure kind %q.", capiCluster.GetName(), kind)
	}
	return provider, nil
}

// ensureRelease fails unless the management cluster has the Release resource
// for the provider and version: without it the cluster chart stops rendering.
func ensureRelease(ctx context.Context, c client.Client, provider releasev1alpha1.Provider, version string) error {
	name := fmt.Sprintf("%s-%s", provider, strings.TrimPrefix(version, "v"))
	err := c.Get(ctx, client.ObjectKey{Name: name}, &releasev1alpha1.Release{})
	if apierrors.IsNotFound(err) {
		return microerror.Maskf(notFoundError, "Release '%s' does not exist on this management cluster. List the available releases with 'kubectl gs get releases'.", name)
	} else if err != nil {
		return microerror.Mask(err)
	}
	return nil
}

// ensureNewer fails unless target is a higher release version than current.
func ensureNewer(current, target string) error {
	currentVersion, err := semver.NewVersion(current)
	if err != nil {
		return microerror.Maskf(invalidConfigError, "Current release version %q is not a semantic version.", current)
	}
	targetVersion, err := semver.NewVersion(target)
	if err != nil {
		return microerror.Maskf(invalidFlagError, "--%s %q is not a semantic version.", flagReleaseVersion, target)
	}
	if !targetVersion.GreaterThan(currentVersion) {
		return microerror.Maskf(notAllowedError, "Release version '%s' is not higher than the cluster's current release version '%s'.", target, current)
	}
	return nil
}

// setReleaseVersion returns the release version the values carry at
// global.release.version and the values with it replaced by target. Only that
// one scalar changes; the rest of the document is kept byte for byte.
func setReleaseVersion(values, target string) (string, string, error) {
	var doc yaml.Node
	err := yaml.Unmarshal([]byte(values), &doc)
	if err != nil {
		return "", "", microerror.Maskf(invalidConfigError, "Cluster values are not valid YAML: %s", err)
	}

	node := lookup(&doc, releaseVersionPath...)
	if node == nil || node.Kind != yaml.ScalarNode || node.Value == "" {
		return "", "", microerror.Maskf(notFoundError, "Cluster values carry no %s.", strings.Join(releaseVersionPath, "."))
	}

	replacement := target
	width := len(node.Value)
	switch node.Style {
	case yaml.DoubleQuotedStyle:
		replacement, width = `"`+target+`"`, width+2
	case yaml.SingleQuotedStyle:
		replacement, width = `'`+target+`'`, width+2
	}

	lines := strings.SplitAfter(values, "\n")
	line := lines[node.Line-1]
	start := node.Column - 1
	lines[node.Line-1] = line[:start] + replacement + line[start+width:]

	return node.Value, strings.Join(lines, ""), nil
}

// lookup returns the value node at the mapping path, or nil.
func lookup(node *yaml.Node, path ...string) *yaml.Node {
	if node.Kind == yaml.DocumentNode && len(node.Content) == 1 {
		node = node.Content[0]
	}
	if len(path) == 0 {
		return node
	}
	if node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == path[0] {
			return lookup(node.Content[i+1], path[1:]...)
		}
	}
	return nil
}
