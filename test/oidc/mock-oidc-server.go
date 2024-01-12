package oidc

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gopkg.in/square/go-jose.v2"

	"github.com/giantswarm/kubectl-gs/v2/pkg/oidc"
)

type MockOidcServerConfig struct {
	ClientID                 string
	InstallationCodename     string
	TokenRecoverableFailures int
	TokenFatalFailures       int
}

type MockOidcServer struct {
	clientID                 string
	installationCodename     string
	tokenRecoverableFailures int
	tokenFatalFailures       int
	server                   *httptest.Server
	issuerURL                string
}

func NewServer(config MockOidcServerConfig) *MockOidcServer {
	return &MockOidcServer{
		clientID:                 config.ClientID,
		installationCodename:     config.InstallationCodename,
		tokenRecoverableFailures: config.TokenRecoverableFailures,
		tokenFatalFailures:       config.TokenFatalFailures,
	}
}

func (s *MockOidcServer) Start(t *testing.T) error {
	var key *rsa.PrivateKey
	{
		var err error
		key, err = GetKey()
		if err != nil {
			return err
		}
	}

	hf := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth" {
			http.Redirect(w, r, "http://localhost:8080/oauth/callback?"+r.URL.RawQuery+"&code=codename", http.StatusFound)
		} else if r.URL.Path == "/token" {
			if s.tokenRecoverableFailures > 0 {
				w.WriteHeader(http.StatusUnauthorized)
				w.Header().Set("Content-Type", "application/json")
				_, err := w.Write([]byte(`{"error":"authorization_pending"}`))
				if err != nil {
					t.Fatal(err)
				}
				s.tokenRecoverableFailures--
				return
			}
			if s.tokenFatalFailures > 0 {
				w.WriteHeader(http.StatusBadRequest)
				w.Header().Set("Content-Type", "application/json")
				_, err := w.Write([]byte(`{"error":"simulated_error"}`))
				if err != nil {
					t.Fatal(err)
				}
				s.tokenFatalFailures--
				return
			}
			_ = r.ParseForm()
			grantType := r.Form.Get(oidc.DeviceAuthKeyGrantType)
			if grantType == oidc.DeviceAuthGrantType {
				w.Header().Set("Content-Type", "application/json")
				body, err := getDeviceTokenResponseData(s.clientID, s.issuerURL, key)
				if err != nil {
					t.Fatal(err)
				}
				_, err = w.Write(body)
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			token, err := generateAndGetToken(s.clientID, s.issuerURL, key)
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
		} else if r.URL.Path == "/device/code" {
			w.Header().Set("Content-Type", "application/json")
			body, err := getDeviceCodeResponseData(s.issuerURL)
			if err != nil {
				t.Fatal(err)
			}
			_, err = w.Write(body)
			if err != nil {
				t.Fatal(err)
			}
		} else if r.URL.Path == "/graphql" {
			w.Header().Set("Content-Type", "application/json")
			_, err := io.WriteString(w, installationInfo(s.installationCodename, s.issuerURL))
			if err != nil {
				t.Fatal(err)
			}
		} else {
			w.Header().Set("Content-Type", "application/json")
			responseStr := strings.ReplaceAll(GetIssuerData(), "ISSUER", s.issuerURL)
			_, err := io.WriteString(w, responseStr)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	s.server = httptest.NewServer(http.HandlerFunc(hf))
	s.issuerURL = s.server.URL

	return nil
}

func (s *MockOidcServer) Stop() {
	if s.server != nil {
		s.server.Close()
	}
}

func (s *MockOidcServer) Issuer() string {
	return s.issuerURL
}

func getDeviceCodeResponseData(issuer string) ([]byte, error) {
	data := oidc.DeviceCodeResponseData{
		DeviceCode:              "test-device-code",
		UserCode:                "USER-CODE",
		VerificationUri:         fmt.Sprintf("%s/device/code", issuer),
		VerificationUriComplete: fmt.Sprintf("%s/device/code?user_code=USER_CODE", issuer),
		ExpiresIn:               3,
		Interval:                1,
	}
	return json.Marshal(data)
}

func getDeviceTokenResponseData(clientID, issuer string, key *rsa.PrivateKey) ([]byte, error) {
	token, err := getRawToken(clientID, issuer, key)
	if err != nil {
		return nil, err
	}
	data := oidc.DeviceTokenResponseData{
		AccessToken:  token,
		TokenType:    "bearer",
		ExpiresIn:    60,
		RefreshToken: "refresh-token",
		IdToken:      token,
	}
	return json.Marshal(data)
}

func getRawToken(clientID, issuer string, key *rsa.PrivateKey) (string, error) {
	token := jwt.New(jwt.SigningMethodRS256)
	claims := make(jwt.MapClaims)
	claims["name"] = "user"
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

func generateAndGetToken(clientID, issuer string, key *rsa.PrivateKey) (string, error) {
	idToken, err := getRawToken(clientID, issuer, key)
	if err != nil {
		return "", err
	}
	return GetToken(idToken, ""), nil
}

func getJSONWebKey(key *rsa.PrivateKey) (string, error) {
	jwk := jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{
			{
				Key:       &key.PublicKey,
				Algorithm: "RS256",
			},
		}}
	keyJson, err := json.Marshal(jwk)
	if err != nil {
		return "", err
	}
	return string(keyJson), nil
}

func installationInfo(codename, issuer string) string {
	return fmt.Sprintf(`{
		"data": {
			"identity": {
				"provider": "capa",
				"codename": "%s"
			},
			"kubernetes": {
				"apiUrl": "https://anything.com:8080",
				"authUrl": "%s",
				"caCert": ""
			}
		}
	}`, codename, issuer)
}
