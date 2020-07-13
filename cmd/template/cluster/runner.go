package cluster

import (
	"context"
	"io"
	"os"
	"sort"
	"text/template"

	"github.com/ghodss/yaml"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/pkg/clusterlabels"
	"github.com/giantswarm/kubectl-gs/pkg/gsrelease"
	"github.com/giantswarm/kubectl-gs/pkg/template/cluster"
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

	var release *gsrelease.GSRelease
	{
		c := gsrelease.Config{}

		release, err = gsrelease.New(c)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	releaseComponents := release.ReleaseComponents(r.flag.Release)

	userLabels, _ := clusterlabels.Parse(r.flag.Label)

	config := cluster.Config{
		ClusterID:         r.flag.ClusterID,
		Credential:        r.flag.Credential,
		Domain:            r.flag.Domain,
		ExternalSNAT:      r.flag.ExternalSNAT,
		MasterAZ:          r.flag.MasterAZ,
		Name:              r.flag.Name,
		PodsCIDR:          r.flag.PodsCIDR,
		Owner:             r.flag.Owner,
		Region:            r.flag.Region,
		ReleaseComponents: releaseComponents,
		ReleaseVersion:    r.flag.Release,
		Labels:            userLabels,
	}

	clusterCR, awsClusterCR, g8sControlPlaneCR, awsControlPlaneCR, err := cluster.NewClusterCRs(config)
	if err != nil {
		return microerror.Mask(err)
	}

	clusterCRYaml, err := yaml.Marshal(clusterCR)
	if err != nil {
		return microerror.Mask(err)
	}

	awsClusterCRYaml, err := yaml.Marshal(awsClusterCR)
	if err != nil {
		return microerror.Mask(err)
	}

	g8sControlPlaneCRYaml, err := yaml.Marshal(g8sControlPlaneCR)
	if err != nil {
		return microerror.Mask(err)
	}

	awsControlPlaneCRYaml, err := yaml.Marshal(awsControlPlaneCR)
	if err != nil {
		return microerror.Mask(err)
	}

	type ClusterCRsOutput struct {
		AWSClusterCR      string
		AWSControlPlaneCR string
		ClusterCR         string
		G8sControlPlaneCR string
	}

	clusterCRsOutput := ClusterCRsOutput{
		AWSClusterCR:      string(awsClusterCRYaml),
		ClusterCR:         string(clusterCRYaml),
		G8sControlPlaneCR: string(g8sControlPlaneCRYaml),
		AWSControlPlaneCR: string(awsControlPlaneCRYaml),
	}

	t := template.Must(template.New("clusterCR").Parse(key.ClusterCRsTemplate))

	var output *os.File
	{
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

		err = t.Execute(output, clusterCRsOutput)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
