package cluster

import (
	"io"
	"strings"
	"text/template"

	"github.com/giantswarm/apiextensions/pkg/apis/infrastructure/v1alpha2"
	"github.com/giantswarm/microerror"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/pkg/clusterlabels"
	"github.com/giantswarm/kubectl-gs/pkg/release"
)

func writeAWSTemplate(out io.Writer, name string, flags *flag) error {
	var err error

	var releaseComponents map[string]string
	{
		c := release.Config{}

		releaseCollection, err := release.New(c)
		if err != nil {
			return microerror.Mask(err)
		}

		releaseComponents = releaseCollection.ReleaseComponents(flags.Release)
	}

	var userLabels map[string]string
	{
		userLabels, err = clusterlabels.Parse(flags.Label)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Remove leading 'v' from release flag input.
	sanitizedRelease := strings.TrimLeft(flags.Release, "v")

	config := v1alpha2.ClusterCRsConfig{
		ClusterID:         flags.ClusterID,
		Credential:        flags.Credential,
		Domain:            flags.Domain,
		ExternalSNAT:      flags.ExternalSNAT,
		MasterAZ:          flags.MasterAZ,
		Description:       flags.Name,
		PodsCIDR:          flags.PodsCIDR,
		Owner:             flags.Owner,
		Region:            flags.Region,
		ReleaseComponents: releaseComponents,
		ReleaseVersion:    sanitizedRelease,
		Labels:            userLabels,
	}

	crs, err := v1alpha2.NewClusterCRs(config)
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

	t := template.Must(template.New(name).Parse(key.ClusterAWSCRsTemplate))
	err = t.Execute(out, data)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
