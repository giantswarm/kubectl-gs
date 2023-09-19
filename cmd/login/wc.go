package login

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"

	"github.com/giantswarm/kubectl-gs/v2/internal/key"
	"github.com/giantswarm/kubectl-gs/v2/pkg/data/domain/clientcert"
	"github.com/giantswarm/kubectl-gs/v2/pkg/data/domain/cluster"
	"github.com/giantswarm/kubectl-gs/v2/pkg/data/domain/organization"
	"github.com/giantswarm/kubectl-gs/v2/pkg/data/domain/release"
	"github.com/giantswarm/kubectl-gs/v2/pkg/kubeconfig"
)

const (
	newClusterMaxAge = 30 * time.Minute
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

	return key.CertOperatorVersionKubeconfig, nil
}

// used only if both MC and WC are specified on command line
func (r *runner) handleWCKubeconfig(ctx context.Context) error {
	provider, err := r.commonConfig.GetProviderFromConfig(ctx, "")
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
	contextName, contextExists, err := r.createClusterKubeconfig(ctx, client, provider)
	if err != nil {
		if IsClusterAPINotReady(err) {
			fmt.Fprintf(r.stdout, "\nCould not create a context for workload cluster %s, as the cluster's API server endpoint is not known yet.\n", r.flag.WCName)
			fmt.Fprintf(r.stdout, "If the cluster has been created recently, please wait for a few minutes and try again.\n")
		} else if IsClusterAPINotKnown(err) {
			fmt.Fprintf(r.stdout, "\nCould not create a context for workload cluster %s, as the cluster's API server endpoint is not known.\n", r.flag.WCName)
			fmt.Fprintf(r.stdout, "Since the cluster has been created a while ago, this appears to be a problem with cluster creation. Please contact Giant Swarm's support.\n")
		}
		return microerror.Mask(err)
	}

	if r.loginOptions.selfContainedWC {
		fmt.Fprintf(r.stdout, "A new kubectl context has been created named '%s' and stored in '%s'. You can select this context like this:\n\n", contextName, r.flag.SelfContained)
		fmt.Fprintf(r.stdout, "  kubectl cluster-info --kubeconfig %s \n", r.flag.SelfContained)
	} else if !r.loginOptions.switchToWCContext {
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

// used only if both MC and WC are specified on command line
func (r *runner) createClusterKubeconfig(ctx context.Context, client k8sclient.Interface, provider string) (contextName string, contextExists bool, err error) {
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

	// for EKS the kubeconfig cannot be client-cert as it uses aws authentication
	// for the rest of WC cluster client-cert kubeconfig will be generated
	if isEKS(c.Cluster) {
		contextName, contextExists, err = r.createEKSKubeconfig(ctx, client, c)
		if err != nil {
			return "", false, microerror.Mask(err)
		}
	} else {
		contextName, contextExists, err = r.createCertKubeconfig(ctx, c, services, provider)
		if err != nil {
			return "", false, microerror.Mask(err)
		}
	}

	return contextName, contextExists, nil
}

// used only if both MC and WC are specified on command line
func (r *runner) createCertKubeconfig(ctx context.Context, c *cluster.Cluster, services serviceSet, provider string) (string, bool, error) {
	certOperatorVersion, err := r.getCertOperatorVersion(c, provider, services, ctx)
	if err != nil {
		return "", false, microerror.Mask(err)
	}

	clusterBasePath, err := getWCBasePath(r.commonConfig.GetConfigAccess(), provider, r.loginOptions.contextOverride)
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

	if c.Cluster.Spec.ControlPlaneEndpoint.Host == "" || c.Cluster.Spec.ControlPlaneEndpoint.Port == 0 {
		if c.Cluster.ObjectMeta.CreationTimestamp.Time.Before(time.Now().Add(-newClusterMaxAge)) {
			return "", false, microerror.Maskf(clusterAPINotKnownError, "API for cluster '%s' is not known", r.flag.WCName)
		}
		return "", false, microerror.Maskf(clusterAPINotReadyError, "API for cluster '%s' is not ready yet", r.flag.WCName)
	}

	clusterServer := fmt.Sprintf("https://%s:%d", c.Cluster.Spec.ControlPlaneEndpoint.Host, c.Cluster.Spec.ControlPlaneEndpoint.Port)

	// When CAPA or CGP we need our custom DNS record for the k8s api, rather than the value found in the CAPI CRs so that our connection through the VPN works.
	if provider == key.ProviderCAPA || provider == key.ProviderGCP {
		clusterServer = fmt.Sprintf("https://api.%s.%s:%d", c.Cluster.Name, clusterBasePath, c.Cluster.Spec.ControlPlaneEndpoint.Port)
	}

	credentialConfig := clientCertCredentialConfig{
		clusterID:     r.flag.WCName,
		certCRT:       secret.Data[credentialKeyCertCRT],
		certKey:       secret.Data[credentialKeyCertKey],
		certCA:        secret.Data[credentialKeyCertCA],
		clusterServer: clusterServer,
		filePath:      r.flag.SelfContained,
		loginOptions:  r.loginOptions,
		proxy:         r.flag.Proxy,
		proxyPort:     r.flag.ProxyPort,
	}

	contextName, contextExists, err := r.storeWCClientCertCredentials(credentialConfig)
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

func (r *runner) createEKSKubeconfig(ctx context.Context, k8sClient k8sclient.Interface, c *cluster.Cluster) (string, bool, error) {
	caData, err := fetchEKSCAData(ctx, k8sClient, c.Cluster.Name, c.Cluster.Namespace)
	if err != nil {
		return "", false, microerror.Mask(err)
	}

	region, err := fetchEKSRegion(ctx, k8sClient, c.Cluster.Name, c.Cluster.Namespace)
	if err != nil {
		return "", false, microerror.Mask(err)
	}

	eksClusterConfig := eksClusterConfig{
		awsProfileName:       r.flag.AWSProfile,
		clusterName:          c.Cluster.Name,
		certCA:               caData,
		controlPlaneEndpoint: c.Cluster.Spec.ControlPlaneEndpoint.Host,
		filePath:             r.flag.SelfContained,
		region:               region,
		loginOptions:         r.loginOptions,
	}

	contextName, contextExists, err := r.storeAWSIAMCredentials(eksClusterConfig)
	if err != nil {
		return "", false, microerror.Mask(err)
	}

	fmt.Fprint(r.stdout, color.GreenString("\nCreated aws IAM based kubeconfig for EKS workload cluster '%s'.\n", r.flag.WCName))
	fmt.Fprint(r.stdout, color.YellowString("\nRemember to have valid and active AWS credentials that have 'eks:GetToken' permissions on the EKS cluster resource in order to use the kubeconfig.\n\n"))

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

func (r *runner) storeWCClientCertCredentials(c clientCertCredentialConfig) (string, bool, error) {
	k8sConfigAccess := r.commonConfig.GetConfigAccess()
	// Store client certificate credential either into the current kubeconfig or a self-contained file if a path is given.
	if r.loginOptions.selfContainedWC && c.filePath != "" {
		return printWCClientCertCredentials(k8sConfigAccess, r.fs, c, r.loginOptions.contextOverride)
	}
	return storeWCClientCertCredentials(k8sConfigAccess, c, r.loginOptions.contextOverride)
}

func (r *runner) storeAWSIAMCredentials(c eksClusterConfig) (string, bool, error) {
	k8sConfigAccess := r.commonConfig.GetConfigAccess()
	if r.loginOptions.selfContained && c.filePath != "" {
		return printWCAWSIamCredentials(k8sConfigAccess, r.fs, c, r.loginOptions.contextOverride)
	}
	return storeWCAWSIAMKubeconfig(k8sConfigAccess, c, r.loginOptions.contextOverride)
}

func getWCBasePath(k8sConfigAccess clientcmd.ConfigAccess, provider string, currentContext string) (string, error) {
	config, err := k8sConfigAccess.GetStartingConfig()
	if err != nil {
		return "", microerror.Mask(err)
	}

	if currentContext == "" {
		currentContext = config.CurrentContext
	}
	clusterServer, _ := kubeconfig.GetClusterServer(config, currentContext)

	// Ensure any trailing ports are trimmed.
	reg := regexp.MustCompile(`:[0-9]+$`)
	clusterServer = reg.ReplaceAllString(clusterServer, "")

	// Sanitize the cluster server address and extract base domain name
	clusterServer = strings.TrimPrefix(clusterServer, "https://")

	clusterServer = strings.TrimPrefix(clusterServer, "internal-")

	clusterServer = strings.TrimPrefix(clusterServer, "api.")

	// pure CAPI clusters have an api.$INSTALLATION prefix
	if key.IsPureCAPIProvider(provider) {
		if _, contextType := kubeconfig.IsKubeContext(currentContext); contextType == kubeconfig.ContextTypeMC {
			clusterName := kubeconfig.GetCodeNameFromKubeContext(currentContext)
			clusterServer = strings.TrimPrefix(clusterServer, clusterName+".")
		} else {
			return "", microerror.Maskf(selectedContextNonCompatibleError, "Can not parse MC codename from context %v. Valid MC context schema is `gs-$CODENAME`.", currentContext)
		}
	}

	return strings.TrimPrefix(clusterServer, "g8s."), nil
}

func isEKS(c *capi.Cluster) bool {
	return c.Spec.InfrastructureRef != nil && c.Spec.InfrastructureRef.Kind == "AWSManagedCluster"
}
