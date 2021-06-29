package provider

import (
	"io"
	"os"
	"text/template"

	"github.com/giantswarm/apiextensions/v3/pkg/annotation"
	"github.com/giantswarm/apiextensions/v3/pkg/apis/infrastructure/v1alpha2"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/runtime"
	capav1alpha3 "sigs.k8s.io/cluster-api-provider-aws/exp/api/v1alpha3"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/internal/label"
)

func WriteAWSTemplate(out io.Writer, config NodePoolCRsConfig) error {
	var err error

	if key.IsCAPAVersion(config.ReleaseVersion) {
		err = WriteCAPATemplate(out, config)
		if err != nil {
			return microerror.Mask(err)
		}
	} else {
		err = WriteGSAWSTemplate(out, config)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func WriteCAPATemplate(out io.Writer, config NodePoolCRsConfig) error {
	var err error

	nodepoolTemplate, err := getCAPANodepoolTemplate(config)
	if err != nil {
		return err
	}

	data := struct {
		ProviderMachinePoolCR string
		MachinePoolCR         string
		KubeadmConfigCR       string
	}{}

	objects := nodepoolTemplate.Objs()
	for _, o := range objects {
		o.SetLabels(map[string]string{
			label.ReleaseVersion: config.ReleaseVersion,
			label.Cluster:        config.ClusterID,
			"cluster.x-k8s.io":   config.ClusterID,
			label.Organization:   config.Owner})
		switch o.GetKind() {
		case "AWSMachinePool":
			var awsmachinepool capav1alpha3.AWSMachinePool
			err = runtime.DefaultUnstructuredConverter.
				FromUnstructured(o.Object, &awsmachinepool)
			if err != nil {
				return microerror.Mask(err)
			}
			awsmachinepool.Spec.AvailabilityZones = config.AvailabilityZones
			awsMachinePoolCRYaml, err := yaml.Marshal(awsmachinepool)
			if err != nil {
				return microerror.Mask(err)
			}
			data.ProviderMachinePoolCR = string(awsMachinePoolCRYaml)
		case "MachinePool":
			MachinePoolCRYaml, err := yaml.Marshal(o.Object)
			if err != nil {
				return microerror.Mask(err)
			}
			data.MachinePoolCR = string(MachinePoolCRYaml)
		case "KubeadmConfig":
			kubeadmConfigCRYaml, err := yaml.Marshal(o.Object)
			if err != nil {
				return microerror.Mask(err)
			}
			data.KubeadmConfigCR = string(kubeadmConfigCRYaml)
		}
	}

	t := template.Must(template.New(config.FileName).Parse(key.MachinePoolAWSCRsTemplate))
	err = t.Execute(out, data)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func WriteGSAWSTemplate(out io.Writer, config NodePoolCRsConfig) error {
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
		UseAlikeInstanceTypes:               config.UseAlikeInstanceTypes,
	}

	crs, err := v1alpha2.NewNodePoolCRs(crsConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	if config.MachineDeploymentSubnet != "" {
		crs.AWSMachineDeployment.Annotations[annotation.AWSSubnetSize] = config.MachineDeploymentSubnet
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

func getCAPANodepoolTemplate(config NodePoolCRsConfig) (client.Template, error) {
	var err error

	c, err := client.New("")
	if err != nil {
		return nil, err
	}

	templateOptions := client.GetClusterTemplateOptions{
		ClusterName:       config.ClusterID,
		TargetNamespace:   config.Owner,
		KubernetesVersion: "v1.19.9",
		ProviderRepositorySource: &client.ProviderRepositorySourceOptions{
			InfrastructureProvider: "aws:v0.6.6",
			Flavor:                 "machinepool",
		},
	}
	os.Setenv("AWS_SUBNET", "")
	os.Setenv("AWS_CONTROL_PLANE_MACHINE_TYPE", "")
	os.Setenv("AWS_REGION", "")
	os.Setenv("AWS_SSH_KEY_NAME", "")

	if replicas := int64(config.NodesMin); replicas > 0 {
		templateOptions.WorkerMachineCount = &replicas
	}

	nodepoolTemplate, err := c.GetClusterTemplate(templateOptions)
	if err != nil {
		return nil, err
	}
	return nodepoolTemplate, nil
}
