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

		expectClientID string
		expectError    bool
	}{
		{
			name:           "case 0",
			expectClientID: clientID,
			expectError:    false,
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var err error
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// generate a key
			key, err := getKey()
			if err != nil {
				t.Fatal(err)
			}

			// mock the OIDC issuer
			var issuer string
			hf := func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/auth" {
					http.Redirect(w, r, "http://localhost:8080/oauth/callback?"+r.URL.RawQuery+"&code=codename", http.StatusFound)
				} else if r.URL.Path == "/token" {
					token, err := getToken(issuer, key)
					if err != nil {
						t.Fatal(err)
					}
					w.Header().Set("Content-Type", "text/plain")
					_, err = io.WriteString(w, token)
					if err != nil {
						t.Fatal(err)
					}
				} else if r.URL.Path == "/keys" {
					webKey, err := getJSONWebKey(key)
					if err != nil {
						t.Fatal(err)
					}
					w.Header().Set("Content-Type", "application/json")
					_, err = io.WriteString(w, webKey)
					if err != nil {
						t.Fatal(err)
					}
				} else {
					w.Header().Set("Content-Type", "application/json")
					_, err = io.WriteString(w, strings.ReplaceAll(getIssuerData(), "ISSUER", issuer))
					if err != nil {
						t.Fatal(err)
					}
				}
			}
			s := httptest.NewServer(http.HandlerFunc(hf))
			defer s.Close()

			issuer = s.URL

			// running the OIDC process
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

func getToken(issuer string, key *rsa.PrivateKey) (string, error) {
	idToken, err := getRawToken(issuer, key)
	if err != nil {
		return "", err
	}
	params := url.Values{}
	params.Add("id_token", idToken)
	params.Add("access_token", "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9")
	return params.Encode(), nil
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

func getRawToken(issuer string, key *rsa.PrivateKey) (string, error) {
	token := jwt.New(jwt.SigningMethodRS256)
	claims := make(jwt.MapClaims)
	claims["iss"] = issuer
	claims["aud"] = clientID
	claims["exp"] = time.Now().Add(time.Hour).Unix()
	token.Claims = claims
	tokenString, err := token.SignedString(key)
	if err != nil {
		return "", err
	}
	return tokenString, nil
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

func getKey() (*rsa.PrivateKey, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func getJSONWebKey(key *rsa.PrivateKey) (string, error) {
	jwk := jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{
			{
				Key:       &key.PublicKey,
				Algorithm: "RS256",
			},
		}}
	json, err := json.Marshal(jwk)
	if err != nil {
		return "", err
	}
	return string(json), nil
}
