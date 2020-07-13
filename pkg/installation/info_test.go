package installation

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
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
				Provider:  "aws",
				K8sCaCert: "-----BEGIN CERTIFICATE-----\nsomething\notherthing\nlastthing\n-----END CERTIFICATE-----",
				Name:      "test",
			},
		},
		{
			name:                   "case 1: fetch installation information, invalid certificate",
			httpResponseStatusCode: http.StatusOK,
			responsePath:           "testdata/get_installation_info_invalid_certificate.in",
			errorMatcher:           IsCannotParseCertificate,
		},
		{
			name:                   "case 2: fetch installation information, empty certificate",
			httpResponseStatusCode: http.StatusOK,
			responsePath:           "testdata/get_installation_info_empty_certificate.in",
			errorMatcher:           IsCannotParseCertificate,
		},
		{
			name:                   "case 3: fetch installation information, bad request",
			httpResponseStatusCode: http.StatusBadRequest,
			errorMatcher:           IsCannotGetInstallationInfo,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var (
				res []byte
				err error
			)
			if tc.responsePath != "" {
				res, err = ioutil.ReadFile(tc.responsePath)
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}
			}

			ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.httpResponseStatusCode)
				w.Write(res) // nolint:errcheck
			}))
			defer ts.Close()

			info, err := getInstallationInfo(ts.Client(), ts.URL)
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
