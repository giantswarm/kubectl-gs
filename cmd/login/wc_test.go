package login

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/x509"
	"fmt"
	"math/big"
	"os"
	"reflect"
	"strconv"
	"testing"

	application "github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	corev1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/core/v1alpha1"
	infrastructurev1alpha3 "github.com/giantswarm/apiextensions/v3/pkg/apis/infrastructure/v1alpha3"
	releasev1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/release/v1alpha1"
	securityv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/security/v1alpha1"
	"github.com/giantswarm/apiextensions/v3/pkg/clientset/versioned"
	fakeg8s "github.com/giantswarm/apiextensions/v3/pkg/clientset/versioned/fake"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/k8sclient/v5/pkg/k8scrdclient"
	"github.com/giantswarm/microerror"

	"github.com/spf13/afero"
	corev1 "k8s.io/api/core/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	fakek8s "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake" //nolint:staticcheck

	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/internal/label"
)

func TestWCLogin(t *testing.T) {
	testCases := []struct {
		name                 string
		flags                *flag
		provider             string
		clustersInNamespaces map[string]string
		expectError          *microerror.Error
	}{
		// Logging into WC
		{
			name:                 "case 0",
			clustersInNamespaces: map[string]string{"cluster": "org-organization"},
			flags: &flag{
				WCName:    "cluster",
				WCCertTTL: "8h",
			},
			provider: "aws",
		},
		// Logging into WC that does not exist
		{
			name:                 "case 1",
			clustersInNamespaces: map[string]string{"cluster": "org-organization"},
			flags: &flag{
				WCName:    "anothercluster",
				WCCertTTL: "8h",
			},
			provider:    "aws",
			expectError: clusterNotFoundError,
		},
		// self contained file
		{
			name:                 "case 2",
			clustersInNamespaces: map[string]string{"cluster": "org-organization"},
			flags: &flag{
				WCName:        "cluster",
				WCCertTTL:     "8h",
				SelfContained: "/cluster.yaml",
			},
			provider: "aws",
		},
		// keeping MC context
		{
			name:                 "case 3",
			clustersInNamespaces: map[string]string{"cluster": "org-organization"},
			flags: &flag{
				WCName:      "cluster",
				WCCertTTL:   "8h",
				KeepContext: true,
			},
			provider: "aws",
		},
		// Explicit organization
		{
			name:                 "case 4",
			clustersInNamespaces: map[string]string{"cluster": "org-organization"},
			flags: &flag{
				WCName:         "cluster",
				WCCertTTL:      "8h",
				WCOrganization: "organization",
			},
			provider: "aws",
		},
		// Several clusters in several namespaces exist
		{
			name:                 "case 5",
			clustersInNamespaces: map[string]string{"cluster": "org-organization", "anothercluster": "default"},
			flags: &flag{
				WCName:    "cluster",
				WCCertTTL: "8h",
			},
			provider: "aws",
		},
		// Trying to log into a cluster in default namespace without insecure namespace
		{
			name:                 "case 6",
			clustersInNamespaces: map[string]string{"cluster": "default"},
			flags: &flag{
				WCName:    "cluster",
				WCCertTTL: "8h",
			},
			provider:    "aws",
			expectError: clusterNotFoundError,
		},
		// Trying to log into a cluster in default namespace with insecure namespace
		{
			name:                 "case 7",
			clustersInNamespaces: map[string]string{"cluster": "default"},
			flags: &flag{
				WCName:              "cluster",
				WCCertTTL:           "8h",
				WCInsecureNamespace: true,
			},
			provider: "aws",
		},
		// Trying to log into a cluster on kvm
		{
			name:                 "case 8",
			clustersInNamespaces: map[string]string{"cluster": "org-organization"},
			flags: &flag{
				WCName:    "cluster",
				WCCertTTL: "8h",
			},
			provider:    "kvm",
			expectError: unsupportedProviderError,
		},
		// Trying to log into a cluster on azure
		{
			name:                 "case 9",
			clustersInNamespaces: map[string]string{"cluster": "org-organization"},
			flags: &flag{
				WCName:    "cluster",
				WCCertTTL: "8h",
			},
			provider: "azure",
		},
		// Trying to log into a cluster on openstack
		{
			name:                 "case 9",
			clustersInNamespaces: map[string]string{"cluster": "org-organization"},
			flags: &flag{
				WCName:    "cluster",
				WCCertTTL: "8h",
			},
			provider: "openstack",
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			configDir, err := os.MkdirTemp("", "loginTest")
			if err != nil {
				t.Fatal(err)
			}
			fs := afero.NewOsFs()
			if len(tc.flags.SelfContained) > 0 {
				tc.flags.SelfContained = configDir + tc.flags.SelfContained
			}

			r := runner{
				k8sConfigAccess: &clientcmd.ClientConfigLoadingRules{
					ExplicitPath: configDir + "/config.yaml",
				},
				stdout: new(bytes.Buffer),
				flag:   tc.flags,
				fs:     afero.NewBasePathFs(fs, configDir),
			}
			err = clientcmd.ModifyConfig(r.k8sConfigAccess, *createValidTestConfig("", false), false)
			if err != nil {
				t.Fatal(err)
			}
			originConfig, err := r.k8sConfigAccess.GetStartingConfig()
			if err != nil {
				t.Fatal(err)
			}

			client := FakeK8sClient()
			ctx := context.Background()
			{
				// We create some resources
				err = client.CtrlClient().Create(ctx, getOrganization())
				if err != nil {
					t.Fatal(err)
				}
				for wcName, wcNamespace := range tc.clustersInNamespaces {
					err = client.CtrlClient().Create(ctx, getCluster(wcName, wcNamespace))
					if err != nil {
						t.Fatal(err)
					}
					switch tc.provider {
					case "aws":
						err = client.CtrlClient().Create(ctx, getAWSCluster(wcName, wcNamespace))
						if err != nil {
							t.Fatal(err)
						}
					case "azure":
						err = client.CtrlClient().Create(ctx, getAzureCluster(wcName, wcNamespace))
						if err != nil {
							t.Fatal(err)
						}
					case key.ProviderOpenStack:
						err = client.CtrlClient().Create(ctx, getSecret(wcName+"-ca", wcNamespace, getCAdata()))
						if err != nil {
							fmt.Print(err)
						}
					}
				}
				err = client.CtrlClient().Create(ctx, getRelease())
				if err != nil {
					t.Fatal(err)
				}
			}
			r.setLoginOptions(ctx, []string{"codename"})

			// this is running in a go routine to simulate cert-operator creating the secret
			go createSecret(ctx, client)

			err = r.createClusterClientCert(ctx, client, tc.provider)
			if err != nil {
				if microerror.Cause(err) != tc.expectError {
					t.Fatalf("unexpected error: %s", err.Error())
				}
			} else if tc.expectError != nil {
				t.Fatalf("unexpected success")
			}

			targetConfig, err := r.k8sConfigAccess.GetStartingConfig()
			if err != nil {
				t.Fatal(err)
			}
			if tc.flags.KeepContext && targetConfig.CurrentContext != originConfig.CurrentContext {
				t.Fatalf("expected to keep context %s, got context %s", originConfig.CurrentContext, targetConfig.CurrentContext)
			}
			if len(tc.flags.SelfContained) > 0 {
				if _, err := os.Stat(configDir + "/cluster.yaml"); err != nil {
					t.Fatalf("expected self-contained config file: %s", err)
				}
				if !reflect.DeepEqual(targetConfig, originConfig) {
					t.Fatal("expected origin config to not be modified.")
				}
			}
		})
	}
}

