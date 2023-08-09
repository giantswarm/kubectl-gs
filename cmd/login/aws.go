package login

import (
	"context"
	"fmt"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	eks "sigs.k8s.io/cluster-api-provider-aws/controlplane/eks/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/v2/pkg/kubeconfig"
)

type eksClusterConfig struct {
	clusterName          string
	certCA               []byte
	controlPlaneEndpoint string
	filePath             string
	loginOptions         LoginOptions
	region               string

	awsProfileName string
}

// storeWCClientCertCredentials saves the created client certificate credentials into the kubectl config.
func storeWCAWSIAMKubeconfig(k8sConfigAccess clientcmd.ConfigAccess, c eksClusterConfig, mcContextName string) (string, bool, error) {
	config, err := k8sConfigAccess.GetStartingConfig()
	if err != nil {
		return "", false, microerror.Mask(err)
	}

	if mcContextName == "" {
		mcContextName = config.CurrentContext
	}
	contextName := kubeconfig.GenerateWCAWSIAMKubeContextName(mcContextName, c.clusterName)
	userName := fmt.Sprintf("%s-user", contextName)
	clusterName := contextName

	contextExists := false

	{
		// Create authenticated user.
		user, exists := config.AuthInfos[userName]
		if !exists {
			user = clientcmdapi.NewAuthInfo()
		}

		user.Exec = &clientcmdapi.ExecConfig{
			APIVersion: "client.authentication.k8s.io/v1beta1",
			Command:    "aws",
			Args:       []string{"--region", c.region, "eks", "get-token", "--cluster-name", c.clusterName, "--output", "json"},
		}

		if c.awsProfileName != "" {
			user.Exec.Env = []clientcmdapi.ExecEnvVar{
				{
					Name:  "AWS_PROFILE",
					Value: c.awsProfileName,
				},
			}
		}
		// Add user information to config.
		config.AuthInfos[userName] = user
	}

	{
		// Create authenticated cluster.
		cluster, exists := config.Clusters[clusterName]
		if !exists {
			cluster = clientcmdapi.NewCluster()
		}

		cluster.Server = c.controlPlaneEndpoint
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

type kubeconfigFile struct {
	Clusters []kubeCluster `json:"clusters"`
}

type kubeCluster struct {
	Cluster kubeClusterSpec `json:"cluster"`
}

type kubeClusterSpec struct {
	CertificateAuthorityData []byte `json:"certificate-authority-data"`
}

func fetchEKSCAData(ctx context.Context, c k8sclient.Interface, clusterName string, clusterNamespace string) ([]byte, error) {
	var secret v1.Secret
	err := c.CtrlClient().Get(ctx, client.ObjectKey{Name: eksKubeconfigSecretName(clusterName), Namespace: clusterNamespace}, &secret)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	secretData := secret.Data["value"]

	kConfig := &kubeconfigFile{}

	err = yaml.Unmarshal(secretData, kConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	return kConfig.Clusters[0].Cluster.CertificateAuthorityData, err
}

func fetchEKSRegion(ctx context.Context, c k8sclient.Interface, clusterName string, clusterNamespace string) (string, error) {
	var eksCluster eks.AWSManagedControlPlane
	err := c.CtrlClient().Get(ctx, client.ObjectKey{Name: clusterName, Namespace: clusterNamespace}, &eksCluster)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return eksCluster.Spec.Region, nil
}

func eksKubeconfigSecretName(clusterName string) string {
	return fmt.Sprintf("%s-user-kubeconfig", clusterName)
}
