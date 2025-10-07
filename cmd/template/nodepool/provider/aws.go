package provider

import (
	"context"
	"io"
	"text/template"

	"github.com/giantswarm/k8sclient/v8/pkg/k8sclient"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/v5/cmd/template/nodepool/provider/templates/aws"
	"github.com/giantswarm/kubectl-gs/v5/internal/key"
)

func WriteAWSTemplate(ctx context.Context, client k8sclient.Interface, out io.Writer, config NodePoolCRsConfig) error {
	return WriteGSAWSTemplate(ctx, client, out, config)
}

func WriteGSAWSTemplate(ctx context.Context, client k8sclient.Interface, out io.Writer, config NodePoolCRsConfig) error {
	var err error
	config.ReleaseComponents, err = key.GetReleaseComponents(ctx, client.CtrlClient(), config.ReleaseVersion)
	if err != nil {
		return microerror.Mask(err)
	}

	crsConfig := aws.NodePoolCRsConfig{
		AvailabilityZones:                   config.AvailabilityZones,
		AWSInstanceType:                     config.AWSInstanceType,
		ClusterName:                         config.ClusterName,
		Description:                         config.Description,
		MachineDeploymentName:               config.NodePoolName,
		NodesMax:                            config.NodesMax,
		NodesMin:                            config.NodesMin,
		OnDemandBaseCapacity:                config.OnDemandBaseCapacity,
		OnDemandPercentageAboveBaseCapacity: config.OnDemandPercentageAboveBaseCapacity,
		Owner:                               config.Organization,
		UseAlikeInstanceTypes:               config.UseAlikeInstanceTypes,
		ReleaseVersion:                      config.ReleaseVersion,
		ReleaseComponents:                   config.ReleaseComponents,
	}

	crs, err := aws.NewNodePoolCRs(crsConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	if config.MachineDeploymentSubnet != "" {
		crs.AWSMachineDeployment.Annotations[annotation.AWSSubnetSize] = config.MachineDeploymentSubnet
	}

	// Starting with v16.0.0, clusters are created in the org-namespace. This also applies to nodepools.
	// However, there is a possibility that a cluster in a higher version has been upgraded and is still in
	// the default namespace. Therefore we allow to explicitly set the namespace here so that users can
	// ensure their nodepool is in the cluster namespace.
	var namespace string
	{
		if config.Namespace != "" {
			namespace = config.Namespace
		} else if key.IsOrgNamespaceVersion(config.ReleaseVersion) {
			namespace = key.OrganizationNamespaceFromName(config.Organization)
		} else {
			namespace = metav1.NamespaceDefault
		}
	}
	crs = moveCRsToNamespace(crs, namespace)

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

func moveCRsToNamespace(crs aws.NodePoolCRs, namespace string) aws.NodePoolCRs {
	crs.MachineDeployment.SetNamespace(namespace)
	crs.MachineDeployment.Spec.Template.Spec.InfrastructureRef.Namespace = namespace
	crs.AWSMachineDeployment.SetNamespace(namespace)
	return crs
}
