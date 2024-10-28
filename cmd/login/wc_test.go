package login

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/x509"
	"fmt"
	"math/big"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	v1 "k8s.io/api/authorization/v1"

	corev1alpha1 "github.com/giantswarm/apiextensions/v6/pkg/apis/core/v1alpha1"
	infrastructurev1alpha3 "github.com/giantswarm/apiextensions/v6/pkg/apis/infrastructure/v1alpha3"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/k8sclient/v8/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	securityv1alpha1 "github.com/giantswarm/organization-operator/api/v1alpha1"
	releasev1alpha1 "github.com/giantswarm/release-operator/v4/api/v1alpha1"
	"github.com/spf13/afero"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/ptr"
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"

	//nolint:staticcheck
	"github.com/giantswarm/kubectl-gs/v5/internal/key"
	"github.com/giantswarm/kubectl-gs/v5/internal/label"
	"github.com/giantswarm/kubectl-gs/v5/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/v5/test/kubeclient"
	testoidc "github.com/giantswarm/kubectl-gs/v5/test/oidc"
)

func TestWCClientCert(t *testing.T) {
	testCases := []struct {
		name                 string
		flags                *flag
		provider             string
		controlPlaneEndpoint string
		creationTimestamp    time.Time
		capi                 bool
		clustersInNamespaces map[string]string
		isAdmin              bool
		permittedResources   []v1.ResourceRule
		expectError          *microerror.Error
	}{
		// Logging into WC
		{
			name:                 "case 0",
			clustersInNamespaces: map[string]string{"cluster": "org-organization"},
			controlPlaneEndpoint: "https://localhost:6443",
			flags: &flag{
				WCName:    "cluster",
				WCCertTTL: "8h",
			},
			provider: "aws",
			isAdmin:  true,
		},
		// Logging into WC that does not exist
		{
			name:                 "case 1",
			clustersInNamespaces: map[string]string{"cluster": "org-organization"},
			controlPlaneEndpoint: "https://localhost:6443",
			flags: &flag{
				WCName:    "anothercluster",
				WCCertTTL: "8h",
			},
			provider:    "aws",
			isAdmin:     true,
			expectError: clusterNotFoundError,
		},
		// self contained file
		{
			name:                 "case 2",
			clustersInNamespaces: map[string]string{"cluster": "org-organization"},
			controlPlaneEndpoint: "https://localhost:6443",
			flags: &flag{
				WCName:        "cluster",
				WCCertTTL:     "8h",
				SelfContained: "/cluster.yaml",
			},
			provider: "aws",
			permittedResources: []v1.ResourceRule{
				{
					Verbs:         []string{"get"},
					Resources:     []string{"organizations"},
					APIGroups:     []string{"security.giantswarm.io/v1alpha1"},
					ResourceNames: []string{"organization"},
				},
			},
		},
		// keeping MC context
		{
			name:                 "case 3",
			clustersInNamespaces: map[string]string{"cluster": "org-organization"},
			controlPlaneEndpoint: "https://localhost:6443",
			flags: &flag{
				WCName:      "cluster",
				WCCertTTL:   "8h",
				KeepContext: true,
			},
			provider: "aws",
			permittedResources: []v1.ResourceRule{
				{
					Verbs:         []string{"get"},
					Resources:     []string{"organizations"},
					APIGroups:     []string{"security.giantswarm.io/v1alpha1"},
					ResourceNames: []string{"organization"},
				},
			},
		},
		// Explicit organization
		{
			name:                 "case 4",
			clustersInNamespaces: map[string]string{"cluster": "org-organization"},
			controlPlaneEndpoint: "https://localhost:6443",
			flags: &flag{
				WCName:         "cluster",
				WCCertTTL:      "8h",
				WCOrganization: "organization",
			},
			provider: "aws",
			isAdmin:  true,
		},
		// Several clusters in several namespaces exist
		{
			name:                 "case 5",
			clustersInNamespaces: map[string]string{"cluster": "org-organization", "anothercluster": "default"},
			controlPlaneEndpoint: "https://localhost:6443",
			flags: &flag{
				WCName:    "cluster",
				WCCertTTL: "8h",
			},
			provider: "aws",
			isAdmin:  true,
		},
		// Trying to log into a cluster in default namespace without insecure namespace
		{
			name:                 "case 6",
			clustersInNamespaces: map[string]string{"cluster": "default"},
			controlPlaneEndpoint: "https://localhost:6443",
			flags: &flag{
				WCName:    "cluster",
				WCCertTTL: "8h",
			},
			provider: "aws",
			permittedResources: []v1.ResourceRule{
				{
					Verbs:         []string{"get"},
					Resources:     []string{"organizations"},
					APIGroups:     []string{"security.giantswarm.io/v1alpha1"},
					ResourceNames: []string{"organization"},
				},
			},
			expectError: clusterNotFoundError,
		},
		// Trying to log into a cluster in default namespace with insecure namespace
		{
			name:                 "case 7",
			clustersInNamespaces: map[string]string{"cluster": "default"},
			controlPlaneEndpoint: "https://localhost:6443",
			flags: &flag{
				WCName:              "cluster",
				WCCertTTL:           "8h",
				WCInsecureNamespace: true,
			},
			provider: "aws",
			permittedResources: []v1.ResourceRule{
				{
					Verbs:         []string{"get"},
					Resources:     []string{"organizations"},
					APIGroups:     []string{"security.giantswarm.io/v1alpha1"},
					ResourceNames: []string{"organization"},
				},
			},
		},
		// Trying to log into a cluster on kvm
		{
			name:                 "case 8",
			clustersInNamespaces: map[string]string{"cluster": "org-organization"},
			controlPlaneEndpoint: "https://localhost:6443",
			flags: &flag{
				WCName:    "cluster",
				WCCertTTL: "8h",
			},
			provider:    "kvm",
			isAdmin:     true,
			expectError: unsupportedProviderError,
		},
		// Trying to log into a cluster on azure
		{
			name:                 "case 9",
			clustersInNamespaces: map[string]string{"cluster": "org-organization"},
			controlPlaneEndpoint: "https://localhost:6443",
			flags: &flag{
				WCName:    "cluster",
				WCCertTTL: "8h",
			},
			provider: "azure",
			isAdmin:  true,
		},
		// Trying to log into a cluster on openstack
		{
			name:                 "case 10",
			clustersInNamespaces: map[string]string{"cluster": "org-organization"},
			controlPlaneEndpoint: "https://localhost:6443",
			flags: &flag{
				WCName:    "cluster",
				WCCertTTL: "8h",
			},
			provider: "openstack",
			capi:     true,
			permittedResources: []v1.ResourceRule{
				{
					Verbs:         []string{"get"},
					Resources:     []string{"organizations"},
					APIGroups:     []string{"security.giantswarm.io/v1alpha1"},
					ResourceNames: []string{"organization"},
				},
			},
		},
		// Logging into WC using cn prefix flag
		{
			name:                 "case 11",
			clustersInNamespaces: map[string]string{"cluster": "org-organization"},
			controlPlaneEndpoint: "https://localhost:6443",
			flags: &flag{
				WCName:         "cluster",
				WCCertTTL:      "8h",
				WCCertCNPrefix: "some-prefix",
			},
			provider: "aws",
			isAdmin:  true,
		},
		// Logging into WC using cn prefix flag in capi
		{
			name:                 "case 12",
			clustersInNamespaces: map[string]string{"cluster": "org-organization"},
			controlPlaneEndpoint: "https://localhost:6443",
			flags: &flag{
				WCName:         "cluster",
				WCCertTTL:      "8h",
				WCCertCNPrefix: "some-prefix",
			},
			provider: "openstack",
			capi:     true,
			isAdmin:  true,
		},
		// Logging into WC with empty control plane endpoint
		{
			name:                 "case 13",
			clustersInNamespaces: map[string]string{"cluster": "org-organization"},
			flags: &flag{
				WCName:    "cluster",
				WCCertTTL: "8h",
			},
			provider:    "aws",
			isAdmin:     true,
			expectError: clusterAPINotKnownError,
		},
		// Logging into newly created WC with empty control plane endpoint
		{
			name:                 "case 14",
			clustersInNamespaces: map[string]string{"cluster": "org-organization"},
			creationTimestamp:    time.Now().Add(-15 * time.Minute),
			flags: &flag{
				WCName:    "cluster",
				WCCertTTL: "8h",
			},
			provider:    "capa",
			capi:        true,
			isAdmin:     true,
			expectError: clusterAPINotReadyError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			configDir, err := os.MkdirTemp("", "loginTest")
			if err != nil {
				t.Fatal(err)
			}
			cf := genericclioptions.NewConfigFlags(true)
			cf.KubeConfig = ptr.To[string](fmt.Sprintf("%s/config.yaml", configDir))
			fs := afero.NewOsFs()
			if len(tc.flags.SelfContained) > 0 {
				tc.flags.SelfContained = configDir + tc.flags.SelfContained
			}
			r := runner{
				commonConfig: commonconfig.New(cf),
				stdout:       new(bytes.Buffer),
				flag:         tc.flags,
				fs:           afero.NewBasePathFs(fs, configDir),
			}
			k8sConfigAccess := r.commonConfig.GetConfigAccess()
			err = clientcmd.ModifyConfig(k8sConfigAccess, *createValidTestConfig("", false), false)
			if err != nil {
				t.Fatal(err)
			}
			originConfig, err := k8sConfigAccess.GetStartingConfig()
			if err != nil {
				t.Fatal(err)
			}

			client := kubeclient.FakeK8sClient()
			client.AddSubjectAccess(tc.isAdmin)
			client.AddSubjectResourceRules("default", tc.permittedResources)

			ctx := context.Background()
			{
				err = client.CtrlClient().Create(ctx, getOrganization("org-organization"))
				if err != nil {
					t.Fatal(err)
				}
				if !key.IsPureCAPIProvider(tc.provider) {
					err = client.CtrlClient().Create(ctx, getRelease(tc.capi))
					if err != nil {
						t.Fatal(err)
					}
				}
				for wcName, wcNamespace := range tc.clustersInNamespaces {
					err = client.CtrlClient().Create(ctx, getCluster(wcName, wcNamespace, tc.controlPlaneEndpoint, tc.creationTimestamp))
					if err != nil {
						t.Fatal(err)
					}
					if tc.capi {
						err = client.CtrlClient().Create(ctx, getSecret(wcName+"-ca", wcNamespace, getCAdata()))
						if err != nil {
							fmt.Print(err)
						}
					} else {
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
						}
					}
				}
			}
			r.setLoginOptions(ctx, &[]string{"codename"})

			// this is running in a go routine to simulate cert-operator creating the secret
			if !tc.capi {
				go createSecret(ctx, client, tc.provider)
			}

			_, _, err = r.createClusterKubeconfig(ctx, client, tc.provider)
			if err != nil {
				if microerror.Cause(err) != tc.expectError {
					t.Fatalf("unexpected error: %s", err.Error())
				}
			} else if tc.expectError != nil {
				t.Fatalf("unexpected success")
			}
			targetConfig, err := k8sConfigAccess.GetStartingConfig()
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

func Test_getWCBasePath(t *testing.T) {
	testCases := []struct {
		name             string
		provider         string
		clusterApiUrl    string
		expectedBasePath string
	}{
		{
			name:             "case 0: simple url",
			provider:         "aws",
			clusterApiUrl:    "https://anything.com",
			expectedBasePath: "anything.com",
		},
		{
			name:             "case 1: prefixed g8s url",
			provider:         "aws",
			clusterApiUrl:    "https://g8s.anything.com",
			expectedBasePath: "anything.com",
		},
		{
			name:             "case 2: prefixed api url",
			provider:         "aws",
			clusterApiUrl:    "https://api.anything.com",
			expectedBasePath: "anything.com",
		},
		{
			name:             "case 3: prefixed api g8s url with port",
			provider:         "aws",
			clusterApiUrl:    "https://api.g8s.anything.com:8080",
			expectedBasePath: "anything.com",
		},
		{
			name:             "case 4: prefixed internal url",
			provider:         "aws",
			clusterApiUrl:    "https://internal-anything.com",
			expectedBasePath: "anything.com",
		},
		{
			name:             "case 5: prefixed internal url",
			provider:         "aws",
			clusterApiUrl:    "https://internal-anything.com",
			expectedBasePath: "anything.com",
		},
		{
			name:             "case 6: prefixed internal g8s url",
			provider:         "aws",
			clusterApiUrl:    "https://internal-g8s.anything.com",
			expectedBasePath: "anything.com",
		},
		{
			name:             "case 7: prefixed internal api url",
			provider:         "aws",
			clusterApiUrl:    "https://internal-api.anything.com",
			expectedBasePath: "anything.com",
		},
		{
			name:             "case 8: prefixed internal api g8s url with port",
			provider:         "aws",
			clusterApiUrl:    "https://internal-api.g8s.anything.com:8080",
			expectedBasePath: "anything.com",
		},
		{
			name:             "case 9: prefixed internal api g8s hostname with port",
			provider:         "aws",
			clusterApiUrl:    "internal-api.g8s.anything.com:8080",
			expectedBasePath: "anything.com",
		},
		{
			name:             "case 10: api url with unknown pattern and port",
			provider:         "aws",
			clusterApiUrl:    "https://invalid-internal-api.g8s.anything.com:8080",
			expectedBasePath: "invalid-internal-api.g8s.anything.com",
		},
		{
			name:             "case 11: pure capi provider url",
			provider:         "capa",
			clusterApiUrl:    "https://api.codename.g8s.anything.com:8080",
			expectedBasePath: "anything.com",
		},
	}
	for _, tc := range testCases {
		cf := genericclioptions.NewConfigFlags(true)
		commonConfig := commonconfig.New(cf)
		k8sConfigAccess := commonConfig.GetConfigAccess()
		testConfig := *createValidTestConfig("", false)
		for _, value := range testConfig.Clusters {
			value.Server = tc.clusterApiUrl
		}
		err := clientcmd.ModifyConfig(k8sConfigAccess, testConfig, false)
		if err != nil {
			t.Fatalf("unexpected error %s", err)
		}

		clusterBasePath, err := getWCBasePath(k8sConfigAccess, tc.provider, "")
		if err != nil {
			t.Fatalf("unexpected error %s", err)
		}

		if clusterBasePath != tc.expectedBasePath {
			t.Fatalf("Base path mismatch: expected %s, actual %s", tc.expectedBasePath, clusterBasePath)
		}
	}
}

func createSecret(ctx context.Context, client k8sclient.Interface, provider string) {
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
	if provider == key.ProviderAzure {
		secretNamespace = metav1.NamespaceDefault
	}
	err = client.CtrlClient().Create(ctx, getSecret(secretName, secretNamespace, nil))
	if err != nil {
		fmt.Print(err)
	}
}

func getOrganization(orgnamespace string) *securityv1alpha1.Organization {
	organization := &securityv1alpha1.Organization{
		ObjectMeta: metav1.ObjectMeta{
			Name: strings.TrimPrefix(orgnamespace, "org-"),
		},
		Spec: securityv1alpha1.OrganizationSpec{},
		Status: securityv1alpha1.OrganizationStatus{
			Namespace: orgnamespace,
		},
	}
	return organization
}

func getCluster(name, namespace, controlPlaneEndpoint string, creationTimestamp time.Time) *capi.Cluster {
	controlPlaneURL, _ := url.Parse(controlPlaneEndpoint)
	cluster := &capi.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "cluster.x-k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				label.Cluster:         name,
				capi.ClusterNameLabel: name,
				label.Organization:    "organization",
				label.ReleaseVersion:  "20.0.0",
			},
		},
		Spec: capi.ClusterSpec{},
	}

	if controlPlaneURL != nil {
		port, err := strconv.ParseInt(controlPlaneURL.Port(), 10, 32)
		if err == nil {
			cluster.Spec.ControlPlaneEndpoint = capi.APIEndpoint{
				Host: controlPlaneURL.Host,
				Port: int32(port), //nolint:gosec
			}
		}
	}

	if !creationTimestamp.IsZero() {
		cluster.ObjectMeta.CreationTimestamp = metav1.NewTime(creationTimestamp)
	}

	return cluster
}

