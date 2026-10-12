package cluster

import (
	"context"
	"fmt"
	"os"

	"github.com/giantswarm/k8sclient/v8/pkg/k8sclient"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/v6/cmd/template/cluster/common"
	"github.com/giantswarm/kubectl-gs/v6/cmd/template/cluster/flags"
	"github.com/giantswarm/kubectl-gs/v6/internal/key"
)

const (
	defaultOIDCUsernameClaim = "email"
	defaultOIDCGroupsClaim   = "groups"
)

// oidcProviders render global.controlPlane.oidc into the API server of a
// kubeadm control plane. EKS and AKS run a managed control plane and are left
// unchanged.
var oidcProviders = map[string]bool{
	key.ProviderCAPA:          true,
	key.ProviderCAPZ:          true,
	key.ProviderVSphere:       true,
	key.ProviderCloudDirector: true,
}

// oidcValues returns the OIDC issuer the workload cluster's API server
// trusts: by default the one the management cluster's own API server trusts
// (its Dex), each --oidc-* flag given overriding its field. With
// --oidc-issuer-url the management cluster is not asked. With --disable-oidc,
// and for providers without the setting, it returns nil.
func (r *runner) oidcValues(ctx context.Context, client k8sclient.Interface) (*common.OIDCValues, error) {
	if !oidcProviders[r.flag.Provider] || r.flag.DisableOIDC {
		return nil, nil
	}

	given := r.flag.OIDC
	values := common.OIDCValues{
		IssuerURL:     given.IssuerURL,
		ClientID:      given.ClientID,
		UsernameClaim: common.DefaultTo(given.UsernameClaim, defaultOIDCUsernameClaim),
		GroupsClaim:   common.DefaultTo(given.GroupsClaim, defaultOIDCGroupsClaim),
	}

	if given.IssuerURL == "" {
		mc, err := common.ManagementClusterOIDC(ctx, client.CtrlClient(), r.flag.ManagementCluster)
		if err != nil {
			if r.noKubeContext() {
				err = fmt.Errorf("no kubeconfig context is selected: %w", err)
			}
			return nil, fmt.Errorf("%w\nThe workload cluster trusts the management cluster's OIDC issuer (Dex) by default. Name an issuer with --oidc-issuer-url and --oidc-client-id, or leave the OIDC settings out with --%s",
				err, flags.FlagDisableOIDC)
		}

		values = common.OIDCValues{
			IssuerURL:     mc.IssuerURL,
			ClientID:      common.DefaultTo(given.ClientID, mc.ClientID),
			UsernameClaim: common.DefaultTo(given.UsernameClaim, mc.UsernameClaim),
			GroupsClaim:   common.DefaultTo(given.GroupsClaim, mc.GroupsClaim),
		}
	}

	if given.CAFile != "" {
		caPem, err := os.ReadFile(given.CAFile)
		if err != nil {
			return nil, microerror.Maskf(invalidConfigError, "cannot read --oidc-ca-file: %s", err)
		}
		values.CAPem = string(caPem)
	}

	return &values, nil
}

// noKubeContext tells whether the kubeconfig selects no context, so that the
// client fell back to a default server.
func (r *runner) noKubeContext() bool {
	if r.commonConfig == nil {
		return false
	}
	name, err := r.commonConfig.GetCurrentContextName()
	return err == nil && name == ""
}
