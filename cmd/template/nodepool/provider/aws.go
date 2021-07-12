package provider

import (
	"io"
	"os"
	"strconv"
	"text/template"

	"github.com/giantswarm/apiextensions/v3/pkg/annotation"
	"github.com/giantswarm/apiextensions/v3/pkg/apis/infrastructure/v1alpha2"
	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	capav1alpha3 "sigs.k8s.io/cluster-api-provider-aws/exp/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client"
	capiv1alpha3exp "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/internal/key"
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

	if config.UseAlikeInstanceTypes {
		return microerror.Maskf(invalidFlagError, "--use-alike-instance-types setting is not available for release %v", config.ReleaseVersion)
	}

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
		o.SetName(config.NodePoolID)
		o.SetLabels(map[string]string{
			label.ReleaseVersion:            config.ReleaseVersion,
			label.Cluster:                   config.ClusterID,
			label.MachinePool:               config.NodePoolID,
			capiv1alpha3.ClusterLabelName:   config.ClusterID,
			label.Organization:              config.Owner,
			"cluster.x-k8s.io/watch-filter": "capi",
		})
		switch o.GetKind() {
		case "AWSMachinePool":
			awsmachinepool, err := newAWSMachinePoolFromUnstructured(config, o)
			if err != nil {
				return microerror.Mask(err)
			}
			awsMachinePoolCRYaml, err := yaml.Marshal(awsmachinepool)
			if err != nil {
				return microerror.Mask(err)
			}
			data.ProviderMachinePoolCR = string(awsMachinePoolCRYaml)
		case "MachinePool":
			machinepool, err := newMachinePoolFromUnstructured(config, o)
			if err != nil {
				return microerror.Mask(err)
			}
			machinePoolCRYaml, err := yaml.Marshal(machinepool)
			if err != nil {
				return microerror.Mask(err)
			}
			data.MachinePoolCR = string(machinePoolCRYaml)
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

	// Set all environment variables expected by the upstream client to empty strings. These values are defaulted later
	// Make sure that the values are reset.
	for _, envVar := range key.GetCAPAEnvVars() {
		if os.Getenv(envVar) != "" {
			prevEnv := os.Getenv(envVar)
			os.Setenv(envVar, "")
			defer os.Setenv(envVar, prevEnv)
		} else {
			os.Setenv(envVar, "")
			defer os.Unsetenv(envVar)
		}
	}

	if replicas := int64(config.NodesMin); replicas > 0 {
		templateOptions.WorkerMachineCount = &replicas
	}

	nodepoolTemplate, err := c.GetClusterTemplate(templateOptions)
	if err != nil {
		return nil, err
	}
	return nodepoolTemplate, nil
}

func newAWSMachinePoolFromUnstructured(config NodePoolCRsConfig, o unstructured.Unstructured) (*capav1alpha3.AWSMachinePool, error) {
	var awsmachinepool capav1alpha3.AWSMachinePool
	{
		err := runtime.DefaultUnstructuredConverter.
			FromUnstructured(o.Object, &awsmachinepool)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		awsmachinepool.Spec.AvailabilityZones = config.AvailabilityZones
		awsmachinepool.Spec.AWSLaunchTemplate.InstanceType = config.AWSInstanceType
		awsmachinepool.Spec.AWSLaunchTemplate.IamInstanceProfile = key.GetNodeInstanceProfile(config.NodePoolID, config.ClusterID)
		onDemandBaseCapacity := int64(config.OnDemandBaseCapacity)
		onDemandPercentageAboveBaseCapacity := int64(config.OnDemandPercentageAboveBaseCapacity)
		awsmachinepool.Spec.MixedInstancesPolicy = &capav1alpha3.MixedInstancesPolicy{
			InstancesDistribution: &capav1alpha3.InstancesDistribution{
				OnDemandBaseCapacity:                &onDemandBaseCapacity,
				OnDemandPercentageAboveBaseCapacity: &onDemandPercentageAboveBaseCapacity,
			},
		}
		if config.MachineDeploymentSubnet != "" {
			awsmachinepool.SetAnnotations(map[string]string{annotation.AWSSubnetSize: config.MachineDeploymentSubnet})
		}
	}
	return &awsmachinepool, nil
}

func newMachinePoolFromUnstructured(config NodePoolCRsConfig, o unstructured.Unstructured) (*capiv1alpha3exp.MachinePool, error) {
	var machinepool capiv1alpha3exp.MachinePool
	{
		err := runtime.DefaultUnstructuredConverter.
			FromUnstructured(o.Object, &machinepool)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		machinepool.SetAnnotations(map[string]string{
			annotation.MachinePoolName: config.Description,
			annotation.NodePoolMinSize: strconv.Itoa(config.NodesMin),
			annotation.NodePoolMaxSize: strconv.Itoa(config.NodesMax)})

		machinepool.Spec.Template.Spec.Bootstrap.ConfigRef.Name = config.NodePoolID
		machinepool.Spec.Template.Spec.InfrastructureRef.Name = config.NodePoolID
	}
	return &machinepool, nil
}