func getAzureCluster(name string, namespace string) *capz.AzureCluster {
	cr := &capz.AzureCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AzureCluster",
			APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				label.Cluster:         name,
				capi.ClusterNameLabel: name,
				label.Organization:    "organization",
				label.ReleaseVersion:  "20.0.0",
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
				label.Cluster:         name,
				capi.ClusterNameLabel: name,
				label.Organization:    "organization",
				label.ReleaseVersion:  "20.0.0",
			},
		},
		Spec: infrastructurev1alpha3.AWSClusterSpec{},
	}

	return cr
}

func getRelease(capi bool) *releasev1alpha1.Release {
	cr := &releasev1alpha1.Release{
		ObjectMeta: metav1.ObjectMeta{
			Name: "v20.0.0",
		},
		Spec: releasev1alpha1.ReleaseSpec{},
	}
	if !capi {
		cr.Spec.Components = []releasev1alpha1.ReleaseSpecComponent{
			{
				Name:    "cert-operator",
				Version: "1.0.0",
			},
		}
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
	privateKey, _ := testoidc.GetKey()
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(5),
		IsCA:         true,
	}
	ca, _ := x509.CreateCertificate(rand.Reader, cert, cert, &privateKey.PublicKey, privateKey)
	return map[string][]byte{
		"tls.key": getPrivKeyPEM(privateKey),
		"tls.crt": getCertPEM(ca),
	}
}
