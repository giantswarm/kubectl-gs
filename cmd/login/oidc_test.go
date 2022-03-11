package login

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"gopkg.in/square/go-jose.v2"

	"github.com/giantswarm/kubectl-gs/pkg/installation"
)

func TestOIDC(t *testing.T) {
	testCases := []struct {
		name string
		key  *rsa.PrivateKey

		expectClientID string
		expectError    bool
	}{
		{
			name:           "case 0",
			key:            getKey(),
			expectClientID: clientID,
			expectError:    false,
		},
		{
			name:        "case 0",
			key:         getInvalidKey(),
			expectError: true,
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var issuer string
			hf := func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/auth" {
					http.Redirect(w, r, "http://localhost:8080/oauth/callback?"+r.URL.RawQuery+"&code=codename", http.StatusFound)
				} else if r.URL.Path == "/token" {
					w.Header().Set("Content-Type", "text/plain")
					io.WriteString(w, getToken(issuer, tc.key))
				} else if r.URL.Path == "/keys" {
					w.Header().Set("Content-Type", "application/json")
					io.WriteString(w, getJSONWebKey(tc.key))
				} else {
					w.Header().Set("Content-Type", "application/json")
					io.WriteString(w, strings.ReplaceAll(getIssuerData(), "ISSUER", issuer))
				}
			}
			s := httptest.NewServer(http.HandlerFunc(hf))
			defer s.Close()

			issuer = s.URL

			authInfo, err := handleOIDC(ctx, new(bytes.Buffer), new(bytes.Buffer), CreateTestInstallationWithIssuer(issuer), false, 8080)
			if err != nil {
				if !tc.expectError {
					t.Fatalf("unexpected error: %s", err.Error())
				}
			} else if tc.expectError {
				t.Fatalf("unexpected success")
			}
			if tc.expectClientID != authInfo.clientID {
				t.Fatalf("expected client ID %s, got  %s", tc.expectClientID, authInfo.clientID)
			}
		})
	}
}

func getToken(issuer string, key *rsa.PrivateKey) string {
	params := url.Values{}
	params.Add("id_token", getRawToken(issuer, key))
	params.Add("access_token", "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9")
	return params.Encode()
}

func CreateTestInstallationWithIssuer(issuer string) *installation.Installation {
	return &installation.Installation{
		K8sApiURL:         "https://g8s.codename.eu-west-1.aws.gigantic.io",
		K8sInternalApiURL: "https://g8s.codename.internal.eu-west-1.aws.gigantic.io",
		AuthURL:           issuer,
		Provider:          "aws",
		Codename:          "codename",
		CACert:            "-----BEGIN CERTIFICATE-----\nsomething\notherthing\nlastthing\n-----END CERTIFICATE-----",
	}
}

func getRawToken(issuer string, key *rsa.PrivateKey) string {
	token := jwt.New(jwt.SigningMethodRS256)
	claims := make(jwt.MapClaims)
	claims["iss"] = issuer
	claims["aud"] = clientID
	claims["exp"] = time.Now().Add(time.Hour).Unix()
	token.Claims = claims
	tokenString, _ := token.SignedString(key)
	return tokenString
}

func getIssuerData() string {
	return `{
		"issuer": "ISSUER",
		"authorization_endpoint": "ISSUER/auth",
		"token_endpoint": "ISSUER/token",
		"jwks_uri": "ISSUER/keys",
		"userinfo_endpoint": "ISSUER/userinfo",
		"response_types_supported": [
		  "code"
		],
		"subject_types_supported": [
		  "public"
		],
		"id_token_signing_alg_values_supported": [
		  "RS256"
		],
		"scopes_supported": [
		  "openid",
		  "email",
		  "groups",
		  "profile",
		  "offline_access"
		],
		"token_endpoint_auth_methods_supported": [
		  "client_secret_basic"
		],
		"claims_supported": [
		  "aud",
		  "email",
		  "email_verified",
		  "exp",
		  "iat",
		  "iss",
		  "locale",
		  "name",
		  "sub"
		]
	}`
}

func getKey() *rsa.PrivateKey {
	key, _ := rsa.GenerateKey(rand.Reader, 1024)
	return key
}

func getInvalidKey() *rsa.PrivateKey {
	key, _ := rsa.GenerateKey(rand.Reader, 100)
	return key
}

func getJSONWebKey(key *rsa.PrivateKey) string {
	jwk := jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{
			{
				Key:       &key.PublicKey,
				Algorithm: "RS256",
			},
		}}
	json, _ := json.Marshal(jwk)
	return string(json)
}
