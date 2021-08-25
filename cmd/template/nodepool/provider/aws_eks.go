package provider

import (
	"io"
	"strconv"
	"text/template"

	"github.com/giantswarm/apiextensions/v3/pkg/annotation"
	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	capav1alpha3 "sigs.k8s.io/cluster-api-provider-aws/exp/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiv1alpha3exp "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/internal/key"
)

func WriteCAPAEKSTemplate(out io.Writer, config NodePoolCRsConfig) error {
	var err error

	if config.UseAlikeInstanceTypes {
		return microerror.Maskf(invalidFlagError, "--use-alike-instance-types setting is not available for release %v", config.ReleaseVersion)
	}

	nodepoolTemplate, err := getCAPANodepoolTemplate(config, "eks-managedmachinepool")
	if err != nil {
		return err
	}

	data := struct {
		ManagedMachinePoolCR string
		MachinePoolCR        string
	}{}

	objects := nodepoolTemplate.Objs()
	for _, o := range objects {
		o.SetName(config.NodePoolID)
		o.SetLabels(map[string]string{
			label.ReleaseVersion:            config.ReleaseVersion,
			label.Cluster:                   config.ClusterName,
			label.MachinePool:               config.NodePoolID,
			capiv1alpha3.ClusterLabelName:   config.ClusterName,
			label.Organization:              config.Owner,
			"cluster.x-k8s.io/watch-filter": "capi",
		})
		switch o.GetKind() {
		case "AWSManagedMachinePool":
			awsmachinepool, err := newAWSManagedMachinePoolFromUnstructured(config, o)
			if err != nil {
				return microerror.Mask(err)
			}
			awsMachinePoolCRYaml, err := yaml.Marshal(awsmachinepool)
			if err != nil {
				return microerror.Mask(err)
			}
			data.ManagedMachinePoolCR = string(awsMachinePoolCRYaml)
		case "MachinePool":
			machinepool, err := newMachinePoolFromUnstructuredEKS(config, o)
			if err != nil {
				return microerror.Mask(err)
			}
			machinePoolCRYaml, err := yaml.Marshal(machinepool)
			if err != nil {
				return microerror.Mask(err)
			}
			data.MachinePoolCR = string(machinePoolCRYaml)
		}
	}

	t := template.Must(template.New(config.FileName).Parse(key.MachinePoolEKSCRsTemplate))
	err = t.Execute(out, data)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func newAWSManagedMachinePoolFromUnstructured(config NodePoolCRsConfig, o unstructured.Unstructured) (*capav1alpha3.AWSManagedMachinePool, error) {
	var awsmachinepool capav1alpha3.AWSManagedMachinePool
	{
		err := runtime.DefaultUnstructuredConverter.
			FromUnstructured(o.Object, &awsmachinepool)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		awsmachinepool.Spec.AvailabilityZones = config.AvailabilityZones
		awsmachinepool.Spec.InstanceType = &config.AWSInstanceType

		var min = int32(config.NodesMin)
		var max = int32(config.NodesMax)
		awsmachinepool.Spec.Scaling = &capav1alpha3.ManagedMachinePoolScaling{
			MinSize: &min,
			MaxSize: &max,
		}

		awsmachinepool.Spec.SubnetIDs = nil // for now we dont allow spec of subnets ID and  this needs to be ommited
		if config.MachineDeploymentSubnet != "" {
			awsmachinepool.SetAnnotations(map[string]string{annotation.AWSSubnetSize: config.MachineDeploymentSubnet})
		}
	}
	return &awsmachinepool, nil
}

func newMachinePoolFromUnstructuredEKS(config NodePoolCRsConfig, o unstructured.Unstructured) (*capiv1alpha3exp.MachinePool, error) {
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

		machinepool.Spec.Template.Spec.InfrastructureRef.Name = config.NodePoolID

	}
	return &machinepool, nil
}
