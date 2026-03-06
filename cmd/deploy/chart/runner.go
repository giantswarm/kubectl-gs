package chart

import (
	"context"
	"fmt"
	"io"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/v6/internal/deploychart"
	"github.com/giantswarm/kubectl-gs/v6/internal/key"
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

	ociURL := r.flag.OCIURLPrefix + r.flag.ChartName

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
		Version:     r.flag.Version,
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

	ociRepoYAML, err := deploychart.MarshalCRD(ociRepo)
	if err != nil {
		return microerror.Mask(err)
	}

	helmReleaseYAML, err := deploychart.MarshalCRD(helmRelease)
	if err != nil {
		return microerror.Mask(err)
	}

	fmt.Fprintf(r.stdout, "%s---\n%s", string(ociRepoYAML), string(helmReleaseYAML))

	return nil
}
