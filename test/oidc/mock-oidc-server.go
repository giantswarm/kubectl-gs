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
	ClientID             string
	InstallationCodename string
	TokenFailures        int
}

type MockOidcServer struct {
	clientID             string
	installationCodename string
	tokenFailures        int
	server               *httptest.Server
	issuerURL            string
}

func NewServer(config MockOidcServerConfig) *MockOidcServer {
	return &MockOidcServer{
		clientID:             config.ClientID,
		installationCodename: config.InstallationCodename,
		tokenFailures:        config.TokenFailures,
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
			if s.tokenFailures > 0 {
				w.WriteHeader(http.StatusBadRequest)
				_, err := w.Write([]byte{})
				if err != nil {
					t.Fatal(err)
				}
				s.tokenFailures--
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
		} else if r.URL.Path == "/device/token" {
			w.Header().Set("Content-Type", "application/json")
			body, err := getDeviceTokenResponseData(s.clientID, s.issuerURL, key)
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
		ExpiresIn:               60,
		Interval:                10,
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
	/*
		{
		  "data": {
		    "identity": {
		      "provider": "aws",
		      "codename": "viking"
		    },
		    "kubernetes": {
		      "apiUrl": "https://api.g8s.eu-central-1.aws.cps.vodafone.com:443",
		      "authUrl": "https://dex.g8s.eu-central-1.aws.cps.vodafone.com",
		      "caCert": "-----BEGIN CERTIFICATE-----\nMIIDYjCCAkqgAwIBAgIUJGdipWo8k8ls2ys4W1rtYp56DKAwDQYJKoZIhvcNAQEL\nBQAwMDEuMCwGA1UEAxMlZzhzLmV1LWNlbnRyYWwtMS5hd3MuY3BzLnZvZGFmb25l\nLmNvbTAeFw0xNzA3MTExMjQ2NTdaFw0yNzA1MjAxMjQ3MjdaMDAxLjAsBgNVBAMT\nJWc4cy5ldS1jZW50cmFsLTEuYXdzLmNwcy52b2RhZm9uZS5jb20wggEiMA0GCSqG\nSIb3DQEBAQUAA4IBDwAwggEKAoIBAQC1Zd5dzkcPPZ0+RJX8Qi5A5jrvcYibttZZ\nnrQy9JdMCBTr2TigtYkLhle1m/yirETe2qSxZi9/H6Tc/qOaUsjonLa92p8Tqe8q\nRfCT9xYHkAq1SxgfolHT1HtKFb7DTTy3NgFoVi+eKLp/KwEWEf/FndVTpy2xmeas\n6jaCDlLQaR1Fy+ufuhSsFCGD5BMMntN4op0rgByvjNxDT7vlLS07/iGKFXEGk1ur\nj1o8w68gQyvUkCh+oLmqtIBEnmUKDvVZQcPjseVDFb8PtYlR2fXxfamPlx77U0Pz\nBqmknhCDlbZh+ok/rtkH0ucLEQKzMnc6Tu6AXntDTTKCtsPiXC7dAgMBAAGjdDBy\nMA4GA1UdDwEB/wQEAwIBBjAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQWBBSbnngl\naO8+suqjXx64xDP6meQ6fTAwBgNVHREEKTAngiVnOHMuZXUtY2VudHJhbC0xLmF3\ncy5jcHMudm9kYWZvbmUuY29tMA0GCSqGSIb3DQEBCwUAA4IBAQCDIw2PaNSbu0ZA\n7CdynZdp6m4niWPMFCTzdgHOq5QlmsHKc+rzihqchJ/wpII6B7kvNrfCOHB75Y6K\n8bwkrXk5wHvzJYRmk4CWfDZR67Tp5gWWbmCcogxfMLruVJCwG5tUCB2xs/wIm6cW\nlQERw6oPMuXb6fOsj+xgq4r0e9Qd/h7oq0zv03vexuv5cP1EKj+s9Pgy53So4gPh\nt1V2x1f9R7jAl5cCs6GJzBUQ1A+SVmd3zMVKhGsuaK0EvWouJ3ITm5iKte7Jog2/\ne6lenoLgH3vfpufN/sZ2OZU1g+dClxz4CmCO+z5wEZbqsNhzBL6EZnpbr1bosDnF\nZFiXYZp6\n-----END CERTIFICATE-----"
		    }
		  }
		}
	*/
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
