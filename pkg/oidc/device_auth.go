package oidc

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/v2/pkg/installation"
)

const (
	deviceCodeUrlTemplate  = "%s/device/code"
	deviceTokenUrlTemplate = "%s/token"

	DeviceAuthKeyClientID   = "client_id"
	DeviceAuthKeyScope      = "scope"
	DeviceAuthKeyDeviceCode = "device_code"
	DeviceAuthKeyGrantType  = "grant_type"

	DeviceAuthScopes    = "openid profile email groups offline_access audience:server:client_id:dex-k8s-authenticator"
	DeviceAuthGrantType = "urn:ietf:params:oauth:grant-type:device_code"
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
	formData.Add(DeviceAuthKeyClientID, a.clientID)
	formData.Add(DeviceAuthKeyScope, DeviceAuthScopes)

	result := DeviceCodeResponseData{}
	response, err := http.PostForm(fmt.Sprintf(deviceCodeUrlTemplate, a.authURL), formData)
	if err != nil {
		return result, microerror.Maskf(cannotGetDeviceCodeError, err.Error())
	}

	responseBytes, err := bytesFromResponse(response)
	if err != nil {
		return result, microerror.Maskf(cannotGetDeviceCodeError, err.Error())
	}

	if err != nil {
		return result, microerror.Maskf(cannotGetDeviceCodeError, err.Error())
	}

	err = json.Unmarshal(responseBytes, &result)
	if err != nil {
		return result, microerror.Maskf(cannotGetDeviceCodeError, err.Error())
	}

	return result, nil
}

func (a *DeviceAuthenticator) LoadDeviceToken(deviceCode string) (DeviceTokenResponseData, string, error) {

	formData := url.Values{}
	formData.Add(DeviceAuthKeyDeviceCode, deviceCode)
	formData.Add(DeviceAuthKeyGrantType, DeviceAuthGrantType)

	response, err := http.PostForm(fmt.Sprintf(deviceTokenUrlTemplate, a.authURL), formData)
	if err != nil {
		return DeviceTokenResponseData{}, "", microerror.Maskf(cannotGetDeviceTokenError, err.Error())
	}

	responseBytes, err := bytesFromResponse(response)
	if err != nil {
		return DeviceTokenResponseData{}, "", microerror.Maskf(cannotGetDeviceTokenError, err.Error())
	}

	result := DeviceTokenResponseData{}
	err = json.Unmarshal(responseBytes, &result)
	if err != nil {
		return DeviceTokenResponseData{}, "", microerror.Maskf(cannotGetDeviceTokenError, err.Error())
	}

	userName, err := nameFromToken(result.IdToken)
	if err != nil {
		return DeviceTokenResponseData{}, "", microerror.Maskf(cannotGetDeviceTokenError, err.Error())
	}

	return result, userName, nil
}

func bytesFromResponse(response *http.Response) ([]byte, error) {
	defer func() {
		_ = response.Body.Close()
	}()
	if response.StatusCode > 200 {
		return nil, microerror.Mask(errors.New(response.Status))
	}

	bytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return bytes, nil
}

func nameFromToken(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", microerror.Mask(cannotParseJwtError)
	}

	tokenBody, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", microerror.Maskf(cannotParseJwtError, err.Error())
	}

	tokenName := JwtName{}
	err = json.Unmarshal(tokenBody, &tokenName)
	if err != nil {
		return "", microerror.Maskf(cannotParseJwtError, err.Error())
	}

	return strings.ToLower(strings.ReplaceAll(tokenName.Name, " ", ".")), nil
}
