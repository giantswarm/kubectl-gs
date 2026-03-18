package chart

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/v6/internal/deploychart"
	"github.com/giantswarm/kubectl-gs/v6/internal/key"
	"github.com/giantswarm/kubectl-gs/v6/internal/ociregistry"
	"github.com/giantswarm/kubectl-gs/v6/pkg/commonconfig"
)

type runner struct {
	commonConfig *commonconfig.CommonConfig
	fileSystem   afero.Fs
	flag         *flag
	logger       micrologger.Logger
	stdout       io.Writer
	stderr       io.Writer
}

func (r *runner) Run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	err := r.flag.Validate()
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.run(ctx, cmd, args)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *runner) run(ctx context.Context, cmd *cobra.Command, args []string) error {
	resourceName := r.flag.Name
	if resourceName == "" {
		resourceName = fmt.Sprintf("%s-%s", r.flag.Cluster, r.flag.ChartName)
	}

	namespace := key.OrganizationNamespaceFromName(r.flag.Organization)

	// Split the OCI URL prefix into registry host and repository prefix.
	// OCIURLPrefix is normalized to "oci://host/path/" by flag.Validate().
	registry, repoPath := splitOCIURLPrefix(r.flag.OCIURLPrefix, r.flag.ChartName)

	// The full OCI URL is needed for the OCIRepository manifest.
	ociURL := r.flag.OCIURLPrefix + r.flag.ChartName

	// Resolve registry password if username is provided.
	registryPassword, err := r.resolveRegistryPassword()
	if err != nil {
		return microerror.Mask(err)
	}

	// Contact the registry to validate and optionally resolve the version.
	version := r.flag.Version

	ociClient, err := ociregistry.NewClient(ociregistry.ClientOptions{
		Username: r.flag.RegistryUsername,
		Password: registryPassword,
	})
	if err != nil {
		return microerror.Mask(err)
	}

	if version != "" {
		// Validate that the specified version exists (single HEAD request).
		exists, err := ociClient.TagExists(ctx, registry, repoPath, version)
		if err != nil {
			return microerror.Mask(err)
		}
		if !exists {
			return fmt.Errorf("version %q not found in %s", version, ociURL)
		}
	} else {
		// List tags to validate the repository and resolve version.
		tags, err := ociClient.ListTags(ctx, registry, repoPath)
		if err != nil {
			return microerror.Mask(err)
		}

		if r.flag.AutoUpgrade != "all" {
			version, err = ociregistry.LatestSemverTag(tags)
			if err != nil {
				return fmt.Errorf("resolving latest version from %s: %w", ociURL, err)
			}

			// Strip "v" prefix — OCIRepository ref.tag should use bare semver.
			version = strings.TrimPrefix(version, "v")

			fmt.Fprintf(r.stderr, "Resolved latest version: %s\n", version)
		}
	}

	// Read values file if provided.
	var values map[string]any
	if r.flag.ValuesFile != "" {
		data, err := afero.ReadFile(r.fileSystem, r.flag.ValuesFile)
		if err != nil {
			return microerror.Mask(err)
		}
		err = yaml.Unmarshal(data, &values)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	ociRepoOpts := deploychart.OCIRepositoryOptions{
		Name:        resourceName,
		Namespace:   namespace,
		ClusterName: r.flag.Cluster,
		URL:         ociURL,
		Version:     version,
		AutoUpgrade: r.flag.AutoUpgrade,
		Interval:    r.flag.Interval,
	}

	helmReleaseOpts := deploychart.HelmReleaseOptions{
		Name:            resourceName,
		Namespace:       namespace,
		ClusterName:     r.flag.Cluster,
		ChartName:       r.flag.ChartName,
		TargetNamespace: r.flag.TargetNS,
		Interval:        r.flag.Interval,
		Values:          values,
	}

	ociRepo := deploychart.BuildOCIRepository(ociRepoOpts)
	helmRelease := deploychart.BuildHelmRelease(helmReleaseOpts)

	ociRepoYAML, err := deploychart.MarshalManifest(ociRepo)
	if err != nil {
		return microerror.Mask(err)
	}

	helmReleaseYAML, err := deploychart.MarshalManifest(helmRelease)
	if err != nil {
		return microerror.Mask(err)
	}

	fmt.Fprintf(r.stdout, "%s---\n%s", string(ociRepoYAML), string(helmReleaseYAML))

	return nil
}

// resolveRegistryPassword returns the registry password when a username is provided.
// Resolution order: KUBECTL_GS_REGISTRY_PASSWORD env var > interactive prompt > empty.
func (r *runner) resolveRegistryPassword() (string, error) {
	if r.flag.RegistryUsername == "" {
		return "", nil
	}

	// Check environment variable first.
	if password := os.Getenv(envRegistryPassword); password != "" {
		return password, nil
	}

	// Prompt interactively if stdin is a terminal.
	if key.IsTTY() {
		fmt.Fprintf(r.stderr, "Registry password for %s: ", r.flag.RegistryUsername)
		password, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(r.stderr) // newline after password input
		if err != nil {
			return "", fmt.Errorf("reading registry password: %w", err)
		}
		return string(password), nil
	}

	return "", fmt.Errorf("registry password required: set %s or run interactively", envRegistryPassword)
}

// splitOCIURLPrefix extracts the registry host and full repository path
// from a normalized OCI URL prefix and chart name.
// For example: "oci://gsoci.azurecr.io/charts/giantswarm/", "hello-world"
// returns "gsoci.azurecr.io", "charts/giantswarm/hello-world".
func splitOCIURLPrefix(ociURLPrefix, chartName string) (registry, repoPath string) {
	// Strip oci:// scheme.
	s := strings.TrimPrefix(ociURLPrefix, "oci://")
	// Strip trailing slash.
	s = strings.TrimSuffix(s, "/")
	// Split on first slash: registry / repo-prefix.
	registry, repoPrefix, _ := strings.Cut(s, "/")
	if repoPrefix != "" {
		repoPath = repoPrefix + "/" + chartName
	} else {
		repoPath = chartName
	}
	return registry, repoPath
}
