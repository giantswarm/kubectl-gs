package login

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"dario.cat/mergo"
	"github.com/giantswarm/micrologger"
	securityv1alpha "github.com/giantswarm/organization-operator/api/v1alpha1"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/cluster-api/api/v1beta1"

	"github.com/giantswarm/kubectl-gs/v2/pkg/commonconfig"
)

func TestKubeConfigModification(t *testing.T) {
	testCases := []struct {
		name                string
		flags               *flag
		mcArg               []string
		startConfigs        []*clientcmdapi.Config
		contextOverride     string
		expectUpdatedConfig bool
		organization        securityv1alpha.Organization
		idToken             string
		renewToken          string
		workloadCluster     v1beta1.Cluster
	}{
		{
			name:  "case 0: MC login with provider auth type, selected context, new tokens",
			mcArg: []string{"gs-codename"},
			startConfigs: []*clientcmdapi.Config{
				createValidTestConfig("", true),
				createValidTestConfig("two", true),
			},
			idToken:             "new-id-token",
			renewToken:          "new-refresh-token",
			flags:               &flag{},
			expectUpdatedConfig: false,
		},
		{
			name:  "case 1: MC login with provider auth type, new context, new tokens",
			mcArg: []string{"gs-codenametwo"},
			startConfigs: []*clientcmdapi.Config{
				createValidTestConfig("", true),
				createValidTestConfig("two", true),
			},
			idToken:             "new-id-token",
			renewToken:          "new-refresh-token",
			flags:               &flag{},
			expectUpdatedConfig: true,
		},
		{
			name:  "case 2: MC login with provider auth type, context override, new tokens",
			mcArg: []string{"gs-codenametwo"},
			startConfigs: []*clientcmdapi.Config{
				createValidTestConfig("", true),
				createValidTestConfig("two", true),
			},
			idToken:             "new-id-token",
			renewToken:          "new-refresh-token",
			flags:               &flag{},
			expectUpdatedConfig: true,
			contextOverride:     "gs-codenametwo",
		},
		{
			name:  "case 3: MC login with provider auth type, context override, old tokens",
			mcArg: []string{"gs-codenametwo"},
			startConfigs: []*clientcmdapi.Config{
				createValidTestConfig("", true),
				createValidTestConfig("two", true),
			},
			idToken:             "the-token",
			renewToken:          "the-fresh-token",
			flags:               &flag{},
			expectUpdatedConfig: false,
			contextOverride:     "gs-codenametwo",
		},
		{
			name:                "case 4: MC login with ServiceAccount auth type, selected context",
			mcArg:               []string{"gs-codename"},
			startConfigs:        []*clientcmdapi.Config{createValidTestConfig("", false)},
			flags:               &flag{},
			expectUpdatedConfig: false,
		},
		{
			name:  "case 5: MC login with ServiceAccount auth type, new context",
			mcArg: []string{"gs-codenametwo"},
			startConfigs: []*clientcmdapi.Config{
				createValidTestConfig("", false),
				createValidTestConfig("two", false),
			},
			flags:               &flag{},
			expectUpdatedConfig: true,
		},
		{
			name:  "case 6: MC login with ServiceAccount auth type,context override",
			mcArg: []string{"gs-codenametwo"},
			startConfigs: []*clientcmdapi.Config{
				createValidTestConfig("", false),
				createValidTestConfig("two", false),
			},
			flags:               &flag{},
			contextOverride:     "gs-codenametwo",
			expectUpdatedConfig: false,
		},
		{
			name:  "case 7: WC client certificate with provider auth type, selected context, new tokens",
			mcArg: []string{"gs-codename"},
			startConfigs: []*clientcmdapi.Config{
				createValidTestConfig("", true),
				createValidTestConfig("two", true),
			},
			organization:    createOrganization("test-org", "org-test-org"),
			workloadCluster: createCluster("wc", "org-test-org"),
			idToken:         "new-id-token",
			renewToken:      "new-refresh-token",
			flags: &flag{
				WCName:        "wc",
				WCCertTTL:     "8h",
				SelfContained: "true",
			},
			expectUpdatedConfig: false,
		},
		{
			name:  "case 8: WC client certificate with service account auth type, selected context, new tokens",
			mcArg: []string{"gs-codename"},
			startConfigs: []*clientcmdapi.Config{
				createValidTestConfig("", false),
				createValidTestConfig("two", false),
			},
			organization:    createOrganization("test-org", "org-test-org"),
			workloadCluster: createCluster("wc", "org-test-org"),
			flags: &flag{
				WCName:        "wc",
				WCCertTTL:     "8h",
				SelfContained: "true",
			},
			expectUpdatedConfig: false,
		},
		{
			name:  "case 9: WC client certificate with provider auth type, context not selected, new tokens",
			mcArg: []string{"gs-codenametwo"},
			startConfigs: []*clientcmdapi.Config{
				createValidTestConfig("", true),
				createValidTestConfig("two", true),
			},
			organization:    createOrganization("test-org", "org-test-org"),
			workloadCluster: createCluster("wc", "org-test-org"),
			idToken:         "new-id-token",
			renewToken:      "new-refresh-token",
			flags: &flag{
				WCName:        "wc",
				WCCertTTL:     "8h",
				SelfContained: "true",
			},
			expectUpdatedConfig: true,
		},
		{
			name:  "case 10: WC client certificate with service account auth type, context not selected, new tokens",
			mcArg: []string{"gs-codenametwo"},
			startConfigs: []*clientcmdapi.Config{
				createValidTestConfig("", false),
				createValidTestConfig("two", false),
			},
			organization:    createOrganization("test-org", "org-test-org"),
			workloadCluster: createCluster("wc", "org-test-org"),
			flags: &flag{
				WCName:        "wc",
				WCCertTTL:     "8h",
				SelfContained: "true",
			},
			expectUpdatedConfig: true,
		},
		{
			name:  "case 11: WC client certificate with provider auth type, context not selected, old tokens",
			mcArg: []string{"gs-codenametwo"},
			startConfigs: []*clientcmdapi.Config{
				createValidTestConfig("", true),
				createValidTestConfig("two", true),
			},
			organization:    createOrganization("test-org", "org-test-org"),
			workloadCluster: createCluster("wc", "org-test-org"),
			idToken:         "new-id-token",
			renewToken:      "new-refresh-token",
			flags: &flag{
				WCName:        "wc",
				WCCertTTL:     "8h",
				SelfContained: "true",
			},
			expectUpdatedConfig: true,
		},
		{
			name:  "case 12: WC client certificate with service account auth type, context not selected, old tokens",
			mcArg: []string{"gs-codenametwo"},
			startConfigs: []*clientcmdapi.Config{
				createValidTestConfig("", false),
				createValidTestConfig("two", false),
			},
			organization:    createOrganization("test-org", "org-test-org"),
			workloadCluster: createCluster("wc", "org-test-org"),
			flags: &flag{
				WCName:        "wc",
				WCCertTTL:     "8h",
				SelfContained: "true",
			},
			expectUpdatedConfig: true,
		},
		{
			name:            "case 13: WC client certificate with provider auth type, context overridden, new tokens",
			mcArg:           []string{"gs-codename"},
			contextOverride: "gs-codenametwo",
			startConfigs: []*clientcmdapi.Config{
				createValidTestConfig("", true),
				createValidTestConfig("two", true),
			},
			idToken:         "new-id-token",
			renewToken:      "new-refresh-token",
			organization:    createOrganization("test-org", "org-test-org"),
			workloadCluster: createCluster("wc", "org-test-org"),
			flags: &flag{
				WCName:        "wc",
				WCCertTTL:     "8h",
				SelfContained: "true",
			},
			expectUpdatedConfig: false,
		},
		{
			name:            "case 14: WC client certificate with provider auth type, context overridden, new tokens",
			mcArg:           []string{"gs-codename"},
			contextOverride: "gs-codenametwo",
			startConfigs: []*clientcmdapi.Config{
				createValidTestConfig("", false),
				createValidTestConfig("two", false),
			},
			organization:    createOrganization("test-org", "org-test-org"),
			workloadCluster: createCluster("wc", "org-test-org"),
			flags: &flag{
				WCName:        "wc",
				WCCertTTL:     "8h",
				SelfContained: "true",
			},
			expectUpdatedConfig: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			configDir, err := os.MkdirTemp("", "loginTest")
			if err != nil {
				t.Fatal(err)
			}

			configPath := path.Join(configDir, "config.yaml")
			cf := genericclioptions.NewConfigFlags(true)
			cf.KubeConfig = ptr.To[string](configPath)
			if tc.contextOverride != "" {
				cf.Context = &tc.contextOverride
			}
			fs := afero.NewOsFs()
			if len(tc.flags.SelfContained) > 0 {
				tc.flags.SelfContained = configDir + tc.flags.SelfContained
			}

			logger, err := micrologger.New(micrologger.Config{})
			if err != nil {
				t.Fatal(err)
			}

			s, err := mockKubernetesAndAuthServer(tc.organization, tc.workloadCluster, tc.idToken, tc.renewToken)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()

			r := runner{
				commonConfig: commonconfig.New(cf),
				flag:         tc.flags,
				stdout:       new(bytes.Buffer),
				stderr:       new(bytes.Buffer),
				fs:           afero.NewBasePathFs(fs, configDir),
				logger:       logger,
			}
			k8sConfigAccess := r.commonConfig.GetConfigAccess()

			var mergedConfig *clientcmdapi.Config
			for _, config := range tc.startConfigs {
				if mergedConfig == nil {
					mergedConfig = config
				} else if err := mergo.Merge(mergedConfig, config); err != nil {
					t.Fatal(err)
				}
			}

			for _, cluster := range mergedConfig.Clusters {
				if cluster.Server == "https://anything.com:8080" {
					cluster.Server = s.URL
				}
			}
			for _, authInfo := range mergedConfig.AuthInfos {
				if authInfo.AuthProvider != nil && authInfo.AuthProvider.Config["idp-issuer-url"] == "https://anything.com:8080" {
					authInfo.AuthProvider.Config["idp-issuer-url"] = s.URL
				}
			}

			err = clientcmd.ModifyConfig(k8sConfigAccess, *mergedConfig, false)
			if err != nil {
				t.Fatal(err)
			}

			fileInfo, err := os.Stat(configPath)
			if err != nil {
				t.Fatal(err)
			}

			initialLastMod := fileInfo.ModTime()

			// Simulate a delay in request processing
			time.Sleep(50 * time.Millisecond)

			ctx := context.Background()
			r.setLoginOptions(ctx, &tc.mcArg)
			err = r.run(ctx, &cobra.Command{}, tc.mcArg)
			if err != nil {
				t.Fatalf("%s: unexpected error: %s", tc.name, err.Error())
			}

			fileInfo, err = os.Stat(configPath)
			if err != nil {
				t.Fatal(err)
			}

			newLastMod := fileInfo.ModTime()
			hasUpdatedConfig := newLastMod.After(initialLastMod)
			if hasUpdatedConfig == tc.expectUpdatedConfig {
				return
			}
			if hasUpdatedConfig {
				t.Fatalf("%s: Unexpected kubeconfig update", tc.name)
			} else {
				t.Fatalf("%s: Failed to update kubeconfig as expected", tc.name)
			}
		})
	}
}

