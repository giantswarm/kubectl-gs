package nodepool

import (
	"context"
	"html/template"
	"io"
	"os"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/giantswarm/apiextensions/pkg/apis/infrastructure/v1alpha2"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/pkg/aws"
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

	var availabilityZones []string
	{
		if r.flag.AvailabilityZones != "" {
			availabilityZones = strings.Split(r.flag.AvailabilityZones, ",")
		} else {
			availabilityZones = aws.GetAvailabilityZones(r.flag.NumAvailabilityZones, r.flag.Region)
		}
	}

	var releaseComponents map[string]string
	{
		c := release.Config{}

		releaseCollection, err := release.New(c)
		if err != nil {
			return microerror.Mask(err)
		}

		releaseComponents = releaseCollection.ReleaseComponents(r.flag.Release)
	}

	config := v1alpha2.NodePoolCRsConfig{
		AvailabilityZones:                   availabilityZones,
		AWSInstanceType:                     r.flag.AWSInstanceType,
		ClusterID:                           r.flag.ClusterID,
		Description:                         r.flag.NodepoolName,
		NodesMax:                            r.flag.NodesMax,
		NodesMin:                            r.flag.NodesMin,
		OnDemandBaseCapacity:                r.flag.OnDemandBaseCapacity,
		OnDemandPercentageAboveBaseCapacity: r.flag.OnDemandPercentageAboveBaseCapacity,
		Owner:                               r.flag.Owner,
		ReleaseComponents:                   releaseComponents,
		ReleaseVersion:                      r.flag.Release,
		UseAlikeInstanceTypes:               r.flag.UseAlikeInstanceTypes,
	}

	crs, err := v1alpha2.NewNodePoolCRs(config)
	if err != nil {
		return microerror.Mask(err)
	}

	mdCRYaml, err := yaml.Marshal(crs.MachineDeployment)
	if err != nil {
		return microerror.Mask(err)
	}

	awsMDCRYaml, err := yaml.Marshal(crs.AWSMachineDeployment)
	if err != nil {
		return microerror.Mask(err)
	}

	data := struct {
		AWSMachineDeploymentCR string
		MachineDeploymentCR    string
	}{
		AWSMachineDeploymentCR: string(awsMDCRYaml),
		MachineDeploymentCR:    string(mdCRYaml),
	}

	t := template.Must(template.New("nodepoolCR").Parse(key.MachineDeploymentCRsTemplate))

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
