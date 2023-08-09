package login

import (
	"fmt"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"

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


func fetchEKSCAData(c k8sclient.Interface, clusterName string, clusterNamespace string) ([]byte , error) {
	c.CtrlClient().Get(ctx, client.ObjectKey{Name: })



}

func eksKubeconfigSecretName(clusterName string) string  {
	fmt.Sprintf("%s-")
}