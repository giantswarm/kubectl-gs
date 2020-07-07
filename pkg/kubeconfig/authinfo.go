package kubeconfig

import (
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func GetAuthProvider(config *clientcmdapi.Config, contextName string) (*clientcmdapi.AuthProviderConfig, bool) {
	if contextName == "" {
		return nil, false
	}

	currentContext, exists := config.Contexts[contextName]
	if !exists {
		return nil, false
	}

	authInfo, exists := config.AuthInfos[currentContext.AuthInfo]
	if !exists {
		return nil, false
	}

	if authInfo.AuthProvider == nil {
		return nil, false
	}

	return authInfo.AuthProvider, true
}