func createSecret(ctx context.Context, client k8sclient.Interface) {
	var certConfigs corev1alpha1.CertConfigList
	var err error

	o := func() error {
		err = client.CtrlClient().List(ctx, &certConfigs)
		if err != nil {
			return microerror.Mask(err)
		}
		if len(certConfigs.Items) != 1 {
			return fmt.Errorf("Expected 1 certConfig, got %v", len(certConfigs.Items))
		}
		return nil
	}
	b := backoff.NewConstant(credentialMaxRetryTimeout, credentialRetryTimeout)

	err = backoff.Retry(o, b)
	if err != nil {
		fmt.Print(err)
		return
	}

	if len(certConfigs.Items) != 1 {
		fmt.Printf("Expected 1 certConfig, got %v", len(certConfigs.Items))
		return
	}
	secretName := certConfigs.Items[0].Name
	secretNamespace := certConfigs.Items[0].Namespace
	err = client.CtrlClient().Create(ctx, getSecret(secretName, secretNamespace, nil))
	if err != nil {
		fmt.Print(err)
	}
}

func getOrganization() *securityv1alpha1.Organization {
	organization := &securityv1alpha1.Organization{
		ObjectMeta: metav1.ObjectMeta{
			Name: "organization",
		},
		Spec: securityv1alpha1.OrganizationSpec{},
		Status: securityv1alpha1.OrganizationStatus{
			Namespace: "org-organization",
		},
	}
	return organization
}