func mockKubernetesAndAuthServer(org securityv1alpha.Organization, wc v1beta1.Cluster, idToken string, refreshToken string) (*httptest.Server, error) {
	// Mock Kubernetes API and auth issuer

	var issuer string

	selfSubjectAccessReview := createSelfSubjectAccessReview()

	secret, err := createFullSecret()
	if err != nil {
		return nil, err
	}

	apiVersions := createApiVersions()

	v1ResourceList, _ := createApiResourceMetadata(secret.TypeMeta, true)
	appResourceList, appGroup := createApiResourceMetadata(v1.TypeMeta{Kind: "App", APIVersion: "application.giantswarm.io/v1alpha1"}, true)
	orgResourceList, orgGroup := createApiResourceMetadata(org.TypeMeta, false)
	clusterResourceList, clusterGroup := createApiResourceMetadata(wc.TypeMeta, true)

	apiGroupList := v1.APIGroupList{
		Groups: []v1.APIGroup{appGroup, orgGroup, clusterGroup},
	}

	routeKubernetes := func(r *http.Request) (string, string, error) {
		var responseData interface{}

		switch r.URL.Path {
		case "/graphql":
			responseData = createAthenaGraphQL("", issuer)
		case "/api":
			responseData = apiVersions
		case "/apis":
			responseData = apiGroupList
		case "/api/v1":
			responseData = v1ResourceList
		case "/apis/application.giantswarm.io/v1alpha1":
			responseData = appResourceList
		case "/apis/security.giantswarm.io/v1alpha1":
			responseData = orgResourceList
		case "/apis/cluster.x-k8s.io/v1beta1":
			responseData = clusterResourceList
		case "/apis/authorization.k8s.io/v1/selfsubjectaccessreviews":
			responseData = selfSubjectAccessReview
		case "/apis/security.giantswarm.io/v1alpha1/organizations":
			responseData = securityv1alpha.OrganizationList{Items: []securityv1alpha.Organization{org}}
		case fmt.Sprintf("/apis/cluster.x-k8s.io/v1beta1/namespaces/%s/clusters/%s", wc.Namespace, wc.Name):
			responseData = wc
		case fmt.Sprintf("/api/v1/namespaces/%s/secrets/%s-ca", wc.Namespace, wc.Name):
			responseData = secret
		default:
			return "", "", nil
		}

		jsonBytes, err := json.Marshal(responseData)
		if err != nil {
			return "", "", err
		}
		return "application/json", string(jsonBytes), nil
	}

	routeAuth := func(r *http.Request) (string, string) {
		switch r.URL.Path {
		case "/token":
			return "text/plain", getToken(idToken, refreshToken)
		case "/.well-known/openid-configuration":
			return "application/json", strings.ReplaceAll(getIssuerData(), "ISSUER", issuer)
		}
		return "", ""
	}

	hf := func(w http.ResponseWriter, r *http.Request) {
		var err error
		contentType, responseBody := routeAuth(r)
		if responseBody == "" {
			contentType, responseBody, err = routeKubernetes(r)
		}

		if err != nil {
			w.WriteHeader(500)
			_, _ = io.WriteString(w, "")
		}

		if responseBody == "" {
			w.WriteHeader(404)
		} else {
			w.Header().Set("Content-Type", contentType)
		}

		_, _ = io.WriteString(w, responseBody)
	}

	s := httptest.NewServer(http.HandlerFunc(hf))
	issuer = s.URL

	return s, nil
}

