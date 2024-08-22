package oidc

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/v5/pkg/installation"
)

const (
	deviceCodeUrlTemplate  = "%s/device/code"
	deviceTokenUrlTemplate = "%s/token"

	DeviceAuthKeyClientID   = "client_id"
	DeviceAuthKeyScope      = "scope"
	DeviceAuthKeyDeviceCode = "device_code"
	DeviceAuthKeyGrantType  = "grant_type"

	ErrorTypeAuthPending = "authorization_pending"
	ErrorTypeSlowDown    = "slow_down"

	DeviceAuthScopes    = "openid profile email groups offline_access audience:server:client_id:dex-k8s-authenticator"
	DeviceAuthGrantType = "urn:ietf:params:oauth:grant-type:device_code"

	nameClaimKey = "name"
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

type ErrorResponseData struct {
	Error string `json:"error"`
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
		return result, microerror.Maskf(cannotGetDeviceCodeError, "%s", err.Error())
	}

	responseBytes, err := bytesFromResponse(response)
	if err != nil {
		return result, microerror.Maskf(cannotGetDeviceCodeError, "%s", err.Error())
	}

	err = json.Unmarshal(responseBytes, &result)
	if err != nil {
		return result, microerror.Maskf(cannotGetDeviceCodeError, "%s", err.Error())
	}

	return result, nil
}

func (a *DeviceAuthenticator) LoadDeviceToken(data DeviceCodeResponseData) (DeviceTokenResponseData, string, error) {
	response, err := awaitDeviceToken(a.authURL, data)
	if err != nil {
		return DeviceTokenResponseData{}, "", err
	}

	userName, err := nameFromToken(response.IdToken)
	if err != nil {
		return DeviceTokenResponseData{}, "", microerror.Maskf(cannotGetDeviceTokenError, "%s", err.Error())
	}

	return response, userName, nil
}

func awaitDeviceToken(authURL string, data DeviceCodeResponseData) (DeviceTokenResponseData, error) {
	loadTokenTicker := time.NewTicker(time.Duration(data.Interval*1000+250) * time.Millisecond)
	defer loadTokenTicker.Stop()

	expirationTimer := time.NewTimer(time.Duration(data.ExpiresIn) * time.Second)
	defer expirationTimer.Stop()

	responseCh := make(chan DeviceTokenResponseData)
	errCh := make(chan error)

	go func() {
		for {
			<-loadTokenTicker.C
			loadResponse, loadErr := loadDeviceToken(authURL, data.DeviceCode)
			if loadErr == nil {
				responseCh <- loadResponse
				return
			} else if !IsAuthorizationPendingError(loadErr) && !IsTooManyAuthRequestsError(loadErr) {
				errCh <- loadErr
				return
			}
		}
	}()

	go func() {
		<-expirationTimer.C
		errCh <- cannotGetDeviceTokenError
	}()

	var response DeviceTokenResponseData
	var err error
	select {
	case response = <-responseCh:
		break
	case err = <-errCh:
		break
	}

	return response, err
}

func loadDeviceToken(authURL, deviceCode string) (DeviceTokenResponseData, error) {
	formData := url.Values{}
	formData.Add(DeviceAuthKeyDeviceCode, deviceCode)
	formData.Add(DeviceAuthKeyGrantType, DeviceAuthGrantType)

	response, err := http.PostForm(fmt.Sprintf(deviceTokenUrlTemplate, authURL), formData)
	if err != nil {
		return DeviceTokenResponseData{}, microerror.Maskf(cannotGetDeviceTokenError, "%s", err.Error())
	}

	responseBytes, err := bytesFromResponse(response)
	if err != nil {
		fmt.Println(err)
		return DeviceTokenResponseData{}, microerror.Maskf(cannotGetDeviceTokenError, "%s", err.Error())
	}

	if response.StatusCode > 200 {
		result := ErrorResponseData{}
		err = json.Unmarshal(responseBytes, &result)
		if err != nil {
			return DeviceTokenResponseData{}, microerror.Maskf(cannotGetDeviceTokenError, "%s", err.Error())
		}

		switch result.Error {
		case ErrorTypeAuthPending:
			return DeviceTokenResponseData{}, microerror.Maskf(authorizationPendingError, "%s", result.Error)
		case ErrorTypeSlowDown:
			return DeviceTokenResponseData{}, microerror.Maskf(tooManyAuthRequestsError, "%s", result.Error)
		default:
			return DeviceTokenResponseData{}, microerror.Maskf(cannotGetDeviceTokenError, "%s", result.Error)
		}
	}

	result := DeviceTokenResponseData{}
	err = json.Unmarshal(responseBytes, &result)
	if err != nil {
		return DeviceTokenResponseData{}, microerror.Maskf(cannotGetDeviceTokenError, "%s", err.Error())
	}

	return result, nil
}

func bytesFromResponse(response *http.Response) ([]byte, error) {
	defer func() {
		_ = response.Body.Close()
	}()

	bytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return bytes, nil
}

func nameFromToken(token string) (string, error) {
	parsedToken, _, err := new(jwt.Parser).ParseUnverified(token, jwt.MapClaims{})
	if err != nil {
		return "", microerror.Maskf(cannotParseJwtError, "%s", err.Error())
	}

	var claims jwt.MapClaims
	var ok bool
	if claims, ok = parsedToken.Claims.(jwt.MapClaims); !ok {
		return "", microerror.Mask(cannotParseJwtError)
	}

	var nameClaim interface{}
	if nameClaim, ok = claims[nameClaimKey]; !ok {
		return "", microerror.Mask(cannotParseJwtError)
	}

	var name string
	if name, ok = nameClaim.(string); !ok {
		return "", microerror.Mask(cannotParseJwtError)
	}

	return strings.ToLower(strings.ReplaceAll(name, " ", ".")), nil
}
