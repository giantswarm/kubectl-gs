package cluster

import (
	"context"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"
	"k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/kubectl-gs/cmd/template/cluster/provider"
	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/pkg/id"
	"github.com/giantswarm/kubectl-gs/pkg/installation"
	"github.com/giantswarm/kubectl-gs/pkg/labels"
)

type runner struct {
	flag   *flag
	logger micrologger.Logger
	stdout io.Writer
	stderr io.Writer
}

func (r *runner) Run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Sorting is required before validation for uniqueness.
	sort.Slice(r.flag.ControlPlaneAZ, func(i, j int) bool {
		return r.flag.ControlPlaneAZ[i] < r.flag.ControlPlaneAZ[j]
	})

	commonConfig := commonconfig.New(r.flag.config)
	k8sClient, err := commonConfig.GetClient(r.logger)
	if err != nil {
		return microerror.Mask(err)
	}

	var ingress v1beta1.Ingress
	err = k8sClient.CtrlClient().Get(ctx, client.ObjectKey{Name: "happa", Namespace: "giantswarm"}, &ingress)
	if err != nil {
		return microerror.Mask(err)
	}

	installationService, err := installation.New(ctx, ingress.Spec.Rules[0].Host)
	if err != nil {
		return microerror.Mask(err)
	}

	r.flag.Provider = installationService.Provider

	err = r.flag.Validate()
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.run(ctx, cmd, args)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *runner) run(ctx context.Context, cmd *cobra.Command, args []string) error {
	var err error

	var config provider.ClusterConfig
	{
		config = provider.ClusterConfig{
			ControlPlaneAZ:    r.flag.ControlPlaneAZ,
			Description:       r.flag.Description,
			KubernetesVersion: r.flag.KubernetesVersion,
			Name:              r.flag.Name,
			Organization:      r.flag.Organization,
			PodsCIDR:          r.flag.PodsCIDR,
			ReleaseVersion:    r.flag.Release,
			Namespace:         metav1.NamespaceDefault,

			App:       r.flag.App,
			AWS:       r.flag.AWS,
			OIDC:      r.flag.OIDC,
			OpenStack: r.flag.OpenStack,
		}

		if config.Name == "" {
			config.Name = id.Generate()
		}

		// Remove leading 'v' from release flag input.
		config.ReleaseVersion = strings.TrimLeft(config.ReleaseVersion, "v")

		config.Labels, err = labels.Parse(r.flag.Label)
		if err != nil {
			return microerror.Mask(err)
		}

		if r.flag.Provider != key.ProviderAWS {
			config.Namespace = key.OrganizationNamespaceFromName(config.Organization)
		}
	}

	commonConfig := commonconfig.New(r.flag.config)
	c, err := commonConfig.GetClient(r.logger)
	if err != nil {
		return microerror.Mask(err)
	}

	var output *os.File
	{
		if r.flag.Output == "" {
			output = os.Stdout
		} else {
			f, err := os.Create(r.flag.Output)
			if err != nil {
				return microerror.Mask(err)
			}
			defer f.Close()

			output = f
		}
	}

	switch r.flag.Provider {
	case key.ProviderAWS:
		err = provider.WriteAWSTemplate(ctx, c, output, config)
		if err != nil {
			return microerror.Mask(err)
		}
	case key.ProviderAzure:
		err = provider.WriteAzureTemplate(ctx, c, output, config)
		if err != nil {
			return microerror.Mask(err)
		}
	case key.ProviderOpenStack:
		err = provider.WriteOpenStackTemplate(ctx, c, output, config)
		if err != nil {
			return microerror.Mask(err)
		}
	case key.ProviderVSphere:
		err = provider.WriteVSphereTemplate(ctx, c, output, config)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
