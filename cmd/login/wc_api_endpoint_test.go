package login

import (
	"strings"
	"testing"
	"time"
)

func TestNormalizeAPIEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		want     string
		wantErr  bool
	}{
		{
			name:     "full https url is kept",
			endpoint: "https://api.mycluster.example.com:6443",
			want:     "https://api.mycluster.example.com:6443",
		},
		{
			name:     "host:port gets https scheme",
			endpoint: "api.mycluster.example.com:6443",
			want:     "https://api.mycluster.example.com:6443",
		},
		{
			name:     "host only gets https scheme",
			endpoint: "api.mycluster.example.com",
			want:     "https://api.mycluster.example.com",
		},
		{
			name:     "empty is invalid",
			endpoint: "",
			wantErr:  true,
		},
		{
			name:     "scheme only is invalid",
			endpoint: "https://",
			wantErr:  true,
		},
		{
			name:     "explicit http scheme is rejected",
			endpoint: "http://api.mycluster.example.com:6443",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeAPIEndpoint(tt.endpoint)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("normalizeAPIEndpoint(%q) = %q, want %q", tt.endpoint, got, tt.want)
			}
		})
	}
}

func TestValidateWCAPIEndpoint(t *testing.T) {
	// base returns a flag set with the always-required fields populated with
	// valid defaults so only the WC-API-endpoint dependency is under test.
	base := func() *flag {
		return &flag{
			WCCertTTL:    "1h",
			LoginTimeout: 60 * time.Second,
		}
	}

	tests := []struct {
		name        string
		mutate      func(f *flag)
		wantErr     bool
		wantErrText string
	}{
		{
			name:   "endpoint unset is fine",
			mutate: func(f *flag) {},
		},
		{
			name: "endpoint with all companions is fine",
			mutate: func(f *flag) {
				f.WCAPIEndpoint = "https://api.example.com:6443"
				f.WCName = "mywc"
				f.WCOIDCIssuer = "https://issuer.example.com"
				f.WCOIDCClientID = "client-id"
				f.WCOIDCCAFile = "/tmp/ca.crt"
			},
		},
		{
			name: "endpoint without companions lists all missing flags",
			mutate: func(f *flag) {
				f.WCAPIEndpoint = "https://api.example.com:6443"
			},
			wantErr:     true,
			wantErrText: "requires",
		},
		{
			name: "endpoint missing only CA file",
			mutate: func(f *flag) {
				f.WCAPIEndpoint = "https://api.example.com:6443"
				f.WCName = "mywc"
				f.WCOIDCIssuer = "https://issuer.example.com"
				f.WCOIDCClientID = "client-id"
			},
			wantErr:     true,
			wantErrText: "--" + flagWCOIDCCAFile,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := base()
			tt.mutate(f)
			err := f.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErrText != "" && !strings.Contains(err.Error(), tt.wantErrText) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.wantErrText)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