func createCertificate() (certPem []byte, keyPem []byte, err error) {
	key, err := getKey()
	if err != nil {
		return
	}

	ca := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization:  []string{"Company, INC."},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"Washington D.C."},
			StreetAddress: []string{"Pennsylvania Ave."},
			PostalCode:    []string{"12345"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &key.PublicKey, key)
	if err != nil {
		return
	}

	keyPem = pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		},
	)

	certPem = pem.EncodeToMemory(
		&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: caBytes,
		},
	)

	return
}

func createApiResourceMetadata(typeMeta v1.TypeMeta, namespaced bool) (v1.APIResourceList, v1.APIGroup) {
	groupVersion := strings.Split(typeMeta.APIVersion, "/")
	group := ""
	version := groupVersion[0]
	if len(groupVersion) > 1 {
		group = groupVersion[0]
		version = groupVersion[1]
	}
	singularName := strings.ToLower(typeMeta.Kind)
	apiResource := v1.APIResource{
		Group:        group,
		Version:      version,
		Name:         fmt.Sprintf("%ss", singularName),
		SingularName: singularName,
		Namespaced:   namespaced,
		Kind:         typeMeta.Kind,
		Verbs: v1.Verbs{
			"delete",
			"deletecollection",
			"get",
			"list",
			"patch",
			"create",
			"update",
			"watch",
		},
	}
	apiResourceList := v1.APIResourceList{
		GroupVersion: typeMeta.APIVersion,
		APIResources: []v1.APIResource{apiResource},
	}
	apiGroup := v1.APIGroup{
		Name: apiResource.Group,
		Versions: []v1.GroupVersionForDiscovery{
			{
				GroupVersion: typeMeta.APIVersion,
				Version:      apiResource.Version,
			},
		},
		PreferredVersion: v1.GroupVersionForDiscovery{
			GroupVersion: typeMeta.APIVersion,
			Version:      apiResource.Version,
		},
	}
	return apiResourceList, apiGroup
}

