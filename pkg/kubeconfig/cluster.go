package kubeconfig

import (
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func GetClusterServer(config *clientcmdapi.Config, contextName string) (string, bool) {
	if contextName == "" {
		return "", false
	}

	currentContext, exists := config.Contexts[contextName]
	if !exists {
		return "", false
	}

	cluster, exists := config.Clusters[currentContext.Cluster]
	if !exists {
		return "", false
	}

	return cluster.Server, true
}
