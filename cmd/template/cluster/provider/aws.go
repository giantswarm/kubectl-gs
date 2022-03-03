package provider

import (
	"context"
	"io"
	"text/template"

	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/microerror"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/cmd/template/cluster/provider/templates/aws"
	"github.com/giantswarm/kubectl-gs/internal/key"
)

func WriteAWSTemplate(ctx context.Context, client k8sclient.Interface, out io.Writer, config ClusterConfig) error {
	var err error

	isCapiVersion, err := key.IsCAPIVersion(config.ReleaseVersion)
	if err != nil {
		return microerror.Mask(err)
	}

	if isCapiVersion {
		if config.AWS.EKS {
			err = WriteCAPAEKSTemplate(ctx, client, out, config)
			if err != nil {
				return microerror.Mask(err)
			}
		} else {
			err = WriteCAPATemplate(ctx, client, out, config)
			if err != nil {
				return microerror.Mask(err)
			}
		}
	} else {
		err = WriteGSAWSTemplate(out, config)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func WriteGSAWSTemplate(out io.Writer, config ClusterConfig) error {
	var err error

	crsConfig := aws.ClusterCRsConfig{
		ClusterName: config.Name,

		ExternalSNAT:   config.AWS.ExternalSNAT,
		ControlPlaneAZ: config.ControlPlaneAZ,
		Description:    config.Description,
		PodsCIDR:       config.PodsCIDR,
		Owner:          config.Organization,
		ReleaseVersion: config.ReleaseVersion,
		Labels:         config.Labels,
	}

	crs, err := aws.NewClusterCRs(crsConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	if config.AWS.ControlPlaneSubnet != "" {
		crs.AWSCluster.Annotations[annotation.AWSSubnetSize] = config.AWS.ControlPlaneSubnet
	}

	if key.IsOrgNamespaceVersion(config.ReleaseVersion) {
		crs = moveCRsToOrgNamespace(crs, config.Organization)
	}

	clusterCRYaml, err := yaml.Marshal(crs.Cluster)
	if err != nil {
		return microerror.Mask(err)
	}

	awsClusterCRYaml, err := yaml.Marshal(crs.AWSCluster)
	if err != nil {
		return microerror.Mask(err)
	}

	g8sControlPlaneCRYaml, err := yaml.Marshal(crs.G8sControlPlane)
	if err != nil {
		return microerror.Mask(err)
	}

	awsControlPlaneCRYaml, err := yaml.Marshal(crs.AWSControlPlane)
	if err != nil {
		return microerror.Mask(err)
	}

	data := struct {
		AWSClusterCR      string
		AWSControlPlaneCR string
		ClusterCR         string
		G8sControlPlaneCR string
	}{
		AWSClusterCR:      string(awsClusterCRYaml),
		ClusterCR:         string(clusterCRYaml),
		G8sControlPlaneCR: string(g8sControlPlaneCRYaml),
		AWSControlPlaneCR: string(awsControlPlaneCRYaml),
	}

	t := template.Must(template.New(config.FileName).Parse(key.ClusterAWSCRsTemplate))
	err = t.Execute(out, data)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func moveCRsToOrgNamespace(crs aws.ClusterCRs, organization string) aws.ClusterCRs {
	crs.Cluster.SetNamespace(key.OrganizationNamespaceFromName(organization))
	crs.Cluster.Spec.InfrastructureRef.Namespace = key.OrganizationNamespaceFromName(organization)
	crs.AWSCluster.SetNamespace(key.OrganizationNamespaceFromName(organization))
	crs.G8sControlPlane.SetNamespace(key.OrganizationNamespaceFromName(organization))
	crs.G8sControlPlane.Spec.InfrastructureRef.Namespace = key.OrganizationNamespaceFromName(organization)
	crs.AWSControlPlane.SetNamespace(key.OrganizationNamespaceFromName(organization))
	return crs
}