func createFullSecret() (corev1.Secret, error) {
	crtPEM, keyPEM, err := createCertificate()
	if err != nil {
		return corev1.Secret{}, err
	}

	secret := corev1.Secret{
		TypeMeta: v1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		Data: map[string][]byte{
			"tls.crt": crtPEM,
			"tls.key": keyPEM,
		},
	}

	return secret, nil
}

func createSelfSubjectAccessReview() authorizationv1.SelfSubjectAccessReview {
	return authorizationv1.SelfSubjectAccessReview{
		TypeMeta: v1.TypeMeta{
			Kind:       "SelfSubjectAccessReview",
			APIVersion: "authorization.k8s.io/v1",
		},
		Status: authorizationv1.SubjectAccessReviewStatus{
			Allowed: true,
		},
	}
}

func createCluster(name string, namespace string) v1beta1.Cluster {
	return v1beta1.Cluster{
		TypeMeta:   v1.TypeMeta{Kind: "Cluster", APIVersion: "cluster.x-k8s.io/v1beta1"},
		ObjectMeta: v1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: v1beta1.ClusterSpec{
			ControlPlaneEndpoint: v1beta1.APIEndpoint{
				Host: "localhost",
				Port: 6443,
			},
		},
		Status: v1beta1.ClusterStatus{},
	}
}

