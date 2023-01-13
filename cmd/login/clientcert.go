package login

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math"
	"math/big"
	"strings"
	"sync"
	"time"

	corev1alpha1 "github.com/giantswarm/apiextensions/v6/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/imdario/mergo"
	"github.com/spf13/afero"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/giantswarm/kubectl-gs/v2/internal/key"
	kgslabel "github.com/giantswarm/kubectl-gs/v2/internal/label"
	"github.com/giantswarm/kubectl-gs/v2/pkg/data/domain/clientcert"
	"github.com/giantswarm/kubectl-gs/v2/pkg/data/domain/cluster"
	"github.com/giantswarm/kubectl-gs/v2/pkg/data/domain/organization"
	"github.com/giantswarm/kubectl-gs/v2/pkg/data/domain/release"
	"github.com/giantswarm/kubectl-gs/v2/pkg/kubeconfig"
)

const (
	credentialRetryTimeout    = 1 * time.Second
	credentialMaxRetryTimeout = 10 * time.Second

	credentialKeyCertCRT = "crt"
	credentialKeyCertKey = "key"
	credentialKeyCertCA  = "ca"
)

type clientCertConfig struct {
	provider            string
	clusterName         string
	clusterNamespace    string
	organizationName    string
	ttl                 string
	groups              []string
	clusterBasePath     string
	certOperatorVersion string
	cnPrefix            string
}

type serviceSet struct {
	clientCertService   clientcert.Interface
	organizationService organization.Interface
	clusterService      cluster.Interface
	releaseService      release.Interface
}

type credentialConfig struct {
	clusterID     string
	certCRT       []byte
	certKey       []byte
	certCA        []byte
	clusterServer string
	loginOptions  LoginOptions
	filePath      string
}

func generateClientCertUID() string {
	hash := sha256.New()
	_, _ = hash.Write([]byte(time.Now().String()))

	uid := fmt.Sprintf("%x", hash.Sum(nil))

	return uid[:16]
}

func generateClientCert(config clientCertConfig) (*clientcert.ClientCert, error) {
	clientCertUID := generateClientCertUID()
	clientCertName := fmt.Sprintf("%s-%s", config.clusterName, clientCertUID)
	var clientCertCNPrefix string
	{
		if config.cnPrefix != "" {
			clientCertCNPrefix = config.cnPrefix
		} else {
			clientCertCNPrefix = clientCertUID
		}
	}
	commonName := fmt.Sprintf("%s.%s.k8s.%s", clientCertCNPrefix, config.clusterName, config.clusterBasePath)

	certConfig := &corev1alpha1.CertConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clientCertName,
			Namespace: config.clusterNamespace,
			Labels: map[string]string{
				kgslabel.CertOperatorVersion: config.certOperatorVersion,
				kgslabel.Certificate:         clientCertUID,
				label.Cluster:                config.clusterName,
				label.Organization:           config.organizationName,
			},
		},
		Spec: corev1alpha1.CertConfigSpec{
			Cert: corev1alpha1.CertConfigSpecCert{
				AllowBareDomains:    true,
				ClusterComponent:    clientCertUID,
				ClusterID:           config.clusterName,
				CommonName:          commonName,
				DisableRegeneration: true,
				Organizations:       config.groups,
				TTL:                 config.ttl,
			},
			VersionBundle: corev1alpha1.CertConfigSpecVersionBundle{
				Version: config.certOperatorVersion,
			},
		},
	}

	r := &clientcert.ClientCert{
		CertConfig: certConfig,
	}

	return r, nil
}

func createCert(ctx context.Context, clientCertService clientcert.Interface, config clientCertConfig) (*clientcert.ClientCert, error) {
	clientCert, err := generateClientCert(config)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	return clientCert, nil
}

