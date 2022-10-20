package graphql

import (
	"context"
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"

	"github.com/giantswarm/kubectl-gs/v2/test/goldenfile"
)

var update = flag.Bool("update", false, "update .golden reference test files")

// TestClientImpl_ExecuteQuery uses golden files.
//
// go test ./pkg/graphql -run TestClientImpl_ExecuteQuery -update
func TestClientImpl_ExecuteQuery(t *testing.T) {
	testCases := []struct {
		name                string
		query               string
		variables           map[string]string
		responseFile        string
		responseStatusCode  int
		expectedRequestFile string
		expectedResult      map[string]interface{}
		errorMatcher        func(err error) bool
	}{
		{
			name:               "case 0: try to send an empty query",
			query:              "",
			responseStatusCode: http.StatusOK,
			errorMatcher:       IsQuery,
		},
		{
			name: "case 1: send a simple query",
			query: `
query GetSomething {
	something {
		something1
		something2
	}
}
			`,
			responseFile:        "testdata/simple_query_response.in",
			responseStatusCode:  http.StatusOK,
			expectedRequestFile: "simple_query_request.golden",
			expectedResult: map[string]interface{}{
				"something": map[string]interface{}{
					"something1": true,
					"something2": "test",
				},
			},
		},
		{
			name: "case 2: send a more compelx query, with variables",
			query: `
query GetSomething {
	something(name: $test) {
		something1
		something2
	}
}
			`,
			variables: map[string]string{
				"test": "some-test-value",
			},
			responseFile:        "testdata/complex_query_response.in",
			responseStatusCode:  http.StatusOK,
			expectedRequestFile: "complex_query_request.golden",
			expectedResult: map[string]interface{}{
				"something": map[string]interface{}{
					"something1": true,
					"something2": "test",
				},
			},
		},
		{
			name: "case 3: send a simple query, get an empty response",
			query: `
query GetSomething {
	something {
		something1
		something2
	}
}
			`,
			responseFile:        "testdata/simple_query_empty_response.in",
			responseStatusCode:  http.StatusOK,
			expectedRequestFile: "simple_query_request.golden",
			errorMatcher:        IsQuery,
		},
		{
			name: "case 4: send a simple query, get an error response",
			query: `
query GetSomething {
	something {
		something1
		something2
	}
}
			`,
			responseFile:        "testdata/simple_query_error_response.in",
			responseStatusCode:  http.StatusInternalServerError,
			expectedRequestFile: "simple_query_request.golden",
			errorMatcher:        IsHttp,
		},
		{
			name: "case 5: send a simple query, get a GraphQL-specific error response",
			query: `
query GetSomething {
	something {
		something1
		something2
	}
}
			`,
			responseFile:        "testdata/simple_query_gql_error_response.in",
			responseStatusCode:  http.StatusOK,
			expectedRequestFile: "simple_query_request.golden",
			errorMatcher:        IsResponseErrorCollection,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var err error

			var expectedResponse []byte
			{
				if len(tc.responseFile) > 0 {
					expectedResponse, err = os.ReadFile(tc.responseFile)
					if err != nil {
						t.Fatalf("unexpected error: %s", err.Error())
					}
				}
			}

			ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Fatalf("unexpected request method: %s", r.Method)
				}

				req, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}
				r.Body.Close()

				var expectedRequest []byte
				{
					gf := goldenfile.New("testdata", tc.expectedRequestFile)
					if *update {
						err = gf.Update(req)
						if err != nil {
							t.Fatalf("unexpected error: %s", err.Error())
						}

						expectedRequest = req
					} else {
						expectedRequest, err = gf.Read()
						if err != nil {
							t.Fatalf("unexpected error: %s", err.Error())
						}
					}
				}

				diff := cmp.Diff(req, expectedRequest)
				if diff != "" {
					t.Fatalf("result not expected, got: %s", diff)
				}

				w.WriteHeader(tc.responseStatusCode)
				w.Write(expectedResponse) // nolint:errcheck
			}))
			defer ts.Close()

			var gqlClient Client
			{
				httpClient := ts.Client()

				config := ClientImplConfig{
					HttpClient: httpClient,
					Url:        ts.URL,
				}
				gqlClient, err = NewClient(config)
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}
			}

			var result map[string]interface{}
			err = gqlClient.ExecuteQuery(context.Background(), tc.query, tc.variables, &result)
			if tc.errorMatcher != nil {
				if !tc.errorMatcher(err) {
					t.Fatalf("error not matching expected matcher, got: %s", errors.Cause(err))
				}

				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			diff := cmp.Diff(result, tc.expectedResult)
			if diff != "" {
				t.Fatalf("result not expected, got: %s", diff)
			}
		})
	}
}
