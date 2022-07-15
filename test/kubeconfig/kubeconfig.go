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

func CreateValidTestConfig() *clientcmdapi.Config {
	const (
		server = "https://anything.com:8080"
		token  = "the-token"
	)

	config := clientcmdapi.NewConfig()
	config.Clusters["clean"] = &clientcmdapi.Cluster{
		Server: server,
	}
	config.AuthInfos["clean"] = &clientcmdapi.AuthInfo{
		Token: token,
	}
	config.Contexts["clean"] = &clientcmdapi.Context{
		Cluster:   "clean",
		AuthInfo:  "clean",
		Namespace: "default",
	}
	config.CurrentContext = "clean"

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
	config.Clusters["gs-"+existingcontext] = &clientcmdapi.Cluster{
		Server: server,
	}
	config.Contexts["gs-"+existingcontext] = &clientcmdapi.Context{
		Cluster:  "gs-" + existingcontext,
		AuthInfo: "gs-user-" + existingcontext,
	}
	config.AuthInfos["gs-user-"+existingcontext] = &clientcmdapi.AuthInfo{
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
