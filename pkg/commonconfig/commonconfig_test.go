package commonconfig

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/giantswarm/k8sclient/v5/pkg/k8sclienttest"
	"k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/pkg/graphql"
	"github.com/giantswarm/kubectl-gs/pkg/scheme"
)

type gqlRoundTripper struct {
	handler http.HandlerFunc
}

func (g gqlRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	recorder := httptest.NewRecorder()
	g.handler(recorder, request)
	return recorder.Result(), nil
}

func TestCommonConfig_GetProvider(t *testing.T) {
	testCases := []struct {
		name            string
		graphQLResponse string
		expectedResult  string
	}{
		{
			name:            "case 0: AWS",
			graphQLResponse: "{\"data\":{\"identity\":{\"provider\":\"aws\"}}}",
			expectedResult:  key.ProviderAWS,
		},
		{
			name:            "case 1: Azure",
			graphQLResponse: "{\"data\":{\"identity\":{\"provider\":\"azure\"}}}",
			expectedResult:  key.ProviderAzure,
		},
		{
			name:            "case 2: OpenStack",
			graphQLResponse: "{\"data\":{\"identity\":{\"provider\":\"openstack\"}}}",
			expectedResult:  key.ProviderOpenStack,
		},
		{
			name:            "case 3: OpenStack",
			graphQLResponse: "{\"data\":{\"identity\":{\"provider\":\"vsphere\"}}}",
			expectedResult:  key.ProviderVSphere,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			httpClient := http.Client{
				Transport: gqlRoundTripper{
					handler: func(writer http.ResponseWriter, request *http.Request) {
						_, _ = writer.Write([]byte(tc.graphQLResponse))
					},
				},
			}

			var err error
			var commonConfig CommonConfig
			commonConfig.athenaClient, err = graphql.NewClient(graphql.ClientImplConfig{
				HttpClient: &httpClient,
				Url:        "https://example.com",
			})
			result, err := commonConfig.GetProvider(context.Background())
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			if result != tc.expectedResult {
				t.Fatalf("value not expected, got: %s", result)
			}
		})
	}
}

func TestCommonConfig_GetAthenaClient(t *testing.T) {
	testCases := []struct {
		name         string
		objects      []runtime.Object
		errorChecker func(t *testing.T, err error)
	}{
		{
			name: "case 0: valid ingress",
			objects: []runtime.Object{
				&v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "athena",
						Namespace: "giantswarm",
					},
					Spec: v1beta1.IngressSpec{
						Rules: []v1beta1.IngressRule{
							{
								Host: "https://athena.test.test.gigantic.io",
								IngressRuleValue: v1beta1.IngressRuleValue{
									HTTP: &v1beta1.HTTPIngressRuleValue{
										Paths: []v1beta1.HTTPIngressPath{
											{
												Backend: v1beta1.IngressBackend{
													ServiceName: "athena",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			errorChecker: func(t *testing.T, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clientScheme, err := scheme.NewScheme()
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			commonConfig := CommonConfig{
				httpClient: http.DefaultClient,
				k8sClient: k8sclienttest.NewClients(k8sclienttest.ClientsConfig{
					CtrlClient: fake.NewFakeClientWithScheme(clientScheme, tc.objects...),
				}),
			}

			_, err = commonConfig.GetAthenaClient(context.Background())
			tc.errorChecker(t, err)
		})
	}
}