// fetchCredential tries to fetch the client certificate credential
// for a couple of times, until the resource is fetched, or until the timeout is reached.
func fetchCredential(ctx context.Context, provider string, clientCertService clientcert.Interface, clientCert *clientcert.ClientCert) (*corev1.Secret, error) {
	var secret *corev1.Secret
	var err error

	o := func() error {
		secret, err = clientCertService.GetCredential(ctx, clientCert.CertConfig.GetNamespace(), clientCert.CertConfig.Name)
		if clientcert.IsNotFound(err) {
			// Client certificate credential has not been created yet, try again until it is.
			return microerror.Mask(err)
		} else if err != nil {
			return backoff.Permanent(microerror.Mask(err))
		}

		return nil
	}
	b := backoff.NewConstant(credentialMaxRetryTimeout, credentialRetryTimeout)

	err = backoff.Retry(o, b)
	if clientcert.IsNotFound(err) {
		if provider == key.ProviderAzure {
			// Try in default namespace for legacy azure clusters.
			secret, err = clientCertService.GetCredential(ctx, metav1.NamespaceDefault, clientCert.CertConfig.Name)
		}
		if clientcert.IsNotFound(err) {
			return nil, microerror.Maskf(credentialRetrievalTimedOut, "failed to get the client certificate credential on time")
		}
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	return secret, nil
}

func generateCredential(ctx context.Context, ca *corev1.Secret, config clientCertConfig) (*corev1.Secret, error) {
	var caPEM, certPEM, keyPEM []byte
	{
		// Get the WCs CA data
		caPEM = ca.Data["tls.crt"]
		caCert, err := getCert(caPEM)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		caPrivKeyPEM := ca.Data["tls.key"]
		caPrivKey, err := getPrivKey(caPrivKeyPEM)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		// Create the Certificate
		exp, err := time.ParseDuration(config.ttl)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		serial, err := rand.Int(rand.Reader, new(big.Int).SetInt64(math.MaxInt64))
		if err != nil {
			return nil, microerror.Maskf(invalidConfigError, "Failed to generate client certificate serial number.")
		}
		var clientCertCNPrefix string
		{
			if config.cnPrefix != "" {
				clientCertCNPrefix = config.cnPrefix
			} else {
				clientCertCNPrefix = serial.String()
			}
		}
		certificate := &x509.Certificate{
			SerialNumber:       serial,
			SignatureAlgorithm: x509.SHA256WithRSA,
			Subject: pkix.Name{
				Organization: config.groups,
				CommonName:   fmt.Sprintf("%s.%s.%s", clientCertCNPrefix, config.clusterName, config.clusterBasePath),
			},
			NotBefore:   time.Now(),
			NotAfter:    time.Now().Add(exp),
			KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement,
			ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		}

		// Create the key
		certPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		keyPEM = getPrivKeyPEM(certPrivKey)

		// Sign the certificate
		certBytes, err := x509.CreateCertificate(rand.Reader, certificate, caCert, &certPrivKey.PublicKey, caPrivKey)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		certPEM = getCertPEM(certBytes)
	}
	secret := &corev1.Secret{
		Data: map[string][]byte{
			credentialKeyCertCA:  caPEM,
			credentialKeyCertCRT: certPEM,
			credentialKeyCertKey: keyPEM,
		},
		Type: corev1.SecretTypeTLS,
	}
	return secret, nil
}

func getCertPEM(cert []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	})
}

func getCert(certPEM []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(certPEM)
	return x509.ParseCertificate(block.Bytes)
}

func getPrivKeyPEM(key *rsa.PrivateKey) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
}

