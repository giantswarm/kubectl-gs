package provider

import (
	"github.com/giantswarm/apiextensions/pkg/apis/infrastructure/v1alpha2"
	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/microerror"
	"io"
	"sigs.k8s.io/yaml"
	"text/template"
)

func WriteAWSTemplate(out io.Writer, config ClusterCRConfig) error {
	var err error

	crsConfig := v1alpha2.ClusterCRsConfig{
		ClusterID:         config.ClusterID,
		Credential:        config.Credential,
		Domain:            config.Domain,
		ExternalSNAT:      config.ExternalSNAT,
		MasterAZ:          config.MasterAZ,
		Description:       config.Description,
		PodsCIDR:          config.PodsCIDR,
		Owner:             config.Owner,
		Region:            config.Region,
		ReleaseComponents: config.ReleaseComponents,
		ReleaseVersion:    config.ReleaseVersion,
		Labels:            config.Labels,
	}

	crs, err := v1alpha2.NewClusterCRs(crsConfig)
	if err != nil {
		return microerror.Mask(err)
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
