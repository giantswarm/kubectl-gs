package cluster

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/v2/cmd/template/cluster/provider"
	"github.com/giantswarm/kubectl-gs/v2/internal/key"
	"github.com/giantswarm/kubectl-gs/v2/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/v2/pkg/labels"
)

type runner struct {
	commonConfig *commonconfig.CommonConfig
	flag         *flag
	logger       micrologger.Logger
	stdout       io.Writer
	stderr       io.Writer

	clusterName         string
	clusterOrganization string
}

func (r *runner) Run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Sorting is required before validation for uniqueness.
	sort.Slice(r.flag.ControlPlaneAZ, func(i, j int) bool {
		return r.flag.ControlPlaneAZ[i] < r.flag.ControlPlaneAZ[j]
	})

	err := r.flag.Validate()
	if err != nil {
		return microerror.Mask(err)
	}
	client, err := r.commonConfig.GetClient(r.logger)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.run(ctx, client)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *runner) run(ctx context.Context, client k8sclient.Interface) error {
	output := r.stdout
	if r.flag.Output != "" {
		outFile, err := os.Create(r.flag.Output)
		if err != nil {
			return microerror.Mask(err)
		}

		defer outFile.Close()
		output = outFile
	}

	if r.flag.ClusterAppConfigYAML == "" && r.flag.DefaultAppConfigYAML == "" {
		config, err := r.getClusterConfig()
		if err != nil {
			return microerror.Mask(err)
		}

		switch r.flag.Provider {
		case key.ProviderAWS:
			err = provider.WriteAWSTemplate(ctx, client, output, config)
			if err != nil {
				return microerror.Mask(err)
			}
		case key.ProviderAzure:
			err = provider.WriteAzureTemplate(ctx, client, output, config)
			if err != nil {
				return microerror.Mask(err)
			}
		case key.ProviderCAPA:
			err = provider.WriteCAPATemplate(ctx, client, output, config)
			if err != nil {
				return microerror.Mask(err)
			}
		case key.ProviderGCP:
			err = provider.WriteGCPTemplate(ctx, client, output, config)
			if err != nil {
				return microerror.Mask(err)
			}
		case key.ProviderOpenStack:
			err = provider.WriteOpenStackTemplate(ctx, client, output, config)
			if err != nil {
				return microerror.Mask(err)
			}
		case key.ProviderVSphere:
			err = provider.WriteVSphereTemplate(ctx, client, output, config)
			if err != nil {
				return microerror.Mask(err)
			}
		default:
			return microerror.Mask(templateFlagNotImplemented)
		}
	} else {
		// read given cluster yaml
		clusterAppConfig, err := r.getClusterYAML()
		if err != nil {
			return microerror.Mask(err)
		}

		clusterApp, err := provider.GetClusterApp(ctx, client, r.flag.Provider, r.flag.App.ClusterCatalog, r.flag.App.ClusterVersion)
		if err != nil {
			return microerror.Mask(err)
		}

		// validate given yaml against cluster-azure app values schema
		err = provider.ValidateYAML(ctx, r.logger, client, clusterApp, clusterAppConfig)
		if err != nil {
			return microerror.Mask(err)
		}

		// template cluster app
		err = provider.TemplateClusterApp(ctx, output, r.flag.Provider, r.clusterName, r.clusterOrganization, clusterApp, clusterAppConfig)
		if err != nil {
			return microerror.Mask(err)
		}

		// read given cluster yaml
		defaultAppConfig, err := r.getDefaultAppYAML()
		if err != nil {
			return microerror.Mask(err)
		}

		clusterDefaultApp, err := provider.GetDefaultApp(ctx, client, r.flag.Provider, r.flag.App.DefaultAppsCatalog, r.flag.App.DefaultAppsVersion)
		if err != nil {
			return microerror.Mask(err)
		}

		// validate given yaml against cluster-azure app values schema
		err = provider.ValidateYAML(ctx, r.logger, client, clusterDefaultApp, defaultAppConfig)
		if err != nil {
			return microerror.Mask(err)
		}

		err = provider.TemplateDefaultApp(ctx, output, r.flag.Provider, r.clusterName, r.clusterOrganization, clusterDefaultApp, defaultAppConfig)
		if err != nil {
			return microerror.Mask(err)
		}

	}

	return nil
}

