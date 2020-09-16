package provider

import (
	"github.com/giantswarm/apiextensions/pkg/apis/infrastructure/v1alpha2"
	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/microerror"
	"io"
	"sigs.k8s.io/yaml"
	"text/template"
)

func WriteAWSTemplate(out io.Writer, config NodePoolCRsConfig) error {
	var err error

	crsConfig := v1alpha2.NodePoolCRsConfig{
		AvailabilityZones:                   config.AvailabilityZones,
		AWSInstanceType:                     config.AWSInstanceType,
		ClusterID:                           config.ClusterID,
		Description:                         config.Description,
		MachineDeploymentID:                 config.NodePoolID,
		NodesMax:                            config.NodesMax,
		NodesMin:                            config.NodesMin,
		OnDemandBaseCapacity:                config.OnDemandBaseCapacity,
		OnDemandPercentageAboveBaseCapacity: config.OnDemandPercentageAboveBaseCapacity,
		Owner:                               config.Owner,
		ReleaseComponents:                   config.ReleaseComponents,
		ReleaseVersion:                      config.ReleaseVersion,
		UseAlikeInstanceTypes:               config.UseAlikeInstanceTypes,
	}

	crs, err := v1alpha2.NewNodePoolCRs(crsConfig)
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

	t := template.Must(template.New(config.FileName).Parse(key.MachineDeploymentCRsTemplate))
	err = t.Execute(out, data)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
