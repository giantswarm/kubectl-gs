package installation

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"

	"github.com/giantswarm/kubectl-gs/v5/pkg/graphql"
)

func Test_getInstallationInfo(t *testing.T) {
	testCases := []struct {
		name                   string
		httpResponseStatusCode int
		responsePath           string
		expectedResult         installationInfo
		errorMatcher           func(error) bool
	}{
		{
			name:                   "case 0: fetch installation information, correct response",
			httpResponseStatusCode: http.StatusOK,
			responsePath:           "testdata/get_installation_info_correct_response.in",
			expectedResult: installationInfo{
				Identity: installationInfoIdentity{
					Provider: "aws",
					Codename: "test",
				},
				Kubernetes: installationInfoKubernetes{
					ApiUrl:  "https://g8s.test.eu-west-1.aws.gigantic.io",
					AuthUrl: "https://dex.g8s.test.eu-west-1.aws.gigantic.io",
					CaCert:  "-----BEGIN CERTIFICATE-----\nsomething\notherthing\nlastthing\n-----END CERTIFICATE-----",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var (
				res []byte
				err error
			)
			if tc.responsePath != "" {
				res, err = os.ReadFile(tc.responsePath)
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}
			}

			ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.httpResponseStatusCode)
				w.Write(res) // nolint:errcheck
			}))
			defer ts.Close()

			var gqlClient graphql.Client
			{
				httpClient := ts.Client()

				config := graphql.ClientImplConfig{
					HttpClient: httpClient,
					Url:        ts.URL,
				}
				gqlClient, err = graphql.NewClient(config)
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}
			}

			info, err := getInstallationInfo(context.Background(), gqlClient)
			if tc.errorMatcher != nil {
				if !tc.errorMatcher(err) {
					t.Fatalf("error not matching expected matcher, got: %s", errors.Cause(err))
				}

				// All good. Fall through.
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			diff := cmp.Diff(info, tc.expectedResult)
			if diff != "" {
				t.Fatalf("installation info not expected, got: %s", diff)
			}
		})
	}
}
