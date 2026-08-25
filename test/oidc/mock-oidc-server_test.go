package oidc

import (
	"net/http"
	"net/url"
	"testing"
)

// TestAuthRedirect covers the /auth endpoint of the mock identity provider.
// The full browser-driven login test that normally exercises it is skipped in
// CI, so assert the redirect contract directly: it must point at the local
// callback server and carry the state the callback handler matches against.
func TestAuthRedirect(t *testing.T) {
	s := NewServer(MockOidcServerConfig{ClientID: "client-id"})
	if err := s.Start(t); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	testCases := []struct {
		name          string
		query         string
		expectedState string
	}{
		{
			name:          "state is carried over to the callback",
			query:         "client_id=client-id&response_type=code&state=some-state",
			expectedState: "some-state",
		},
		{
			// The redirect target is built from constants, so an attacker-supplied
			// redirect_uri must not be able to steer the browser elsewhere.
			name:          "attacker controlled parameters are dropped",
			query:         "state=some-state&redirect_uri=http://evil.example.com/",
			expectedState: "some-state",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := client.Get(s.Issuer() + "/auth?" + tc.query)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != http.StatusFound {
				t.Fatalf("expected status %d, got %d", http.StatusFound, resp.StatusCode)
			}

			location, err := url.Parse(resp.Header.Get("Location"))
			if err != nil {
				t.Fatal(err)
			}
			if location.Host != callbackHost {
				t.Errorf("expected redirect to host %q, got %q", callbackHost, location.Host)
			}
			if location.Path != callbackPath {
				t.Errorf("expected redirect to path %q, got %q", callbackPath, location.Path)
			}
			if got := location.Query().Get("state"); got != tc.expectedState {
				t.Errorf("expected state %q, got %q", tc.expectedState, got)
			}
			if got := location.Query().Get("code"); got != "codename" {
				t.Errorf("expected code %q, got %q", "codename", got)
			}
		})
	}
}
