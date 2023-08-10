package login

import (
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/spf13/afero"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/utils/ptr"

	"github.com/giantswarm/kubectl-gs/v2/pkg/commonconfig"
)

func Test_ClientCert_SelfContainedFiles(t *testing.T) {
	testCases := []struct {
		name             string
		fileName         string
		sourceConfig     *clientcmdapi.Config
		credentialConfig clientCertCredentialConfig
		expectedConfig   clientcmdapi.Config
	}{
		{
			name:     "case 0: Create a new self-contained file",
			fileName: "cluster.yaml",
			credentialConfig: clientCertCredentialConfig{
				clusterID:     "cluster",
				certCRT:       []byte("CertCRT"),
				certKey:       []byte("CertKey"),
				certCA:        []byte("CertCA"),
				clusterServer: "https://api.cluster.k8s.anything.com:443",
			},
			expectedConfig: clientcmdapi.Config{
				Clusters: map[string]*clientcmdapi.Cluster{
					"gs-codename-cluster-clientcert": {
						Server:                   "https://api.cluster.k8s.anything.com:443",
						CertificateAuthorityData: []byte("CertCA"),
					},
				},
				AuthInfos: map[string]*clientcmdapi.AuthInfo{
					"gs-codename-cluster-clientcert-user": {
						ClientCertificateData: []byte("CertCRT"),
						ClientKeyData:         []byte("CertKey"),
					},
				},
				Contexts: map[string]*clientcmdapi.Context{
					"gs-codename-cluster-clientcert": {
						Cluster:  "gs-codename-cluster-clientcert",
						AuthInfo: "gs-codename-cluster-clientcert-user",
					},
				},
				CurrentContext: "gs-codename-cluster-clientcert",
			},
		},
		{
			name:     "case 1: Add a new entry to an existing self-contained file",
			fileName: "cluster.yaml",
			sourceConfig: &clientcmdapi.Config{
				Clusters: map[string]*clientcmdapi.Cluster{
					"initial-context": {
						Server:                   "https://initial-context.anything.com:443",
						CertificateAuthorityData: []byte("InitialContextCertCA"),
					},
				},
				AuthInfos: map[string]*clientcmdapi.AuthInfo{
					"initial-context-user": {
						ClientCertificateData: []byte("InitialContextCertCRT"),
						ClientKeyData:         []byte("InitialContextCertKey"),
					},
				},
				Contexts: map[string]*clientcmdapi.Context{
					"initial-context": {
						Cluster:  "initial-context",
						AuthInfo: "initial-context-user",
					},
				},
				CurrentContext: "initial-context",
			},
			credentialConfig: clientCertCredentialConfig{
				clusterID:     "cluster",
				certCRT:       []byte("CertCRT"),
				certKey:       []byte("CertKey"),
				certCA:        []byte("CertCA"),
				clusterServer: "https://api.cluster.k8s.anything.com:443",
			},
			expectedConfig: clientcmdapi.Config{
				Clusters: map[string]*clientcmdapi.Cluster{
					"initial-context": {
						Server:                   "https://initial-context.anything.com:443",
						CertificateAuthorityData: []byte("InitialContextCertCA"),
					},
					"gs-codename-cluster-clientcert": {
						Server:                   "https://api.cluster.k8s.anything.com:443",
						CertificateAuthorityData: []byte("CertCA"),
					},
				},
				AuthInfos: map[string]*clientcmdapi.AuthInfo{
					"initial-context-user": {
						ClientCertificateData: []byte("InitialContextCertCRT"),
						ClientKeyData:         []byte("InitialContextCertKey"),
					},
					"gs-codename-cluster-clientcert-user": {
						ClientCertificateData: []byte("CertCRT"),
						ClientKeyData:         []byte("CertKey"),
					},
				},
				Contexts: map[string]*clientcmdapi.Context{
					"initial-context": {
						Cluster:  "initial-context",
						AuthInfo: "initial-context-user",
					},
					"gs-codename-cluster-clientcert": {
						Cluster:  "gs-codename-cluster-clientcert",
						AuthInfo: "gs-codename-cluster-clientcert-user",
					},
				},
				CurrentContext: "gs-codename-cluster-clientcert",
			},
		},
		{
			name:     "case 2: Modify an entry in an existing self.contained file",
			fileName: "cluster.yaml",
			sourceConfig: &clientcmdapi.Config{
				Clusters: map[string]*clientcmdapi.Cluster{
					"initial-context": {
						Server:                   "https://initial-context.anything.com:443",
						CertificateAuthorityData: []byte("InitialContextCertCA"),
					},
					"gs-codename-cluster-clientcert": {
						Server:                   "https://api.cluster.k8s.anything.com:443",
						CertificateAuthorityData: []byte("CertCA"),
					},
				},
				AuthInfos: map[string]*clientcmdapi.AuthInfo{
					"initial-context-user": {
						ClientCertificateData: []byte("InitialContextCertCRT"),
						ClientKeyData:         []byte("InitialContextCertKey"),
					},
					"gs-codename-cluster-clientcert-user": {
						ClientCertificateData: []byte("CertCRT"),
						ClientKeyData:         []byte("CertKey"),
					},
				},
				Contexts: map[string]*clientcmdapi.Context{
					"initial-context": {
						Cluster:  "initial-context",
						AuthInfo: "initial-context-user",
					},
					"gs-codename-cluster-clientcert": {
						Cluster:  "gs-codename-cluster-clientcert",
						AuthInfo: "gs-codename-cluster-clientcert-user",
					},
				},
				CurrentContext: "gs-codename-cluster-clientcert",
			},
			credentialConfig: clientCertCredentialConfig{
				clusterID:     "cluster",
				certCRT:       []byte("NewCertCRT"),
				certKey:       []byte("NewCertKey"),
				certCA:        []byte("NewCertCA"),
				clusterServer: "https://api.cluster.k8s.new-anything.com:443",
			},
			expectedConfig: clientcmdapi.Config{
				Clusters: map[string]*clientcmdapi.Cluster{
					"initial-context": {
						Server:                   "https://initial-context.anything.com:443",
						CertificateAuthorityData: []byte("InitialContextCertCA"),
					},
					"gs-codename-cluster-clientcert": {
						Server:                   "https://api.cluster.k8s.new-anything.com:443",
						CertificateAuthorityData: []byte("NewCertCA"),
					},
				},
				AuthInfos: map[string]*clientcmdapi.AuthInfo{
					"initial-context-user": {
						ClientCertificateData: []byte("InitialContextCertCRT"),
						ClientKeyData:         []byte("InitialContextCertKey"),
					},
					"gs-codename-cluster-clientcert-user": {
						ClientCertificateData: []byte("NewCertCRT"),
						ClientKeyData:         []byte("NewCertKey"),
					},
				},
				Contexts: map[string]*clientcmdapi.Context{
					"initial-context": {
						Cluster:  "initial-context",
						AuthInfo: "initial-context-user",
					},
					"gs-codename-cluster-clientcert": {
						Cluster:  "gs-codename-cluster-clientcert",
						AuthInfo: "gs-codename-cluster-clientcert-user",
					},
				},
				CurrentContext: "gs-codename-cluster-clientcert",
			},
		},
		{
			name:     "case 3: Replace all entries in an existing self-contained file",
			fileName: "cluster.yaml",
			sourceConfig: &clientcmdapi.Config{
				Clusters: map[string]*clientcmdapi.Cluster{
					"gs-codename-cluster-clientcert": {
						Server:                   "https://api.cluster.k8s.anything.com:443",
						CertificateAuthorityData: []byte("CertCA"),
					},
				},
				AuthInfos: map[string]*clientcmdapi.AuthInfo{
					"gs-codename-cluster-clientcert-user": {
						ClientCertificateData: []byte("CertCRT"),
						ClientKeyData:         []byte("CertKey"),
					},
				},
				Contexts: map[string]*clientcmdapi.Context{
					"gs-codename-cluster-clientcert": {
						Cluster:  "gs-codename-cluster-clientcert",
						AuthInfo: "gs-codename-cluster-clientcert-user",
					},
				},
				CurrentContext: "gs-codename-cluster-clientcert",
			},
			credentialConfig: clientCertCredentialConfig{
				clusterID:     "cluster",
				certCRT:       []byte("NewCertCRT"),
				certKey:       []byte("NewCertKey"),
				certCA:        []byte("NewCertCA"),
				clusterServer: "https://api.cluster.k8s.new-anything.com:443",
			},
			expectedConfig: clientcmdapi.Config{
				Clusters: map[string]*clientcmdapi.Cluster{
					"gs-codename-cluster-clientcert": {
						Server:                   "https://api.cluster.k8s.new-anything.com:443",
						CertificateAuthorityData: []byte("NewCertCA"),
					},
				},
				AuthInfos: map[string]*clientcmdapi.AuthInfo{
					"gs-codename-cluster-clientcert-user": {
						ClientCertificateData: []byte("NewCertCRT"),
						ClientKeyData:         []byte("NewCertKey"),
					},
				},
				Contexts: map[string]*clientcmdapi.Context{
					"gs-codename-cluster-clientcert": {
						Cluster:  "gs-codename-cluster-clientcert",
						AuthInfo: "gs-codename-cluster-clientcert-user",
					},
				},
				CurrentContext: "gs-codename-cluster-clientcert",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			configDir, err := os.MkdirTemp("", "loginTest")
			if err != nil {
				t.Fatal(err)
			}
			k8sConfigAccess := readConfigFile(path.Join(configDir, "config.yaml"))
			err = clientcmd.ModifyConfig(k8sConfigAccess, *createValidTestConfig("", false), false)
			if err != nil {
				t.Fatal(err)
			}

			fs := afero.NewOsFs()

			selfContainedFilePath := path.Join(configDir, tc.fileName)
			if tc.sourceConfig != nil {
				sourceK8sConfigAccess := readConfigFile(selfContainedFilePath)
				err = clientcmd.ModifyConfig(sourceK8sConfigAccess, *tc.sourceConfig, false)
				if err != nil {
					t.Fatal(err)
				}
			}

			tc.credentialConfig.filePath = selfContainedFilePath
			_, _, err = printWCClientCertCredentials(k8sConfigAccess, fs, tc.credentialConfig, "")
			if err != nil {
				t.Fatal(err)
			}

			selfContainedConfig, err := readConfigFile(selfContainedFilePath).GetStartingConfig()
			if err != nil {
				t.Fatal(err)
			}

			requireConfigsEqual(t, tc.expectedConfig, *selfContainedConfig)
		})
	}
}

