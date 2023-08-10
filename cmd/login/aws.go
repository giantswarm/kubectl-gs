package login

import (
	"context"
	"fmt"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
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

		user.Exec = awsIAMExec(c.clusterName, c.region)

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
		if c.loginOptions.switchToWCContext {
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

// printWCAWSIamCredentials saves the created client certificate credentials into a separate kubectl config file.
func printWCAWSIamCredentials(k8sConfigAccess clientcmd.ConfigAccess, fs afero.Fs, c eksClusterConfig, mcContextName string) (string, bool, error) {
	config, err := k8sConfigAccess.GetStartingConfig()
	if err != nil {
		return "", false, microerror.Mask(err)
	}

	if mcContextName == "" {
		mcContextName = config.CurrentContext
	}
	contextName := kubeconfig.GenerateWCAWSIAMKubeContextName(mcContextName, c.clusterName)

	kubeconfig := clientcmdapi.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: map[string]*clientcmdapi.Cluster{
			contextName: {
				Server:                   c.controlPlaneEndpoint,
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
				Exec: awsIAMExec(c.clusterName, c.region),
			},
		},
		CurrentContext: contextName,
	}
	if c.awsProfileName != "" {
		kubeconfig.AuthInfos[fmt.Sprintf("%s-user", contextName)].Exec.Env = []clientcmdapi.ExecEnvVar{
			{
				Name:  "AWS_PROFILE",
				Value: c.awsProfileName,
			},
		}
	}

	err = mergeKubeconfigs(fs, c.filePath, kubeconfig, contextName)
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

func awsIAMExec(clusterName string, region string) *clientcmdapi.ExecConfig {
	return &clientcmdapi.ExecConfig{
		APIVersion: "client.authentication.k8s.io/v1beta1",
		Command:    "aws",
		Args:       awsKubeconfigExecArgs(clusterName, region),
	}
}

func awsKubeconfigExecArgs(clusterName string, region string) []string {
	return []string{"--region", region, "eks", "get-token", "--cluster-name", clusterName, "--output", "json"}
}
