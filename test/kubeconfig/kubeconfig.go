package kubeconfig

import (
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func CreateFakeKubeConfig() clientcmd.ClientConfig {
	config := CreateValidTestConfig()
	clientBuilder := clientcmd.NewNonInteractiveClientConfig(*config, "clean", &clientcmd.ConfigOverrides{}, nil)

	return clientBuilder
}

func CreateFakeKubeConfigFromConfig(config *clientcmdapi.Config) clientcmd.ClientConfig {
	clientBuilder := clientcmd.NewNonInteractiveClientConfig(*config, "clean", &clientcmd.ConfigOverrides{}, nil)

	return clientBuilder
}

func CreateValidTestConfig() *clientcmdapi.Config {
	const (
		name   = "clean"
		server = "https://anything.com:8080"
		token  = "the-token"
	)

	return CreateTestConfig(name, name, name, server, token)
}

func CreateTestConfig(cluster, context, authInfo, server, token string) *clientcmdapi.Config {
	config := clientcmdapi.NewConfig()
	config.Clusters[cluster] = &clientcmdapi.Cluster{
		Server: server,
	}
	config.AuthInfos[authInfo] = &clientcmdapi.AuthInfo{
		Token: token,
	}
	config.Contexts[context] = &clientcmdapi.Context{
		Cluster:   cluster,
		AuthInfo:  authInfo,
		Namespace: "default",
	}
	config.CurrentContext = context

	return config
}

func AddExtraContext(config *clientcmdapi.Config) *clientcmdapi.Config {
	const (
		server          = "https://something.com:8080"
		token           = "the-other-token"
		clientid        = "id"
		refreshToken    = "the-fresh-other-token"
		existingcontext = "anothercodename"
	)
	// adding another context to the kubeconfig
	clusterKey := "gs-" + existingcontext
	userKey := "gs-user-" + existingcontext
	config.Clusters[clusterKey] = &clientcmdapi.Cluster{
		Server: server,
	}
	config.Contexts[clusterKey] = &clientcmdapi.Context{
		Cluster:  clusterKey,
		AuthInfo: userKey,
	}
	config.AuthInfos[userKey] = &clientcmdapi.AuthInfo{
		Token: token,
	}
	return config
}

func CreateNonDefaultTestConfig() *clientcmdapi.Config {
	const (
		server      = "https://anything.com:8080"
		clientkey   = "the-key"
		clientcert  = "the-cert"
		clustername = "arbitraryname"
	)

	config := clientcmdapi.NewConfig()
	config.Clusters[clustername] = &clientcmdapi.Cluster{
		Server: server,
	}
	config.Contexts[clustername] = &clientcmdapi.Context{
		Cluster:  clustername,
		AuthInfo: "user-" + clustername,
	}
	config.CurrentContext = clustername
	config.AuthInfos["user-"+clustername] = &clientcmdapi.AuthInfo{
		ClientCertificateData: []byte(clientcert),
		ClientKeyData:         []byte(clientkey)}
	return config
}

func CreateProviderTestConfig(cluster, context, authInfo, server, clientID, idToken, issuerURL, refreshToken string) *clientcmdapi.Config {
	const (
		authProviderName = "oidc"
		keyClientID      = "client-id"
		keyIdToken       = "id-token"
		keyIdpIssuerUrl  = "idp-issuer-url"
		keyRefreshToken  = "refresh-token"
	)

	config := clientcmdapi.NewConfig()
	config.Clusters[cluster] = &clientcmdapi.Cluster{
		Server: issuerURL,
	}
	config.AuthInfos[authInfo] = &clientcmdapi.AuthInfo{
		AuthProvider: &clientcmdapi.AuthProviderConfig{
			Name: authProviderName,
			Config: map[string]string{
				keyClientID:     clientID,
				keyIdToken:      idToken,
				keyIdpIssuerUrl: issuerURL,
				keyRefreshToken: refreshToken,
			},
		},
	}
	config.Contexts[context] = &clientcmdapi.Context{
		Cluster:  cluster,
		AuthInfo: authInfo,
	}

	return config
}