func (r *runner) getClusterConfig() (provider.ClusterConfig, error) {
	config := provider.ClusterConfig{
		ControlPlaneAZ:           r.flag.ControlPlaneAZ,
		ControlPlaneInstanceType: r.flag.ControlPlaneInstanceType,
		Description:              r.flag.Description,
		KubernetesVersion:        r.flag.KubernetesVersion,
		Name:                     r.flag.Name,
		Organization:             r.flag.Organization,
		PodsCIDR:                 r.flag.PodsCIDR,
		ReleaseVersion:           r.flag.Release,
		Namespace:                metav1.NamespaceDefault,
		Region:                   r.flag.Region,
		ServicePriority:          r.flag.ServicePriority,

		App:       r.flag.App,
		AWS:       r.flag.AWS,
		GCP:       r.flag.GCP,
		OIDC:      r.flag.OIDC,
		OpenStack: r.flag.OpenStack,
	}

	if config.Name == "" {
		generatedName, err := key.GenerateName(true)
		if err != nil {
			return provider.ClusterConfig{}, microerror.Mask(err)
		}

		config.Name = generatedName
	}

	// Remove leading 'v' from release flag input.
	config.ReleaseVersion = strings.TrimLeft(config.ReleaseVersion, "v")

	var err error
	config.Labels, err = labels.Parse(r.flag.Label)
	if err != nil {
		return provider.ClusterConfig{}, microerror.Mask(err)
	}

	if r.flag.Provider != key.ProviderAWS {
		config.Namespace = key.OrganizationNamespaceFromName(config.Organization)
	}

	return config, nil
}

// getClusterYAML reads the given cluster yaml file
// and overwrite some metadata fields if --name and/or --organization is set
func (r *runner) getClusterYAML() (map[string]interface{}, error) {
	yamlFile, err := os.ReadFile(r.flag.ClusterAppConfigYAML)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var yamlConfig map[string]interface{}

	err = yaml.Unmarshal(yamlFile, &yamlConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// overwrite metadata information from flags if given
	if r.flag.Name != "" {
		yamlConfig["metadata"].(map[string]interface{})["name"] = r.flag.Name
		yamlConfig["metadata"].(map[string]interface{})["description"] = r.flag.Name + " cluster"
		r.clusterName = fmt.Sprintf("%v", yamlConfig["metadata"].(map[string]interface{})["name"])
	} else {
		r.clusterName = fmt.Sprintf("%v", yamlConfig["metadata"].(map[string]interface{})["name"])
	}
	if r.flag.Organization != "" {
		yamlConfig["metadata"].(map[string]interface{})["organization"] = r.flag.Organization
		r.clusterOrganization = fmt.Sprintf("%v", yamlConfig["metadata"].(map[string]interface{})["organization"])
	} else {
		r.clusterOrganization = fmt.Sprintf("%v", yamlConfig["metadata"].(map[string]interface{})["organization"])
	}

	return yamlConfig, nil
}

// getDefaultAppYAML reads the given cluster yaml file
// and overwrite some metadata fields if --name and/or --organization is set
func (r *runner) getDefaultAppYAML() (map[string]interface{}, error) {
	yamlFile, err := os.ReadFile(r.flag.DefaultAppConfigYAML)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var yamlConfig map[string]interface{}

	err = yaml.Unmarshal(yamlFile, &yamlConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	yamlConfig["clusterName"] = r.clusterName
	yamlConfig["organization"] = r.clusterOrganization

	return yamlConfig, nil
}
