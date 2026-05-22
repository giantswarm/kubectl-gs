package credentialplugin

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// makeIDToken builds a syntactically valid (but cryptographically meaningless)
// JWT whose payload encodes the given exp claim. oidc.IsValidIDToken only
// checks the structural format and the exp claim, so this is sufficient.
func makeIDToken(t *testing.T, exp int64) string {
	t.Helper()
	header := base64.RawStdEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payloadBytes, err := json.Marshal(map[string]int64{"exp": exp})
	if err != nil {
		t.Fatal(err)
	}
	payload := base64.RawStdEncoding.EncodeToString(payloadBytes)
	return fmt.Sprintf("%s.%s.sig", header, payload)
}

// isolateCacheHome points the credential cache at a fresh temp dir so tests
// neither read the developer's real cache nor pollute it.
func isolateCacheHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
}

func TestRunReturnsValidEnvIDTokenWithoutRefreshToken(t *testing.T) {
	// Regression: when the OIDC provider does not issue a refresh token the
	// kubeconfig should still work for the lifetime of the id_token issued at
	// login time. The credential plugin must serve that id_token instead of
	// bailing out on the missing refresh token.
	isolateCacheHome(t)
	idToken := makeIDToken(t, time.Now().Add(30*time.Minute).Unix())
	t.Setenv(EnvIssuerURL, "https://idp.example.com")
	t.Setenv(EnvClientID, "client-abc")
	t.Setenv(EnvRefreshToken, "")
	t.Setenv(EnvIDToken, idToken)

	out := &bytes.Buffer{}
	r := &runner{stdout: out, stderr: &bytes.Buffer{}}

	if err := r.Run(&cobra.Command{}, nil); err != nil {
		t.Fatalf("expected no error, got %v\nstdout: %s", err, out.String())
	}

	var resp execCredentialResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("output is not valid ExecCredential JSON: %v\n%s", err, out.String())
	}
	if resp.Status == nil || resp.Status.Token != idToken {
		t.Fatalf("expected token %q, got %+v", idToken, resp.Status)
	}
}

func TestRunRejectsMissingIssuerURL(t *testing.T) {
	isolateCacheHome(t)
	t.Setenv(EnvIssuerURL, "")
	t.Setenv(EnvClientID, "client-abc")
	t.Setenv(EnvRefreshToken, "rt")
	t.Setenv(EnvIDToken, "")

	r := &runner{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	err := r.Run(&cobra.Command{}, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), EnvIssuerURL) {
		t.Errorf("error %q does not mention %s", err.Error(), EnvIssuerURL)
	}
}

func TestRunRejectsMissingClientID(t *testing.T) {
	isolateCacheHome(t)
	t.Setenv(EnvIssuerURL, "https://idp.example.com")
	t.Setenv(EnvClientID, "")
	t.Setenv(EnvRefreshToken, "rt")
	t.Setenv(EnvIDToken, "")

	r := &runner{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	err := r.Run(&cobra.Command{}, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), EnvClientID) {
		t.Errorf("error %q does not mention %s", err.Error(), EnvClientID)
	}
}

func TestRunMissingRefreshTokenAloneNoLongerErrors(t *testing.T) {
	// Just an extra safety net: with issuer/client set and a still-valid
	// id_token, missing refresh token alone must not surface as a missing-env
	// error. (The previous implementation grouped refresh-token into the
	// initial required-env check.)
	isolateCacheHome(t)
	idToken := makeIDToken(t, time.Now().Add(30*time.Minute).Unix())
	t.Setenv(EnvIssuerURL, "https://idp.example.com")
	t.Setenv(EnvClientID, "client-abc")
	t.Setenv(EnvRefreshToken, "")
	t.Setenv(EnvIDToken, idToken)

	r := &runner{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	err := r.Run(&cobra.Command{}, nil)
	if err != nil && strings.Contains(err.Error(), EnvRefreshToken) {
		t.Errorf("did not expect the initial missing-env error to mention %s; got %v",
			EnvRefreshToken, err)
	}
}