func getPrivKey(keyPEM []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(keyPEM)
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

// storeWCClientCertCredentials saves the created client certificate credentials into the kubectl config.
func storeWCClientCertCredentials(k8sConfigAccess clientcmd.ConfigAccess, fs afero.Fs, c credentialConfig, mcContextName string) (string, bool, error) {
	config, err := k8sConfigAccess.GetStartingConfig()
	if err != nil {
		return "", false, microerror.Mask(err)
	}

	if mcContextName == "" {
		mcContextName = config.CurrentContext
	}
	contextName := kubeconfig.GenerateWCClientCertKubeContextName(mcContextName, c.clusterID)
	userName := fmt.Sprintf("%s-user", contextName)
	clusterName := contextName

	contextExists := false

	{
		// Create authenticated user.
		user, exists := config.AuthInfos[userName]
		if !exists {
			user = clientcmdapi.NewAuthInfo()
		}

		user.ClientCertificateData = c.certCRT
		user.ClientKeyData = c.certKey

		// Add user information to config.
		config.AuthInfos[userName] = user
	}

	{
		// Create authenticated cluster.
		cluster, exists := config.Clusters[clusterName]
		if !exists {
			cluster = clientcmdapi.NewCluster()
		}

		cluster.Server = c.clusterServer
		cluster.CertificateAuthority = ""
		cluster.CertificateAuthorityData = c.certCA

		// Add cluster configuration to config.
		config.Clusters[clusterName] = cluster
	}

	{
		// Create authenticated context.
		var context *clientcmdapi.Context
		context, contextExists = config.Contexts[contextName]
		if !contextExists {
			context = clientcmdapi.NewContext()
		}

		context.Cluster = clusterName
		context.AuthInfo = userName

		// Add context configuration to config.
		config.Contexts[contextName] = context

		// Select newly created context as current or revert to origin context if that is desired
		if c.loginOptions.switchToClientCertContext {
			config.CurrentContext = contextName
		} else if c.loginOptions.originContext != "" {
			config.CurrentContext = c.loginOptions.originContext
		}
	}

	err = clientcmd.ModifyConfig(k8sConfigAccess, *config, false)
	if err != nil {
		return "", contextExists, microerror.Mask(err)
	}

	return contextName, contextExists, nil
}

// printWCClientCertCredentials saves the created client certificate credentials into a separate kubectl config file.
func printWCClientCertCredentials(k8sConfigAccess clientcmd.ConfigAccess, fs afero.Fs, c credentialConfig, mcContextName string) (string, bool, error) {
	config, err := k8sConfigAccess.GetStartingConfig()
	if err != nil {
		return "", false, microerror.Mask(err)
	}

	if mcContextName == "" {
		mcContextName = config.CurrentContext
	}
	contextName := kubeconfig.GenerateWCClientCertKubeContextName(mcContextName, c.clusterID)

	kubeconfig := clientcmdapi.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: map[string]*clientcmdapi.Cluster{
			contextName: {
				Server:                   c.clusterServer,
				CertificateAuthorityData: c.certCA,
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			contextName: {
				Cluster:  contextName,
				AuthInfo: fmt.Sprintf("%s-user", contextName),
			},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			fmt.Sprintf("%s-user", contextName): {
				ClientCertificateData: c.certCRT,
				ClientKeyData:         c.certKey,
			},
		},
		CurrentContext: contextName,
	}
	// If the destination file exists, we merge the contexts contained in it with the newly created one
	exists, err := afero.Exists(fs, c.filePath)
	if err != nil {
		return "", false, microerror.Mask(err)
	}
	if exists {
		existingKubeConfig, err := clientcmd.LoadFromFile(c.filePath)
		if err != nil {
			return "", false, microerror.Mask(err)
		}

		// First remove entries included in the new config from the existing one
		for clusterName := range kubeconfig.Clusters {
			delete(existingKubeConfig.Clusters, clusterName)
		}
		for authInfoName := range kubeconfig.AuthInfos {
			delete(existingKubeConfig.AuthInfos, authInfoName)
		}
		for ctxName := range kubeconfig.Contexts {
			delete(existingKubeConfig.Contexts, ctxName)
		}

		// Then merge the 2 configs (entries from the new config will be added to the existing one)
		err = mergo.Merge(&kubeconfig, existingKubeConfig, mergo.WithOverride)
		if err != nil {
			return "", false, microerror.Mask(err)
		}
		kubeconfig.CurrentContext = contextName
	}
	err = clientcmd.WriteToFile(kubeconfig, c.filePath)
	if err != nil {
		return "", false, microerror.Mask(err)
	}
	// Change back to the origin context if needed
	if c.loginOptions.originContext != "" && config.CurrentContext != "" && c.loginOptions.originContext != config.CurrentContext {
		// Because we are still in the MC context we need to switch back to the origin context after creating the WC kubeconfig file
		config.CurrentContext = c.loginOptions.originContext
		err = clientcmd.ModifyConfig(k8sConfigAccess, *config, false)
		if err != nil {
			return "", false, microerror.Mask(err)
		}
	}

	return contextName, false, nil
}

func cleanUpClientCertResources(ctx context.Context, clientCertService clientcert.Interface, clientCertResource *clientcert.ClientCert) error {
	err := clientCertService.Delete(ctx, clientCertResource)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func getOrganizationNamespace(ctx context.Context, organizationService organization.Interface, name string) (string, error) {
	org, err := organizationService.Get(ctx, organization.GetOptions{Name: name})
	if organization.IsNotFound(err) {
		return "", microerror.Maskf(organizationNotFoundError, "The organization %s could not be found.", name)
	} else if err != nil {
		return "", microerror.Mask(err)
	}

	namespace := org.(*organization.Organization).Organization.Status.Namespace
	if len(namespace) < 1 {
		return "", microerror.Maskf(unknownOrganizationNamespaceError, "Could not find the namespace for organization %s.", name)
	}

	return namespace, nil
}

func getAllOrganizationNamespaces(ctx context.Context, organizationService organization.Interface) ([]string, error) {
	orgList, err := organizationService.Get(ctx, organization.GetOptions{})
	if organization.IsNoResources(err) {
		return nil, microerror.Maskf(noOrganizationsError, "Could not find any organizations.")
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	var namespaces []string
	for _, org := range orgList.(*organization.Collection).Items {
		if len(org.Organization.Status.Namespace) > 0 {
			namespaces = append(namespaces, org.Organization.Status.Namespace)
		}
	}

	return namespaces, nil
}

func fetchCluster(ctx context.Context, clusterService cluster.Interface, provider, namespace, name string) (*cluster.Cluster, error) {
	o := cluster.GetOptions{
		Namespace:      namespace,
		Name:           name,
		Provider:       provider,
		FallbackToCapi: true,
	}

	c, err := clusterService.Get(ctx, o)
	if cluster.IsNotFound(err) {
		return nil, microerror.Maskf(clusterNotFoundError, "The workload cluster %s could not be found.", name)
	} else if cluster.IsInsufficientPermissions(err) {
		return nil, microerror.Maskf(insufficientPermissionsError, "You don't have the required permissions to get clusters in the %s namespace.", namespace)
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	return c.(*cluster.Cluster), nil
}

func findCluster(ctx context.Context, clusterService cluster.Interface, organizationService organization.Interface, provider, name string, namespaces ...string) (*cluster.Cluster, error) {
	if len(namespaces) == 1 {
		c, err := fetchCluster(ctx, clusterService, provider, namespaces[0], name)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		return c, nil
	}

	clustersCh := make(chan *cluster.Cluster, len(namespaces))
	errorsCh := make(chan error, len(namespaces))

	var wg sync.WaitGroup

	for _, namespace := range namespaces {
		wg.Add(1)

		go func(namespace string) {
			defer wg.Done()

			c, err := fetchCluster(ctx, clusterService, provider, namespace, name)
			if IsClusterNotFound(err) || IsInsufficientPermissions(err) {
				// Nothing to see here.
				return
			} else if err != nil {
				errorsCh <- microerror.Mask(err)
				return
			}

			clustersCh <- c
		}(namespace)
	}

	wg.Wait()

	close(clustersCh)
	close(errorsCh)

	if len(errorsCh) > 0 {
		return nil, microerror.Mask(<-errorsCh)
	}

	switch len(clustersCh) {
	case 1:
		return <-clustersCh, nil

	case 0:
		return nil, microerror.Maskf(clusterNotFoundError, "The workload cluster %s could not be found.\nMake sure you have access to the cluster's organization namespace. You can try the --%s flag to search for legacy clusters that may reside outside the organization namespace. This might fail for non-admin users.", name, flagWCInsecureNamespace)

	default:
		{
			var errMsg strings.Builder
			errMsg.WriteString(fmt.Sprintf("There are multiple workload clusters with the name %s:\n", name))

			i := 1
			for c := range clustersCh {
				org := c.Cluster.Labels[label.Organization]
				if len(org) < 1 {
					org = "n/a"
				}

				errMsg.WriteString(fmt.Sprintf("%d. %s in organization %s\n", i, name, org))

				i++
			}

			errMsg.WriteString(fmt.Sprintf("\nUse the --%s flag to select one from a specific organization.", flagWCOrganization))

			return nil, microerror.Maskf(multipleClustersFoundError, errMsg.String())
		}
	}
}

func validateProvider(provider string) error {
	if provider == key.ProviderKVM {
		return microerror.Maskf(unsupportedProviderError, "Creating a client certificate for a workload cluster is not supported on provider %s.", provider)
	}

	return nil
}
