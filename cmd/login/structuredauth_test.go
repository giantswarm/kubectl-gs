package login

import (
	"bytes"
	"strings"
	"testing"
)

func TestPickIssuerInteractive(t *testing.T) {
	issuers := []structuredAuthIssuer{
		{IssuerURL: "https://trial-1234.okta.com", ClientID: "okta-client"},
		{IssuerURL: "https://login.microsoftonline.com/tid/v2.0", ClientID: "azure-client"},
		{IssuerURL: "https://accounts.google.com", ClientID: "google-client"},
	}

	tests := []struct {
		name        string
		input       string
		issuers     []structuredAuthIssuer
		wantIssuer  string
		wantErr     bool
		wantErrText string
	}{
		{
			name:       "select first",
			input:      "1\n",
			issuers:    issuers,
			wantIssuer: "https://trial-1234.okta.com",
		},
		{
			name:       "select middle",
			input:      "2\n",
			issuers:    issuers,
			wantIssuer: "https://login.microsoftonline.com/tid/v2.0",
		},
		{
			name:       "select last",
			input:      "3\n",
			issuers:    issuers,
			wantIssuer: "https://accounts.google.com",
		},
		{
			name:       "trims whitespace",
			input:      "  2  \n",
			issuers:    issuers,
			wantIssuer: "https://login.microsoftonline.com/tid/v2.0",
		},
		{
			name:       "retries after invalid then accepts",
			input:      "0\nokta\n2\n",
			issuers:    issuers,
			wantIssuer: "https://login.microsoftonline.com/tid/v2.0",
		},
		{
			name:       "accepts on last attempt",
			input:      "0\n9\n1\n",
			issuers:    issuers,
			wantIssuer: "https://trial-1234.okta.com",
		},
		{
			name:        "gives up after max invalid attempts",
			input:       "0\nokta\n9\n1\n",
			issuers:     issuers,
			wantErr:     true,
			wantErrText: "no valid issuer selection after 3 attempts",
		},
		{
			name:        "EOF after one invalid still aborts",
			input:       "0\n",
			issuers:     issuers,
			wantErr:     true,
			wantErrText: "no issuer selected",
		},
		{
			name:        "rejects immediate EOF",
			input:       "",
			issuers:     issuers,
			wantErr:     true,
			wantErrText: "no issuer selected",
		},
		{
			name:       "single issuer returns immediately without reading input",
			input:      "",
			issuers:    issuers[:1],
			wantIssuer: "https://trial-1234.okta.com",
		},
		{
			name:        "empty list returns error",
			input:       "",
			issuers:     nil,
			wantErr:     true,
			wantErrText: "no OIDC issuers to choose from",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			in := strings.NewReader(tt.input)

			got, err := pickIssuerInteractive(tt.issuers, in, out)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil; output: %q", out.String())
				}
				if tt.wantErrText != "" && !strings.Contains(err.Error(), tt.wantErrText) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.wantErrText)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got == nil {
				t.Fatal("expected an issuer, got nil")
			}
			if got.IssuerURL != tt.wantIssuer {
				t.Errorf("issuer = %q, want %q", got.IssuerURL, tt.wantIssuer)
			}
		})
	}
}

func TestPickIssuerInteractiveRetryHint(t *testing.T) {
	issuers := []structuredAuthIssuer{
		{IssuerURL: "https://a.example.com", ClientID: "a"},
		{IssuerURL: "https://b.example.com", ClientID: "b"},
	}
	out := &bytes.Buffer{}
	_, err := pickIssuerInteractive(issuers, strings.NewReader("nope\n1\n"), out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "Invalid selection \"nope\"") {
		t.Errorf("expected retry hint quoting the bad input; got:\n%s", got)
	}
	if !strings.Contains(got, "attempt(s) left") {
		t.Errorf("expected remaining-attempts hint; got:\n%s", got)
	}
}

func TestPickIssuerInteractiveMenuOutput(t *testing.T) {
	issuers := []structuredAuthIssuer{
		{IssuerURL: "https://trial-1234.okta.com", ClientID: "okta-client"},
		{IssuerURL: "https://login.microsoftonline.com/tid/v2.0", ClientID: "azure-client"},
	}

	out := &bytes.Buffer{}
	_, err := pickIssuerInteractive(issuers, strings.NewReader("1\n"), out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.String()
	wantSubstrings := []string{
		"Multiple OIDC issuers are configured",
		"1) issuer: https://trial-1234.okta.com, client-id: okta-client",
		"2) issuer: https://login.microsoftonline.com/tid/v2.0, client-id: azure-client",
		"Select an issuer [1-2]:",
	}
	for _, s := range wantSubstrings {
		if !strings.Contains(got, s) {
			t.Errorf("menu output missing %q; got:\n%s", s, got)
		}
	}
}
