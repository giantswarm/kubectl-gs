package provider

import (
	"io"
	"text/template"

	"github.com/giantswarm/apiextensions/v3/pkg/annotation"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	capav1alpha3 "sigs.k8s.io/cluster-api-provider-aws/api/v1alpha3"
	controlplanev1 "sigs.k8s.io/cluster-api-provider-aws/controlplane/eks/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/internal/label"
)

func WriteCAPAEKSTemplate(out io.Writer, config ClusterCRsConfig) error {
	var err error

	if config.ExternalSNAT {
		return microerror.Maskf(invalidFlagError, "--external-snat setting is not available for release %v", config.ReleaseVersion)
	}
	if config.PodsCIDR != "" {
		return microerror.Maskf(invalidFlagError, "--pods-cidr setting is not available for release %v", config.ReleaseVersion)
	}

	clusterTemplate, err := getCAPAClusterTemplate(config, "eks-managedmachinepool")
	if err != nil {
		return err
	}

	data := struct {
		AWSManagedControlPlaneCR string
		ClusterCR                string
		AWSClusterRoleIdentityCR string
	}{}

	crLabels := map[string]string{
		label.ReleaseVersion:            config.ReleaseVersion,
		label.Cluster:                   config.Name,
		capiv1alpha3.ClusterLabelName:   config.Name,
		label.Organization:              config.Owner,
		"cluster.x-k8s.io/watch-filter": "capi"}

	objects := clusterTemplate.Objs()
	for _, o := range objects {
		switch o.GetKind() {
		case "AWSManagedControlPlane":
			o.SetLabels(crLabels)
			awsmanagedcontrolplane, err := newAWSManagedControlPLaneFromUnstructured(config, o)
			if err != nil {
				return microerror.Mask(err)
			}
			awsManagedControlPlaneCRYaml, err := yaml.Marshal(awsmanagedcontrolplane)
			if err != nil {
				return microerror.Mask(err)
			}
			data.AWSManagedControlPlaneCR = string(awsManagedControlPlaneCRYaml)
		case "Cluster":
			clusterLabels := crLabels
			for key, value := range config.Labels {
				clusterLabels[key] = value
			}
			o.SetLabels(clusterLabels)
			o.SetAnnotations(map[string]string{annotation.ClusterDescription: config.Description})
			clusterCRYaml, err := yaml.Marshal(o.Object)
			if err != nil {
				return microerror.Mask(err)
			}
			data.ClusterCR = string(clusterCRYaml)
		}
	}
	{
		awsclusterroleidentity := newAWSClusterRoleIdentity(config)
		awsclusterroleidentity.SetLabels(crLabels)
		awsClusterRoleIdentityCRYaml, err := yaml.Marshal(awsclusterroleidentity)
		if err != nil {
			return microerror.Mask(err)
		}
		data.AWSClusterRoleIdentityCR = string(awsClusterRoleIdentityCRYaml)
	}

	t := template.Must(template.New(config.FileName).Parse(key.ClusterEKSCRsTemplate))
	err = t.Execute(out, data)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func newAWSManagedControlPLaneFromUnstructured(config ClusterCRsConfig, o unstructured.Unstructured) (*controlplanev1.AWSManagedControlPlane, error) {
	var awscluster controlplanev1.AWSManagedControlPlane
	{
		err := runtime.DefaultUnstructuredConverter.
			FromUnstructured(o.Object, &awscluster)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		awscluster.Spec.IdentityRef = &capav1alpha3.AWSIdentityReference{
			Name: config.Name,
			Kind: capav1alpha3.ClusterRoleIdentityKind}

		for _, az := range config.ControlPlaneAZ {
			privateSubnet := capav1alpha3.SubnetSpec{AvailabilityZone: az, IsPublic: false}
			publicSubnet := capav1alpha3.SubnetSpec{AvailabilityZone: az, IsPublic: true}
			awscluster.Spec.NetworkSpec.Subnets = append(awscluster.Spec.NetworkSpec.Subnets, &privateSubnet, &publicSubnet)
		}
		if config.ControlPlaneSubnet != "" {
			awscluster.SetAnnotations(map[string]string{annotation.AWSSubnetSize: config.ControlPlaneSubnet})
		}
	}
	return &awscluster, nil
}
