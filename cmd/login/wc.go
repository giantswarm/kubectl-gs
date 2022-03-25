package login

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/fatih/color"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/clientcert"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/cluster"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/organization"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/release"
	"github.com/giantswarm/kubectl-gs/pkg/kubeconfig"
)

func (r *runner) handleWCLogin(ctx context.Context) error {
	// At the moment, the only available login option for WC is client cert
	return r.handleWCClientCert(ctx)
}

func (r *runner) handleWCClientCert(ctx context.Context) error {
	var err error

	config := commonconfig.New(r.flag.config)

	provider, err := config.GetProvider()
	if err != nil {
		return microerror.Mask(err)
	}

	var client k8sclient.Interface
	{
		client, err = config.GetClient(r.logger)
		if err != nil {
			return microerror.Mask(err)
		}
	}
	// At the moment, the only available login option for WC is client cert
	return r.createClusterClientCert(ctx, client, provider)
}

func (r *runner) createClusterClientCert(ctx context.Context, client k8sclient.Interface, provider string) error {
	var err error

	err = validateProvider(provider)
	if err != nil {
		return microerror.Mask(err)
	}

	var clientCertService clientcert.Interface
	{
		serviceConfig := clientcert.Config{
			Client: client.CtrlClient(),
		}
		clientCertService, err = clientcert.New(serviceConfig)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var organizationService organization.Interface
	{
		serviceConfig := organization.Config{
			Client: client.CtrlClient(),
		}
		organizationService, err = organization.New(serviceConfig)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var clusterService cluster.Interface
	{
		serviceConfig := cluster.Config{
			Client: client.CtrlClient(),
		}
		clusterService = cluster.New(serviceConfig)
	}

	var releaseService release.Interface
	{
		serviceConfig := release.Config{
			Client: client.CtrlClient(),
		}
		releaseService, err = release.New(serviceConfig)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var c *cluster.Cluster
	{
		var namespaces []string
		if len(r.flag.WCOrganization) > 0 {
			orgNamespace, err := getOrganizationNamespace(ctx, organizationService, r.flag.WCOrganization)
			if err != nil {
				return microerror.Mask(err)
			}

			namespaces = append(namespaces, orgNamespace)
		} else {
			namespaces, err = getAllOrganizationNamespaces(ctx, organizationService)
			if err != nil {
				return microerror.Mask(err)
			}
		}

		if r.flag.WCInsecureNamespace {
			namespaces = append(namespaces, "default")
		}

		c, err = findCluster(ctx, clusterService, organizationService, provider, r.flag.WCName, namespaces...)
		if err != nil {
			return microerror.Mask(err)
		}
	}
	var releaseVersion, certOperatorVersion string
	if provider != key.ProviderOpenStack {
		releaseVersion, err = getClusterReleaseVersion(c, provider, r.flag.WCInsecureNamespace)
		if err != nil {
			return microerror.Mask(err)
		}

		certOperatorVersion, err = getCertOperatorVersion(ctx, releaseService, releaseVersion)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	clusterBasePath, err := getClusterBasePath(r.k8sConfigAccess)
	if err != nil {
		return microerror.Mask(err)
	}

	certConfig := clientCertConfig{
		provider:            provider,
		clusterName:         r.flag.WCName,
		clusterNamespace:    c.Cluster.GetNamespace(),
		organizationName:    r.flag.WCOrganization,
		ttl:                 r.flag.WCCertTTL,
		groups:              r.flag.WCCertGroups,
		clusterBasePath:     clusterBasePath,
		certOperatorVersion: certOperatorVersion,
	}

	clientCertResource, secret, err := r.getCredentials(ctx, clientCertService, certConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	serverAddress := getServerAddress(clientCertResource.CertConfig.Spec.Cert.ClusterID, clusterBasePath, provider)

	// Store client certificate credential either into the current kubeconfig or a self-contained file if a path is given.
	var contextExists bool
	var contextName string
	if r.loginOptions.selfContainedWC {
		contextName, contextExists, err = printWCCredentials(r.k8sConfigAccess, r.fs, r.flag.SelfContained, clientCertResource, secret, serverAddress, r.loginOptions)
		if err != nil {
			return microerror.Mask(err)
		}
	} else {
		contextName, contextExists, err = storeWCCredentials(r.k8sConfigAccess, r.fs, clientCertResource, secret, serverAddress, r.loginOptions)
		if err != nil {
			return microerror.Mask(err)
		}
	}
	if provider != key.ProviderOpenStack {
		// Cleaning up leftover resources.
		err = cleanUpClientCertResources(ctx, clientCertService, clientCertResource)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	fmt.Fprint(r.stdout, color.GreenString("\nCreated client certificate for workload cluster '%s'.\n", r.flag.WCName))

	if r.loginOptions.selfContainedWC {
		fmt.Fprintf(r.stdout, "A new kubectl context has been created named '%s' and stored in '%s'. You can select this context like this:\n\n", contextName, r.flag.SelfContained)
		fmt.Fprintf(r.stdout, "  kubectl cluster-info --kubeconfig %s \n", r.flag.SelfContained)
	} else if !r.loginOptions.switchToWCcontext {
		fmt.Fprintf(r.stdout, "A new kubectl context has been created named '%s'. To switch back to this context later, use this command:\n\n", contextName)
		fmt.Fprintf(r.stdout, "  kubectl config use-context %s\n", contextName)
	} else if contextExists {
		fmt.Fprintf(r.stdout, "Switched to context '%s'.\n\n", contextName)
		fmt.Fprintf(r.stdout, "To switch back to this context later, use this command:\n\n")
		fmt.Fprintf(r.stdout, "  kubectl config use-context %s\n", contextName)
	} else {
		fmt.Fprintf(r.stdout, "A new kubectl context has been created named '%s' and selected. To switch back to this context later, use this command:\n\n", contextName)
		fmt.Fprintf(r.stdout, "  kubectl config use-context %s\n", contextName)
	}

	return nil
}

func (r *runner) getCredentials(ctx context.Context, clientCertService clientcert.Interface, config clientCertConfig) (*clientcert.ClientCert, *v1.Secret, error) {
	var secret *v1.Secret
	var err error

	clientCert, err := createCert(ctx, clientCertService, config)
	if err != nil {
		return nil, nil, microerror.Mask(err)
	}

	if config.provider != key.ProviderOpenStack {
		// apply the certConfig
		err = clientCertService.Create(ctx, clientCert)
		if err != nil {
			return nil, nil, microerror.Mask(err)
		}
		// Retrieve client certificate credential.
		secret, err = fetchCredential(ctx, config.provider, clientCertService, clientCert)
		if err != nil {
			return nil, nil, microerror.Mask(err)
		}
	} else {
		// Retrieve the WC CA-secret.
		ca, err := clientCertService.GetCredential(ctx, clientCert.CertConfig.GetNamespace(), clientCert.CertConfig.Spec.Cert.ClusterID+"-ca")
		if err != nil {
			return nil, nil, microerror.Mask(err)
		}
		secret, err = generateCredential(ctx, ca, config)
		if err != nil {
			return nil, nil, microerror.Mask(err)
		}
	}
	return clientCert, secret, nil
}

func getClusterBasePath(k8sConfigAccess clientcmd.ConfigAccess) (string, error) {
	config, err := k8sConfigAccess.GetStartingConfig()
	if err != nil {
		return "", microerror.Mask(err)
	}

	clusterServer, _ := kubeconfig.GetClusterServer(config, config.CurrentContext)
	clusterName := kubeconfig.GetCodeNameFromKubeContext(config.CurrentContext)

	// Ensure any trailing ports are trimmed.
	reg := regexp.MustCompile(`:[0-9]+$`)
	clusterServer = reg.ReplaceAllString(clusterServer, "")

	// Some management clusters might have 'api.g8s' as prefix (example: Viking).
	clusterServer = strings.TrimPrefix(clusterServer, "https://api.g8s.")

	// Some management clusters have an api.$INSTALLATION prefix
	clusterServer = strings.TrimPrefix(clusterServer, "https://api."+clusterName+".")

	return strings.TrimPrefix(clusterServer, "https://g8s."), nil
}

func getServerAddress(clusterID string, clusterBasePath string, provider string) string {
	switch provider {
	case key.ProviderOpenStack:
		return fmt.Sprintf("https://api.%s.%s", clusterID, clusterBasePath)
	default:
		return fmt.Sprintf("https://api.%s.k8s.%s", clusterID, clusterBasePath)
	}
}
