package login

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	corev1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/giantswarm/kubectl-gs/pkg/data/domain/keypair"
)

const (
	credentialRetryTimeout    = 250 * time.Millisecond
	credentialMaxRetryTimeout = 2 * time.Second

	credentialKeyCertCRT = "crt"
	credentialKeyCertKey = "key"
	credentialKeyCertCA  = "ca"
)

func generateKeypairUID() string {
	hash := sha256.New()
	_, _ = hash.Write([]byte(time.Now().String()))

	uid := fmt.Sprintf("%x", hash.Sum(nil))

	return uid[:16]
}

func generateKeypair(cluster, organization, ttl string, groups []string, clusterBasePath string) (*keypair.Keypair, error) {
	keypairUID := generateKeypairUID()
	keypairName := fmt.Sprintf("%s-%s", cluster, keypairUID)
	commonName := fmt.Sprintf("%s.%s.k8s.%s", keypairUID, cluster, clusterBasePath)

	certConfig := &corev1alpha1.CertConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keypairName,
			Namespace: metav1.NamespaceDefault,
			Labels: map[string]string{
				"cert-operator.giantswarm.io/version": "0.1.0",
				"giantswarm.io/certificate":           keypairUID,
				"giantswarm.io/cluster":               cluster,
				"giantswarm.io/organization":          organization,
			},
		},
		Spec: corev1alpha1.CertConfigSpec{
			Cert: corev1alpha1.CertConfigSpecCert{
				AllowBareDomains:    true,
				ClusterComponent:    keypairUID,
				ClusterID:           cluster,
				CommonName:          commonName,
				DisableRegeneration: true,
				Organizations:       groups,
				TTL:                 ttl,
			},
			VersionBundle: corev1alpha1.CertConfigSpecVersionBundle{
				Version: "0.1.0",
			},
		},
	}

	r := &keypair.Keypair{
		CertConfig: certConfig,
	}

	return r, nil
}

// tryToCreateKeypairCredential tries to fetch the keypair credential
// for a couple of times, until the resource is fetched, or until the timeout is reached.
func tryToGetKeypairCredential(ctx context.Context, keypairService keypair.Interface, name string) (*corev1.Secret, error) {
	var secret *corev1.Secret
	var err error

	retryTimer := time.NewTicker(credentialRetryTimeout)
	defer retryTimer.Stop()

	timeout := time.NewTimer(credentialMaxRetryTimeout)
	defer timeout.Stop()

	for {
		select {
		case <-retryTimer.C:
			secret, err = keypairService.GetCredential(ctx, name)
			if keypair.IsNotFound(err) {
				// Keypair credential has not been created yet, try again until it is.
				continue
			} else if err != nil {
				return nil, microerror.Mask(err)
			}

			return secret, nil
		case <-timeout.C:
			return nil, microerror.Maskf(credentialRetrievalTimedOut, "failed to get the keypair credential on time")
		}
	}
}

// storeKeypairCredential saves the created keypair credentials into the kubectl config.
func storeKeypairCredential(k8sConfigAccess clientcmd.ConfigAccess, fs afero.Fs, keypair *keypair.Keypair, credential *corev1.Secret, clusterBasePath string) (string, error) {
	config, err := k8sConfigAccess.GetStartingConfig()
	if err != nil {
		return "", microerror.Mask(err)
	}

	contextName := keypair.CertConfig.GetName()
	userName := keypair.CertConfig.Spec.Cert.ClusterComponent
	clusterName := fmt.Sprintf("giantswarm-%s", keypair.CertConfig.Spec.Cert.ClusterID)
	clusterServer := fmt.Sprintf("https://api.%s.k8s.%s", keypair.CertConfig.Spec.Cert.ClusterID, clusterBasePath)

	certCRT := credential.Data[credentialKeyCertCRT]
	certKey := credential.Data[credentialKeyCertKey]
	certCA := credential.Data[credentialKeyCertCA]

	{
		// Create authenticated user.
		user, exists := config.AuthInfos[userName]
		if !exists {
			user = clientcmdapi.NewAuthInfo()
		}

		user.ClientCertificateData = certCRT
		user.ClientKeyData = certKey

		// Add user information to config.
		config.AuthInfos[userName] = user
	}

	{
		// Create authenticated cluster.
		cluster, exists := config.Clusters[clusterName]
		if !exists {
			cluster = clientcmdapi.NewCluster()
		}

		cluster.Server = clusterServer
		cluster.CertificateAuthorityData = certCA

		// Add cluster configuration to config.
		config.Clusters[clusterName] = cluster
	}

	{
		// Create authenticated context.
		context, exists := config.Contexts[contextName]
		if !exists {
			context = clientcmdapi.NewContext()
		}

		context.Cluster = clusterName
		context.AuthInfo = userName

		// Add context configuration to config.
		config.Contexts[contextName] = context

		// Select newly created context as current.
		config.CurrentContext = contextName
	}

	err = clientcmd.ModifyConfig(k8sConfigAccess, *config, false)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return contextName, nil
}

func cleanUpKeypairResources(ctx context.Context, keypairService keypair.Interface, keypairResource *keypair.Keypair) error {
	err := keypairService.Delete(ctx, keypairResource)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