func createOrganization(name string, namespace string) securityv1alpha.Organization {
	return securityv1alpha.Organization{
		TypeMeta:   v1.TypeMeta{Kind: "Organization", APIVersion: "security.giantswarm.io/v1alpha1"},
		ObjectMeta: v1.ObjectMeta{Name: name},
		Spec:       securityv1alpha.OrganizationSpec{},
		Status:     securityv1alpha.OrganizationStatus{Namespace: namespace},
	}
}

func createApiVersions() v1.APIVersions {
	return v1.APIVersions{
		Versions: []string{"v1"},
		ServerAddressByClientCIDRs: []v1.ServerAddressByClientCIDR{
			{ServerAddress: "127.0.0.1:8000"},
		},
	}
}

type athenaIdentity struct {
	Provider string `json:"provider"`
	Codename string `json:"codename"`
}

type athenaKubernetes struct {
	ApiUrl  string `json:"apiUrl"`
	AuthUrl string `json:"authUrl"`
	CaCert  string `json:"caCert"`
}

type athenaData struct {
	Identity   athenaIdentity   `json:"identity"`
	Kubernetes athenaKubernetes `json:"kubernetes"`
}

type athenaContent struct {
	Data athenaData `json:"data"`
}

func createAthenaGraphQL(codename string, url string) athenaContent {
	return athenaContent{
		Data: athenaData{
			Identity: athenaIdentity{
				Provider: "capa",
				Codename: codename,
			},
			Kubernetes: athenaKubernetes{
				ApiUrl:  url,
				AuthUrl: url,
				CaCert:  "-----BEGIN CERTIFICATE-----\\n\\n-----END CERTIFICATE-----",
			},
		},
	}
}