func getCluster(name string, namespace string) *capiv1alpha3.Cluster {
	cluster := &capiv1alpha3.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "cluster.x-k8s.io/v1alpha3",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				label.Cluster:                 name,
				capiv1alpha3.ClusterLabelName: name,
				label.Organization:            "organization",
				label.ReleaseVersion:          "20.0.0",
			},
		},
		Spec: capiv1alpha3.ClusterSpec{},
	}

	return cluster
}
func getAzureCluster(name string, namespace string) *capz.AzureCluster {
	cr := &capz.AzureCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AzureCluster",
			APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha3",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				label.Cluster:                 name,
				capiv1alpha3.ClusterLabelName: name,
				label.Organization:            "organization",
				label.ReleaseVersion:          "20.0.0",
			},
		},
		Spec: capz.AzureClusterSpec{},
	}

	return cr
}
func getAWSCluster(name string, namespace string) *infrastructurev1alpha3.AWSCluster {
	cr := &infrastructurev1alpha3.AWSCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AWSCluster",
			APIVersion: "infrastructure.giantswarm.io/v1alpha3",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				label.Cluster:                 name,
				capiv1alpha3.ClusterLabelName: name,
				label.Organization:            "organization",
				label.ReleaseVersion:          "20.0.0",
			},
		},
		Spec: infrastructurev1alpha3.AWSClusterSpec{},
	}

	return cr
}
func getRelease() *releasev1alpha1.Release {
	cr := &releasev1alpha1.Release{
		ObjectMeta: metav1.ObjectMeta{
			Name: "v20.0.0",
		},
		Spec: releasev1alpha1.ReleaseSpec{
			Components: []releasev1alpha1.ReleaseSpecComponent{
				{
					Name:    "cert-operator",
					Version: "1.0.0",
				},
			},
		},
	}

	return cr
}
func getSecret(name string, namespace string, data map[string][]byte) *corev1.Secret {
	cr := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}

	return cr
}
func getCAdata() map[string][]byte {
	key, _ := getKey()
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(5),
		IsCA:         true,
	}
	ca, _ := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
	return map[string][]byte{
		"tls.key": getPrivKeyPEM(key),
		"tls.crt": getCertPEM(ca),
	}
}

type fakeK8sClient struct {
	ctrlClient client.Client
	k8sClient  *fakek8s.Clientset
	g8sclient  *fakeg8s.Clientset
}

func FakeK8sClient() k8sclient.Interface {
	var err error

	var k8sClient k8sclient.Interface
	{
		scheme := runtime.NewScheme()
		err = capiv1alpha3.AddToScheme(scheme)
		if err != nil {
			panic(err)
		}
		err = capz.AddToScheme(scheme)
		if err != nil {
			panic(err)
		}
		err = infrastructurev1alpha3.AddToScheme(scheme)
		if err != nil {
			panic(err)
		}
		err = application.AddToScheme(scheme)
		if err != nil {
			panic(err)
		}
		err = releasev1alpha1.AddToScheme(scheme)
		if err != nil {
			panic(err)
		}
		err = securityv1alpha1.AddToScheme(scheme)
		if err != nil {
			panic(err)
		}
		err = corev1alpha1.AddToScheme(scheme)
		if err != nil {
			panic(err)
		}
		_ = fakek8s.AddToScheme(scheme)
		client := fakek8s.NewSimpleClientset()
		g8sclient := fakeg8s.NewSimpleClientset()

		k8sClient = &fakeK8sClient{
			ctrlClient: fake.NewFakeClientWithScheme(scheme),
			k8sClient:  client,
			g8sclient:  g8sclient,
		}
	}

	return k8sClient
}

func (f *fakeK8sClient) CRDClient() k8scrdclient.Interface {
	return nil
}

func (f *fakeK8sClient) CtrlClient() client.Client {
	return f.ctrlClient
}

func (f *fakeK8sClient) DynClient() dynamic.Interface {
	return nil
}

func (f *fakeK8sClient) ExtClient() apiextensionsclient.Interface {
	return nil
}

func (f *fakeK8sClient) G8sClient() versioned.Interface {
	return f.g8sclient
}

func (f *fakeK8sClient) K8sClient() kubernetes.Interface {
	return f.k8sClient
}

func (f *fakeK8sClient) RESTClient() rest.Interface {
	return nil
}

func (f *fakeK8sClient) RESTConfig() *rest.Config {
	return nil
}

func (f *fakeK8sClient) Scheme() *runtime.Scheme {
	return nil
}
