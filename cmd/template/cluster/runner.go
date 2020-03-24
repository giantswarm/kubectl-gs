package cluster

import (
	"context"
	"fmt"
	"io"

	"github.com/ghodss/yaml"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"

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
		c := gsrelease.Config{
			NoCache: r.flag.NoCache,
		}

		release, err = gsrelease.New(c)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	releaseComponents := release.ReleaseComponents(r.flag.Release)

	config := cluster.Config{
		Domain:            r.flag.Domain,
		MasterAZ:          r.flag.MasterAZ,
		Name:              r.flag.Name,
		Owner:             r.flag.Owner,
		Region:            r.flag.Region,
		ReleaseComponents: releaseComponents,
		ReleaseVersion:    r.flag.Release,
	}

	clusterCR, awsClusterCR, err := cluster.NewClusterCRs(config)

	clusterCRYaml, err := yaml.Marshal(clusterCR)
	if err != nil {
		return microerror.Mask(err)
	}

	awsClusterCRYaml, err := yaml.Marshal(awsClusterCR)
	if err != nil {
		return microerror.Mask(err)
	}

	fmt.Println(string(clusterCRYaml))
	fmt.Println(string(awsClusterCRYaml))

	return nil
}
