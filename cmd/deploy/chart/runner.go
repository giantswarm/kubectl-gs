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
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

func (r *runner) run(ctx context.Context, _ *cobra.Command, _ []string) error {
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
	defer ociClient.Close(ctx)

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

			_, _ = fmt.Fprintf(r.stderr, "Resolved latest version: %s\n", version)
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

	// Validate values against chart schema if values are provided and a version is known.
	// When --auto-upgrade=all without --version, version stays empty and validation is skipped
	// because we have no specific tag to fetch annotations from.
	if values != nil && version != "" {
		annotations, err := ociClient.GetManifestAnnotations(ctx, registry, repoPath, version)
		if err != nil {
			_, _ = fmt.Fprintf(r.stderr, "Warning: could not fetch chart annotations for schema validation: %v\n", err)
		} else if schemaURL, ok := annotations[deploychart.ValuesSchemaAnnotation]; ok && schemaURL != "" {
			_, _ = fmt.Fprintf(r.stderr, "Validating values against schema...\n")
			if err := deploychart.ValidateValuesAgainstSchema(values, schemaURL); err != nil {
				return microerror.Mask(err)
			}
		}
	}

	// Connect to the cluster.
	k8sClients, err := r.commonConfig.GetClient(r.logger)
	if err != nil {
		return microerror.Mask(err)
	}

	// Verify the organization namespace exists.
	_, err = k8sClients.K8sClient().CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return microerror.Maskf(invalidFlagError,
				"the organization %q wasn't found\nPlease check for existing organizations using\n\n    kubectl gs get organizations",
				r.flag.Organization)
		}
		return microerror.Mask(err)
	}

	// Detect CRD versions.
	crdVersions, err := deploychart.DetectFluxCRDVersions(ctx, k8sClients.ExtClient())
	if err != nil {
		return microerror.Mask(err)
	}

	// Resolve cluster name.
	clusterName := r.flag.Cluster
	if r.flag.ManagementCluster {
		contextName, err := r.commonConfig.GetCurrentContextName()
		if err != nil {
			return microerror.Mask(err)
		}
		clusterName = deploychart.ClusterNameFromContext(contextName)
	}

	// Verify the target cluster exists (workload cluster mode only).
	if !r.flag.ManagementCluster {
		capiClusterGVR := schema.GroupVersionResource{
			Group:    "cluster.x-k8s.io",
			Version:  "v1beta1",
			Resource: "clusters",
		}
		_, err = k8sClients.DynClient().Resource(capiClusterGVR).Namespace(namespace).Get(ctx, clusterName, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return microerror.Maskf(invalidFlagError,
					"cluster %q not found in organization %q\n\n  Please check for existing clusters using\n\n    kubectl gs get clusters --namespace %s",
					clusterName, r.flag.Organization, namespace)
			}
			return microerror.Mask(err)
		}
	}

	// Compute resource name.
	resourceName := r.flag.Name
	if resourceName == "" {
		resourceName = fmt.Sprintf("%s-%s", clusterName, r.flag.ChartName)
	}
	if len(resourceName) > 253 {
		return microerror.Maskf(invalidFlagError, "resource name %q exceeds maximum length of 253 characters", resourceName)
	}

	// Determine registry Secret name and provider for OCIRepository.
	var registrySecretName string
	registryProvider := r.flag.RegistryProvider

	// When credentials are provided and no cloud provider is set, we create a
	// docker-registry Secret for Flux to use. When --registry-provider is set,
	// Flux uses workload identity — credentials are client-side only.
	if r.flag.RegistryUsername != "" && registryPassword != "" && registryProvider == "" {
		registrySecretName = resourceName + "-registry"
	}

	// Build manifests with detected API versions.
	ociRepoOpts := deploychart.OCIRepositoryOptions{
		Name:        resourceName,
		Namespace:   namespace,
		ClusterName: clusterName,
		URL:         ociURL,
		Version:     version,
		AutoUpgrade: r.flag.AutoUpgrade,
		Interval:    r.flag.Interval,
		APIVersion:  crdVersions.OCIRepositoryAPIVersion,
		SecretRef:   registrySecretName,
		Provider:    registryProvider,
	}

	// Parse --values-from references.
	var valuesFrom []deploychart.ValuesFromReference
	for _, vf := range r.flag.ValuesFrom {
		kind, name, _ := strings.Cut(vf, "/")
		valuesFrom = append(valuesFrom, deploychart.ValuesFromReference{Kind: kind, Name: name})
	}

	helmReleaseOpts := deploychart.HelmReleaseOptions{
		Name:              resourceName,
		Namespace:         namespace,
		ClusterName:       clusterName,
		ChartName:         r.flag.ChartName,
		TargetNamespace:   r.flag.TargetNS,
		Interval:          r.flag.Interval,
		Values:            values,
		ValuesFrom:        valuesFrom,
		ManagementCluster: r.flag.ManagementCluster,
		APIVersion:        crdVersions.HelmReleaseAPIVersion,
	}

	ociRepo := deploychart.BuildOCIRepository(ociRepoOpts)
	helmRelease := deploychart.BuildHelmRelease(helmReleaseOpts)

	// Build registry Secret if needed.
	var registrySecretYAML []byte
	if registrySecretName != "" {
		secret, err := deploychart.BuildRegistrySecret(deploychart.RegistrySecretOptions{
			Name:        registrySecretName,
			Namespace:   namespace,
			ClusterName: clusterName,
			Registry:    registry,
			Username:    r.flag.RegistryUsername,
			Password:    registryPassword,
		})
		if err != nil {
			return microerror.Mask(err)
		}
		registrySecretYAML, err = deploychart.MarshalManifest(secret)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	ociRepoYAML, err := deploychart.MarshalManifest(ociRepo)
	if err != nil {
		return microerror.Mask(err)
	}

	helmReleaseYAML, err := deploychart.MarshalManifest(helmRelease)
	if err != nil {
		return microerror.Mask(err)
	}

	// Apply or dry-run each manifest.
	dynClient := k8sClients.DynClient()
	applyOpts := deploychart.ApplyOptions{DryRun: r.flag.DryRun}

	type manifest struct {
		kind         string
		apiVersion   string
		resourceName string
		yaml         []byte
	}

	var manifests []manifest
	if registrySecretYAML != nil {
		manifests = append(manifests, manifest{"Secret", "v1", registrySecretName, registrySecretYAML})
	}
	manifests = append(manifests,
		manifest{"OCIRepository", crdVersions.OCIRepositoryAPIVersion, resourceName, ociRepoYAML},
		manifest{"HelmRelease", crdVersions.HelmReleaseAPIVersion, resourceName, helmReleaseYAML},
	)

	for _, m := range manifests {
		gvr, err := deploychart.ResourceGVR(m.apiVersion, m.kind)
		if err != nil {
			return microerror.Mask(err)
		}

		// Check if resource already exists.
		existing, err := deploychart.GetExistingResource(ctx, dynClient, gvr, namespace, m.resourceName)
		if err != nil {
			return microerror.Mask(err)
		}

		if existing != nil && !r.flag.DryRun {
			// Compute diff.
			existingYAML, err := yaml.Marshal(existing.Object)
			if err != nil {
				return microerror.Mask(err)
			}

			diff, err := deploychart.DiffManifests(
				fmt.Sprintf("%s/%s", namespace, m.resourceName),
				existingYAML, m.yaml,
			)
			if err != nil {
				return microerror.Mask(err)
			}

			if diff != "" {
				_, _ = fmt.Fprintf(r.stderr, "\n%s %s has changes:\n%s\n", m.kind, m.resourceName, diff)

				if !term.IsTerminal(int(os.Stdin.Fd())) { //nolint:gosec // Fd() returns a small file descriptor
					return microerror.Maskf(confirmationRequiredError, "resources already exist and have changes; run interactively or use --dry-run to preview")
				}

				confirmed, err := deploychart.AskConfirmation(
					fmt.Sprintf("Apply changes to %s %s/%s?", m.kind, namespace, m.resourceName),
					os.Stdin, r.stderr,
				)
				if err != nil {
					return microerror.Mask(err)
				}
				if !confirmed {
					return microerror.Maskf(applyAbortedError, "user declined changes to %s %s/%s", m.kind, namespace, m.resourceName)
				}
			}
		}

		err = deploychart.ApplyManifest(ctx, dynClient, m.yaml, applyOpts)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Print YAML to stdout.
	if registrySecretYAML != nil {
		_, _ = fmt.Fprintf(r.stdout, "%s---\n", string(registrySecretYAML))
	}
	_, _ = fmt.Fprintf(r.stdout, "%s---\n%s", string(ociRepoYAML), string(helmReleaseYAML))

	// Print status to stderr.
	if r.flag.DryRun {
		_, _ = fmt.Fprintf(r.stderr, "Server-side dry-run succeeded.\n")
	} else {
		if registrySecretName != "" {
			_, _ = fmt.Fprintf(r.stderr, "Applied Secret %s/%s\n", namespace, registrySecretName)
		}
		_, _ = fmt.Fprintf(r.stderr, "Applied OCIRepository %s/%s\n", namespace, resourceName)
		_, _ = fmt.Fprintf(r.stderr, "Applied HelmRelease %s/%s\n", namespace, resourceName)
	}

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
	if term.IsTerminal(int(os.Stdin.Fd())) { //nolint:gosec // Fd() returns a small file descriptor
		_, _ = fmt.Fprintf(r.stderr, "Registry password for %s: ", r.flag.RegistryUsername)
		password, err := term.ReadPassword(int(os.Stdin.Fd())) //nolint:gosec // Fd() returns a small file descriptor
		_, _ = fmt.Fprintln(r.stderr)                          // newline after password input
		if err != nil {
			return "", fmt.Errorf("reading registry password: %w", err)
		}
		return string(password), nil
	}

	// No password available — return empty and let regclient try Docker config / anonymous.
	return "", nil
}

// splitOCIURLPrefix extracts the registry host and full repository path
// from a normalized OCI URL prefix and chart name.
// For example: "oci://gsoci.azurecr.io/charts/giantswarm/", "hello-world"
// returns "gsoci.azurecr.io", "charts/giantswarm/hello-world".
// Precondition: ociURLPrefix must start with "oci://" and end with "/"
// (guaranteed by normalizeOCIURLPrefix in flag.go).
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
