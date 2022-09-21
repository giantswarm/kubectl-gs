package login

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/fatih/color"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/clientcert"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/cluster"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/organization"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/release"
	"github.com/giantswarm/kubectl-gs/pkg/kubeconfig"
)

func (r *runner) getServiceSet(client k8sclient.Interface) (serviceSet, error) {
	var err error

	var clientCertService clientcert.Interface
	{
		serviceConfig := clientcert.Config{
			Client: client.CtrlClient(),
		}
		clientCertService, err = clientcert.New(serviceConfig)
		if err != nil {
			return serviceSet{}, microerror.Mask(err)
		}
	}

	var organizationService organization.Interface
	{
		serviceConfig := organization.Config{
			Client: client,
		}
		organizationService, err = organization.New(serviceConfig)
		if err != nil {
			return serviceSet{}, microerror.Mask(err)
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
			return serviceSet{}, microerror.Mask(err)
		}
	}

	return serviceSet{
		clientCertService:   clientCertService,
		organizationService: organizationService,
		clusterService:      clusterService,
		releaseService:      releaseService,
	}, nil
}

func (r *runner) getCluster(ctx context.Context, services serviceSet, provider string) (*cluster.Cluster, error) {
	var err error
	var c *cluster.Cluster

	var namespaces []string
	if len(r.flag.WCOrganization) > 0 {
		orgNamespace, err := getOrganizationNamespace(ctx, services.organizationService, r.flag.WCOrganization)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		namespaces = append(namespaces, orgNamespace)
	} else {
		namespaces, err = getAllOrganizationNamespaces(ctx, services.organizationService)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	if r.flag.WCInsecureNamespace {
		namespaces = append(namespaces, "default")
	}

	c, err = findCluster(ctx, services.clusterService, services.organizationService, provider, r.flag.WCName, namespaces...)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	return c, nil
}

func (r *runner) getCertOperatorVersion(c *cluster.Cluster, provider string, services serviceSet, ctx context.Context) (string, error) {
	// Pure CAPI providers (e.g. openstack) do not have release CRs and labels nor do they run cert-operator. so we return an empty string.
	// In this case wecreate the client certificate using the MC PKI
	if key.IsPureCAPIProvider(provider) {
		return "", nil
	}

	releaseVersion, err := getClusterReleaseVersion(c, provider)
	if err != nil {
		return "", microerror.Mask(err)
	}
	certOperatorVersion, err := getCertOperatorVersion(ctx, services.releaseService, releaseVersion)
	// If the release does not contain cert-operator anymore (e.g. in CAPI versions) we return an empty string
	// In this case we try to create the client certificate using the MC PKI
	if IsMissingComponent(err) {
		return "", nil
	} else if err != nil {
		return "", microerror.Mask(err)
	}
	return certOperatorVersion, nil
}

func (r *runner) handleWCClientCert(ctx context.Context) error {

	provider, err := r.commonConfig.GetProvider()
	if err != nil {
		return microerror.Mask(err)
	}

	var client k8sclient.Interface
	{
		client, err = r.commonConfig.GetClient(r.logger)
		if err != nil {
			return microerror.Mask(err)
		}
	}
	// At the moment, the only available login option for WC is client cert
	contextName, contextExists, err := r.createClusterClientCert(ctx, client, provider)
	if err != nil {
		return microerror.Mask(err)
	}

	if r.loginOptions.selfContainedClientCert {
		fmt.Fprintf(r.stdout, "A new kubectl context has been created named '%s' and stored in '%s'. You can select this context like this:\n\n", contextName, r.flag.SelfContained)
		fmt.Fprintf(r.stdout, "  kubectl cluster-info --kubeconfig %s \n", r.flag.SelfContained)
	} else if !r.loginOptions.switchToClientCertContext {
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

func (r *runner) createClusterClientCert(ctx context.Context, client k8sclient.Interface, provider string) (contextName string, contextExists bool, err error) {
	err = validateProvider(provider)
	if err != nil {
		return "", false, microerror.Mask(err)
	}

	services, err := r.getServiceSet(client)
	if err != nil {
		return "", false, microerror.Mask(err)
	}

	c, err := r.getCluster(ctx, services, provider)
	if err != nil {
		return "", false, microerror.Mask(err)
	}

	certOperatorVersion, err := r.getCertOperatorVersion(c, provider, services, ctx)
	if err != nil {
		return "", false, microerror.Mask(err)
	}

	clusterBasePath, err := getWCBasePath(r.commonConfig.GetConfigAccess(), provider)
	if err != nil {
		return "", false, microerror.Mask(err)
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
		cnPrefix:            r.flag.WCCertCNPrefix,
	}

	clientCertResource, secret, err := r.getCredentials(ctx, services.clientCertService, certConfig)
	if err != nil {
		return "", false, microerror.Mask(err)
	}

	clusterServer := fmt.Sprintf("https://%s:%d", c.Cluster.Spec.ControlPlaneEndpoint.Host, c.Cluster.Spec.ControlPlaneEndpoint.Port)

	// If the control plane host is an IP address then it is a CAPI cluster and
	// we need to manually set the cluster server to the correct domain
	if net.ParseIP(c.Cluster.Spec.ControlPlaneEndpoint.Host) != nil {
		clusterServer = fmt.Sprintf("https://api.%s.%s:%d", c.Cluster.Name, clusterBasePath, c.Cluster.Spec.ControlPlaneEndpoint.Port)
	}

	credentialConfig := credentialConfig{
		clusterID:     r.flag.WCName,
		certCRT:       secret.Data[credentialKeyCertCRT],
		certKey:       secret.Data[credentialKeyCertKey],
		certCA:        secret.Data[credentialKeyCertCA],
		clusterServer: clusterServer,
		filePath:      r.flag.SelfContained,
		loginOptions:  r.loginOptions,
	}

	contextName, contextExists, err = r.storeWCClientCertCredentials(credentialConfig)
	if err != nil {
		return "", false, microerror.Mask(err)
	}

	// We only clean up if a clientCertResource has been created (non CAPI case)
	if clientCertResource != nil {
		// Cleaning up leftover resources.
		err = cleanUpClientCertResources(ctx, services.clientCertService, clientCertResource)
		if err != nil {
			return "", false, microerror.Mask(err)
		}
	}

	fmt.Fprint(r.stdout, color.GreenString("\nCreated client certificate for workload cluster '%s'.\n", r.flag.WCName))
	return contextName, contextExists, nil
}

func (r *runner) getCredentials(ctx context.Context, clientCertService clientcert.Interface, config clientCertConfig) (*clientcert.ClientCert, *v1.Secret, error) {
	var clientCertsecret *v1.Secret
	var clientCert *clientcert.ClientCert
	var err error

	// If cert-operator is not running (as in CAPI) we attempt to use the MC PKI to create a certificate
	if config.certOperatorVersion == "" {
		// Retrieve the WC CA-secret.
		ca, err := clientCertService.GetCredential(ctx, config.clusterNamespace, config.clusterName+"-ca")
		if err != nil {
			return nil, nil, microerror.Mask(err)
		}
		clientCertsecret, err = generateCredential(ctx, ca, config)
		if err != nil {
			return nil, nil, microerror.Mask(err)
		}
		return nil, clientCertsecret, nil
	}

	clientCert, err = createCert(ctx, clientCertService, config)
	if err != nil {
		return nil, nil, microerror.Mask(err)
	}
	// apply the certConfig
	err = clientCertService.Create(ctx, clientCert)
	if err != nil {
		return nil, nil, microerror.Mask(err)
	}
	// Retrieve client certificate credential.
	clientCertsecret, err = fetchCredential(ctx, config.provider, clientCertService, clientCert)
	if err != nil {
		return nil, nil, microerror.Mask(err)
	}
	return clientCert, clientCertsecret, nil
}

func (r *runner) storeWCClientCertCredentials(c credentialConfig) (string, bool, error) {
	k8sConfigAccess := r.commonConfig.GetConfigAccess()
	// Store client certificate credential either into the current kubeconfig or a self-contained file if a path is given.
	if r.loginOptions.selfContainedClientCert && c.filePath != "" {
		return printWCClientCertCredentials(k8sConfigAccess, r.fs, c)
	}
	return storeWCClientCertCredentials(k8sConfigAccess, r.fs, c)
}

func getWCBasePath(k8sConfigAccess clientcmd.ConfigAccess, provider string) (string, error) {
	config, err := k8sConfigAccess.GetStartingConfig()
	if err != nil {
		return "", microerror.Mask(err)
	}

	clusterServer, _ := kubeconfig.GetClusterServer(config, config.CurrentContext)

	// Ensure any trailing ports are trimmed.
	reg := regexp.MustCompile(`:[0-9]+$`)
	clusterServer = reg.ReplaceAllString(clusterServer, "")

	// Sanitize the cluster server address and extract base domain name
	clusterServer = strings.TrimPrefix(clusterServer, "https://")

	clusterServer = strings.TrimPrefix(clusterServer, "internal-")

	clusterServer = strings.TrimPrefix(clusterServer, "api.")

	// pure CAPI clusters have an api.$INSTALLATION prefix
	if key.IsPureCAPIProvider(provider) {
		if _, contextType := kubeconfig.IsKubeContext(config.CurrentContext); contextType == kubeconfig.ContextTypeMC {
			clusterName := kubeconfig.GetCodeNameFromKubeContext(config.CurrentContext)
			clusterServer = strings.TrimPrefix(clusterServer, clusterName+".")
		} else {
			return "", microerror.Maskf(selectedContextNonCompatibleError, "Can not parse MC codename from context %v. Valid MC context schema is `gs-$CODENAME`.", config.CurrentContext)
		}
	}

	return strings.TrimPrefix(clusterServer, "g8s."), nil
}