func readConfigFile(filePath string) clientcmd.ConfigAccess {
	cf := genericclioptions.NewConfigFlags(true)
	cf.KubeConfig = ptr.To[string](filePath)
	commonConfig := commonconfig.New(cf)
	return commonConfig.GetConfigAccess()
}

func requireConfigsEqual(t *testing.T, expected clientcmdapi.Config, actual clientcmdapi.Config) {
	if expected.CurrentContext != actual.CurrentContext {
		t.Fatalf("expected current context to be %s\n", expected.CurrentContext)
	}

	if len(expected.Clusters) != len(actual.Clusters) {
		t.Fatalf("expected %d Clusters in self-contained file\n", len(expected.Clusters))
	}

	if len(expected.AuthInfos) != len(actual.AuthInfos) {
		t.Fatalf("expected %d AuthInfos in self-contained file\n", len(expected.Clusters))
	}

	if len(expected.Contexts) != len(actual.Contexts) {
		t.Fatalf("expected %d Contexts in self-contained file\n", len(expected.Clusters))
	}

	for clusterName, expectedCluster := range expected.Clusters {
		actualCluster, ok := actual.Clusters[clusterName]
		if !ok {
			t.Fatalf("expected %s Cluster in self-contained file\n", clusterName)
		}
		requireClustersEqual(t, expectedCluster, actualCluster, clusterName)
	}

	for authInfoName, expectedAuthInfo := range expected.AuthInfos {
		actualAuthInfo, ok := actual.AuthInfos[authInfoName]
		if !ok {
			t.Fatalf("expected %s AuthInfo in self-contained file\n", authInfoName)
		}
		requireAuthInfosEqual(t, expectedAuthInfo, actualAuthInfo, authInfoName)
	}

	for contextName, expectedContext := range expected.Contexts {
		actualContext, ok := actual.Contexts[contextName]
		if !ok {
			t.Fatalf("expected %s Context in self-contained file\n", contextName)
		}
		requireContextsEqual(t, expectedContext, actualContext, contextName)
	}
}

