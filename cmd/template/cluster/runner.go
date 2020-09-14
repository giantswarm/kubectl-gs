package cluster

import (
	"context"
	"io"
	"os"
	"sort"
	"strings"
	"text/template"

	"github.com/ghodss/yaml"
	"github.com/giantswarm/apiextensions/pkg/apis/infrastructure/v1alpha2"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/pkg/clusterlabels"
	"github.com/giantswarm/kubectl-gs/pkg/release"
)

type runner struct {
	flag   *flag
	logger micrologger.Logger
	stdout io.Writer
	stderr io.Writer
}

func (r *runner) Run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Sorting is required before validation for uniqueness.
	sort.Slice(r.flag.MasterAZ, func(i, j int) bool {
		return r.flag.MasterAZ[i] < r.flag.MasterAZ[j]
	})

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
	var err error

	var releaseComponents map[string]string
	{
		c := release.Config{}

		releaseCollection, err := release.New(c)
		if err != nil {
			return microerror.Mask(err)
		}

		releaseComponents = releaseCollection.ReleaseComponents(r.flag.Release)
	}

	var userLabels map[string]string
	{
		userLabels, err = clusterlabels.Parse(r.flag.Label)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// remove leading v from release flag input
	sanitizedRelease := strings.TrimLeft(r.flag.Release, "v")

	config := v1alpha2.ClusterCRsConfig{
		ClusterID:         r.flag.ClusterID,
		Credential:        r.flag.Credential,
		Domain:            r.flag.Domain,
		ExternalSNAT:      r.flag.ExternalSNAT,
		MasterAZ:          r.flag.MasterAZ,
		Description:       r.flag.Name,
		PodsCIDR:          r.flag.PodsCIDR,
		Owner:             r.flag.Owner,
		Region:            r.flag.Region,
		ReleaseComponents: releaseComponents,
		ReleaseVersion:    sanitizedRelease,
		Labels:            userLabels,
	}

	crs, err := v1alpha2.NewClusterCRs(config)
	if err != nil {
		return microerror.Mask(err)
	}

	clusterCRYaml, err := yaml.Marshal(crs.Cluster)
	if err != nil {
		return microerror.Mask(err)
	}

	awsClusterCRYaml, err := yaml.Marshal(crs.AWSCluster)
	if err != nil {
		return microerror.Mask(err)
	}

	g8sControlPlaneCRYaml, err := yaml.Marshal(crs.G8sControlPlane)
	if err != nil {
		return microerror.Mask(err)
	}

	awsControlPlaneCRYaml, err := yaml.Marshal(crs.AWSControlPlane)
	if err != nil {
		return microerror.Mask(err)
	}

	data := struct {
		AWSClusterCR      string
		AWSControlPlaneCR string
		ClusterCR         string
		G8sControlPlaneCR string
	}{
		AWSClusterCR:      string(awsClusterCRYaml),
		ClusterCR:         string(clusterCRYaml),
		G8sControlPlaneCR: string(g8sControlPlaneCRYaml),
		AWSControlPlaneCR: string(awsControlPlaneCRYaml),
	}

	t := template.Must(template.New("clusterCR").Parse(key.ClusterCRsTemplate))

	{
		var output *os.File

		if r.flag.Output == "" {
			output = os.Stdout
		} else {
			f, err := os.Create(r.flag.Output)
			if err != nil {
				return microerror.Mask(err)
			}
			defer f.Close()

			output = f
		}

		err = t.Execute(output, data)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
