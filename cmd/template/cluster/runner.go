package cluster

import (
	"context"
	"io"
	"os"
	"sort"
	"strings"
	"text/template"

	"github.com/ghodss/yaml"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/pkg/aws"
	"github.com/giantswarm/kubectl-gs/pkg/gsrelease"
	"github.com/giantswarm/kubectl-gs/pkg/template/cluster"
	"github.com/giantswarm/kubectl-gs/pkg/template/nodepool"
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
		ClusterID:         r.flag.ClusterID,
		Domain:            r.flag.Domain,
		ExternalSNAT:      r.flag.ExternalSNAT,
		MasterAZ:          r.flag.MasterAZ,
		Name:              r.flag.Name,
		PodsCIDR:          r.flag.PodsCIDR,
		Owner:             r.flag.Owner,
		Region:            r.flag.Region,
		ReleaseComponents: releaseComponents,
		ReleaseVersion:    r.flag.Release,
	}

	clusterCR, awsClusterCR, g8sControlPlaneCR, awsControlPlaneCR, err := cluster.NewClusterCRs(config)

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

	var mdCRYaml, awsMDCRYaml []byte
	{
		if r.flag.TemplateDefaultNodepool {

			var availabilityZones []string
			{
				if r.flag.AvailabilityZones != "" {
					availabilityZones = strings.Split(r.flag.AvailabilityZones, ",")
				} else {
					availabilityZones = aws.GetAvailabilityZones(r.flag.NumAvailabilityZones, r.flag.Region)
				}
			}

			nodepoolConfig := nodepool.Config{
				AvailabilityZones: availabilityZones,
				AWSInstanceType:   r.flag.AWSInstanceType,
				ClusterID:         clusterCR.Name,
				Name:              r.flag.NodepoolName,
				NodesMax:          r.flag.NodesMax,
				NodesMin:          r.flag.NodesMin,
				Owner:             r.flag.Owner,
				ReleaseComponents: releaseComponents,
				ReleaseVersion:    r.flag.Release,
			}

			mdCR, awsMDCR, err := nodepool.NewMachineDeploymentCRs(nodepoolConfig)

			mdCRYaml, err = yaml.Marshal(mdCR)
			if err != nil {
				return microerror.Mask(err)
			}

			awsMDCRYaml, err = yaml.Marshal(awsMDCR)
			if err != nil {
				return microerror.Mask(err)
			}
		}
	}

	type ClusterCRsOutput struct {
		AWSClusterCR            string
		AWSControlPlaneCR       string
		AWSMachineDeploymentCR  string
		ClusterCR               string
		G8sControlPlaneCR       string
		MachineDeploymentCR     string
		TemplateDefaultNodepool bool
	}

	clusterCRsOutput := ClusterCRsOutput{
		AWSClusterCR:            string(awsClusterCRYaml),
		AWSMachineDeploymentCR:  string(awsMDCRYaml),
		ClusterCR:               string(clusterCRYaml),
		MachineDeploymentCR:     string(mdCRYaml),
		G8sControlPlaneCR:       string(g8sControlPlaneCRYaml),
		AWSControlPlaneCR:       string(awsControlPlaneCRYaml),
		TemplateDefaultNodepool: r.flag.TemplateDefaultNodepool,
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
