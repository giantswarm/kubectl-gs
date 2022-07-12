package kubeconfig

import (
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type AuthType int

const (
	AuthTypeUnknown AuthType = iota
	AuthTypeServiceAccount
	AuthTypeAuthProvider
	AuthTypeClientCertificate
)

func GetAuthType(config *clientcmdapi.Config, contextName string) AuthType {
	if contextName == "" {
		return AuthTypeUnknown
	}

	currentContext, exists := config.Contexts[contextName]
	if !exists {
		return AuthTypeUnknown
	}

	authInfo, exists := config.AuthInfos[currentContext.AuthInfo]
	if !exists {
		return AuthTypeUnknown
	}

	switch {
	case len(authInfo.Token) > 0:
		return AuthTypeServiceAccount
	case authInfo.AuthProvider != nil:
		return AuthTypeAuthProvider
	case len(authInfo.ClientCertificate) > 0:
		return AuthTypeClientCertificate
	case len(authInfo.ClientCertificateData) > 0:
		return AuthTypeClientCertificate
	case len(authInfo.ClientKey) > 0:
		return AuthTypeClientCertificate
	case len(authInfo.ClientKeyData) > 0:
		return AuthTypeClientCertificate
	}

	return AuthTypeUnknown
}

// GetAuthProvider fetches the authentication provider from kubeconfig,
// for a desired context name.
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
