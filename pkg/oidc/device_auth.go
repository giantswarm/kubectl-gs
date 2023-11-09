package oidc

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/giantswarm/kubectl-gs/v2/pkg/installation"
	"github.com/giantswarm/microerror"
)

const (
	deviceCodeUrlTemplate  = "%s/device/code"
	deviceTokenUrlTemplate = "%s/device/token"

	payloadKeyClientID   = "client_id"
	payloadKeyScope      = "scope"
	payloadKeyDeviceCode = "device_code"
	payloadKeyGrantType  = "grant_type"

	payloadValueScopes     = "openid profile email groups offline_access audience:server:client_id:dex-k8s-authenticator"
	payloadValueGrantTypes = "urn:ietf:params:oauth:grant-type:device_code"
)

type DeviceAuthenticator struct {
	clientID string
	authURL  string
}

type DeviceCodeResponseData struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationUri         string `json:"verification_uri"`
	VerificationUriComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type DeviceTokenResponseData struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	IdToken      string `json:"id_token"`
}

type JwtName struct {
	Name string `json:"name"`
}

func NewDeviceAuthenticator(clientID string, i *installation.Installation) *DeviceAuthenticator {
	return &DeviceAuthenticator{
		clientID: clientID,
		authURL:  i.AuthURL,
	}
}

func (a *DeviceAuthenticator) LoadDeviceCode() (DeviceCodeResponseData, error) {
	formData := url.Values{}
	formData.Add(payloadKeyClientID, a.clientID)
	formData.Add(payloadKeyScope, payloadValueScopes)

	result := DeviceCodeResponseData{}
	deviceCodeUrl := fmt.Sprintf(deviceCodeUrlTemplate, a.authURL)
	responseBytes, err := submitFormData(deviceCodeUrl, formData)
	if err != nil {
		return result, microerror.Mask(err)
	}

	err = json.Unmarshal(responseBytes, &result)
	if err != nil {
		return result, microerror.Mask(err)
	}

	return result, nil
}

func (a *DeviceAuthenticator) LoadDeviceToken(deviceCode string) (DeviceTokenResponseData, string, error) {

	formData := url.Values{}
	formData.Add(payloadKeyDeviceCode, deviceCode)
	formData.Add(payloadKeyGrantType, payloadValueGrantTypes)

	deviceTokenUrl := fmt.Sprintf(deviceTokenUrlTemplate, a.authURL)
	responseBytes, err := submitFormData(deviceTokenUrl, formData)
	if err != nil {
		return DeviceTokenResponseData{}, "", microerror.Mask(err)
	}

	result := DeviceTokenResponseData{}
	err = json.Unmarshal(responseBytes, &result)
	if err != nil {
		return DeviceTokenResponseData{}, "", microerror.Mask(err)
	}

	userName, err := nameFromToken(result.IdToken)
	if err != nil {
		return DeviceTokenResponseData{}, "", microerror.Mask(err)
	}

	return result, userName, nil
}

func submitFormData(url string, data url.Values) ([]byte, error) {
	response, err := http.PostForm(url, data)

	defer func() {
		_ = response.Body.Close()
	}()

	if err != nil {
		return nil, microerror.Mask(err)
	}

	responseBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return responseBytes, nil
}

func nameFromToken(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", microerror.Mask(fmt.Errorf("invalid jwt token with %d parts", len(parts)))
	}

	tokenBody, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", microerror.Mask(err)
	}

	tokenName := JwtName{}
	err = json.Unmarshal(tokenBody, &tokenName)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return strings.ToLower(strings.ReplaceAll(tokenName.Name, " ", ".")), nil
}
