package login

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/giantswarm/kubectl-gs/v6/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/v6/pkg/data/domain/organization"
)

func TestCanSkipMCLogin(t *testing.T) {
	testCases := []struct {
		name          string
		contexts      []string
		current       string
		args          []string
		flags         flag
		expectSkipped bool
	}{
		{
			name:          "non-gs current context without installation argument",
			contexts:      []string{"teleport-codename"},
			current:       "teleport-codename",
			flags:         flag{WCName: "staging"},
			expectSkipped: true,
		},
		{
			name:          "non-gs current context, installation context exists",
			contexts:      []string{"teleport-codename-staging", "gs-codename"},
			current:       "teleport-codename-staging",
			args:          []string{"codename"},
			flags:         flag{WCName: "staging"},
			expectSkipped: false,
		},
		{
			name:          "non-gs current context, installation clientcert context exists",
			contexts:      []string{"teleport-codename-staging", "gs-codename-clientcert"},
			current:       "teleport-codename-staging",
			args:          []string{"codename"},
			flags:         flag{WCName: "staging"},
			expectSkipped: false,
		},
		{
			name:          "non-gs current context, installation argument is a context name",
			contexts:      []string{"teleport-codename-staging", "gs-codename"},
			current:       "teleport-codename-staging",
			args:          []string{"gs-codename"},
			flags:         flag{WCName: "staging"},
			expectSkipped: false,
		},
		{
			name:          "non-gs current context, no context for installation argument",
			contexts:      []string{"teleport-codename"},
			current:       "teleport-codename",
			args:          []string{"codename"},
			flags:         flag{WCName: "staging"},
			expectSkipped: true,
		},
		{
			name:          "gs current context",
			contexts:      []string{"gs-codename"},
			current:       "gs-codename",
			args:          []string{"codename"},
			flags:         flag{WCName: "staging"},
			expectSkipped: false,
		},
		{
			name:          "issuer and client ID flags without installation argument",
			contexts:      []string{"gs-codename"},
			current:       "gs-codename",
			flags:         flag{WCName: "staging", WCOIDCIssuer: "https://dex.example.com", WCOIDCClientID: "id"},
			expectSkipped: true,
		},
		{
			name:          "issuer and client ID flags, no context for installation argument",
			contexts:      []string{"teleport-codename"},
			current:       "teleport-codename",
			args:          []string{"codename"},
			flags:         flag{WCName: "staging", WCOIDCIssuer: "https://dex.example.com", WCOIDCClientID: "id"},
			expectSkipped: true,
		},
		{
			name:          "issuer and client ID flags, installation context exists",
			contexts:      []string{"teleport-codename-staging", "gs-codename"},
			current:       "teleport-codename-staging",
			args:          []string{"codename"},
			flags:         flag{WCName: "staging", WCOIDCIssuer: "https://dex.example.com", WCOIDCClientID: "id"},
			expectSkipped: false,
		},
		{
			name:          "management cluster login",
			contexts:      []string{"teleport-codename"},
			current:       "teleport-codename",
			args:          []string{"codename"},
			expectSkipped: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := clientcmdapi.NewConfig()
			for _, name := range tc.contexts {
				config.Clusters[name] = &clientcmdapi.Cluster{Server: "https://" + name}
				config.AuthInfos[name] = &clientcmdapi.AuthInfo{Token: "token"}
				config.Contexts[name] = &clientcmdapi.Context{Cluster: name, AuthInfo: name}
			}
			config.CurrentContext = tc.current

			cf := genericclioptions.NewConfigFlags(true)
			cf.KubeConfig = ptr.To(fmt.Sprintf("%s/config.yaml", t.TempDir()))
			r := runner{
				commonConfig: commonconfig.New(cf),
				flag:         &tc.flags,
			}
			if err := clientcmd.ModifyConfig(r.commonConfig.GetConfigAccess(), *config, false); err != nil {
				t.Fatal(err)
			}
			args := tc.args
			r.setLoginOptions(context.Background(), &args)

			if skipped := r.canSkipMCLogin(tc.args); skipped != tc.expectSkipped {
				t.Fatalf("canSkipMCLogin() = %t, want %t", skipped, tc.expectSkipped)
			}
		})
	}
}

type organizationServiceFunc func(context.Context, organization.GetOptions) (organization.Resource, error)

func (f organizationServiceFunc) Get(ctx context.Context, o organization.GetOptions) (organization.Resource, error) {
	return f(ctx, o)
}

func TestGetClusterOrganizationAPIMissing(t *testing.T) {
	gv := schema.GroupVersion{Group: "security.giantswarm.io", Version: "v1alpha1"}
	discoveryErr := apiutil.ErrResourceDiscoveryFailed{
		gv: &meta.NoResourceMatchError{PartialResource: gv.WithResource("")},
	}

	r := runner{
		flag:         &flag{WCName: "staging"},
		loginOptions: LoginOptions{originContext: "teleport-codename-staging"},
	}
	services := serviceSet{
		organizationService: organizationServiceFunc(func(context.Context, organization.GetOptions) (organization.Resource, error) {
			return nil, microerror.Mask(&discoveryErr)
		}),
	}

	_, err := r.getCluster(context.Background(), services, "capi")
	if microerror.Cause(err) != organizationAPINotFoundError {
		t.Fatalf("expected organizationAPINotFoundError, got %v", err)
	}
	for _, want := range []string{`"teleport-codename-staging"`, "--organization"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %s", err.Error(), want)
		}
	}
}
