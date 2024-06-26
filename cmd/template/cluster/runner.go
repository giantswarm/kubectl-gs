package cluster

import (
	"context"
	"errors"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/kubectl-gs/v3/cmd/template/cluster/provider"
	"github.com/giantswarm/kubectl-gs/v3/internal/key"
	"github.com/giantswarm/kubectl-gs/v3/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/v3/pkg/labels"
)

type runner struct {
	commonConfig *commonconfig.CommonConfig
	flag         *flag
	logger       micrologger.Logger
	stdout       io.Writer
	stderr       io.Writer
}

func (r *runner) Run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Sorting is required before validation for uniqueness.
	sort.Slice(r.flag.ControlPlaneAZ, func(i, j int) bool {
		return r.flag.ControlPlaneAZ[i] < r.flag.ControlPlaneAZ[j]
	})

	err := r.flag.Validate(cmd)
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
	case key.ProviderCAPZ:
		err = provider.WriteCAPZTemplate(ctx, client, output, config)
		if err != nil {
			return microerror.Mask(err)
		}
	case key.ProviderEKS:
		err = provider.WriteEKSTemplate(ctx, client, output, config)
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

	return nil
}

func (r *runner) getClusterConfig() (provider.ClusterConfig, error) {
	config := provider.ClusterConfig{
		ControlPlaneAZ:           r.flag.ControlPlaneAZ,
		ControlPlaneInstanceType: r.flag.ControlPlaneInstanceType,
		Description:              r.flag.Description,
		KubernetesVersion:        r.flag.KubernetesVersion,
		ManagementCluster:        r.flag.ManagementCluster,
		Organization:             r.flag.Organization,
		PodsCIDR:                 r.flag.PodsCIDR,
		ReleaseVersion:           r.flag.Release,
		Namespace:                metav1.NamespaceDefault,
		Region:                   r.flag.Region,
		ServicePriority:          r.flag.ServicePriority,
		PreventDeletion:          r.flag.PreventDeletion,

		App:       r.flag.App,
		AWS:       r.flag.AWS,
		Azure:     r.flag.Azure,
		GCP:       r.flag.GCP,
		OIDC:      r.flag.OIDC,
		OpenStack: r.flag.OpenStack,
		VSphere:   r.flag.VSphere,
	}

	if r.flag.GenerateName {
		generatedName, err := key.GenerateName(true)
		if err != nil {
			return provider.ClusterConfig{}, microerror.Mask(err)
		}

		config.Name = generatedName
	} else {
		config.Name = r.flag.Name
	}

	if config.Name == "" {
		return provider.ClusterConfig{}, errors.New("logic error in name assignment")
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