func requireClustersEqual(t *testing.T, expected *clientcmdapi.Cluster, actual *clientcmdapi.Cluster, name string) {
	if expected.Server != actual.Server {
		t.Fatalf("expected %s Server in %s Cluster in self-contained file\n", expected.Server, name)
	}

	if !reflect.DeepEqual(expected.CertificateAuthorityData, actual.CertificateAuthorityData) {
		t.Fatalf("expected %s certificate authority data in %s Cluster in self-contained file\n", expected.CertificateAuthorityData, name)
	}
}

func requireAuthInfosEqual(t *testing.T, expected *clientcmdapi.AuthInfo, actual *clientcmdapi.AuthInfo, name string) {
	if !reflect.DeepEqual(expected.ClientCertificateData, actual.ClientCertificateData) {
		t.Fatalf("expected %s client certificate data in %s AuthInfo in self-contained file\n", expected.ClientCertificateData, name)
	}

	if !reflect.DeepEqual(expected.ClientKeyData, actual.ClientKeyData) {
		t.Fatalf("expected %s client key data in %s AuthInfo in self-contained file\n", expected.ClientKeyData, name)
	}
}

func requireContextsEqual(t *testing.T, expected *clientcmdapi.Context, actual *clientcmdapi.Context, name string) {
	if expected.Cluster != actual.Cluster {
		t.Fatalf("expected %s Cluster in %s Context in self-contained file\n", expected.Cluster, name)
	}

	if expected.AuthInfo != actual.AuthInfo {
		t.Fatalf("expected %s AuthInfo in %s Context in self-contained file\n", expected.AuthInfo, name)
	}
}
