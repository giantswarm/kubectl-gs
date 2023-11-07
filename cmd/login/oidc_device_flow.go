package login

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/giantswarm/kubectl-gs/v2/pkg/installation"
	"github.com/giantswarm/microerror"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type deviceCodeResponseData struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationUri         string `json:"verification_uri"`
	VerificationUriComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type deviceTokenResponseData struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	IdToken      string `json:"id_token"`
}

type jwtName struct {
	Name string `json:"name"`
}

func handleDeviceFlowOIDC(ctx context.Context, out io.Writer, errOut io.Writer, in *os.File, i *installation.Installation) (authInfo, error) {
	deviceCodeData, err := loadDeviceCode(i)
	if err != nil {
		return authInfo{}, microerror.Mask(err)
	}

	_, _ = fmt.Fprintf(out, "Open this URL in the browser to log in:\n%s\nPress ENTER when ready to continue\n", deviceCodeData.VerificationUriComplete)

	inputReader := bufio.NewReader(in)
	_, err = inputReader.ReadString('\n')
	if err != nil {
		return authInfo{}, microerror.Mask(err)
	}

	deviceTokenData, err := loadDeviceToken(i, deviceCodeData.DeviceCode)
	if err != nil {
		return authInfo{}, microerror.Mask(err)
	}

	userName, err := nameFromToken(deviceTokenData.IdToken)
	if err != nil {
		return authInfo{}, microerror.Mask(err)
	}

	return authInfo{
		username:     fmt.Sprintf("device-%s", userName),
		token:        deviceTokenData.IdToken,
		refreshToken: deviceTokenData.RefreshToken,
		clientID:     clientID,
	}, nil
}

func loadDeviceCode(i *installation.Installation) (deviceCodeResponseData, error) {
	deviceCodeUrl := fmt.Sprintf("%s/device/code", i.AuthURL)

	formData := url.Values{}
	formData.Add("client_id", clientID)
	formData.Add("scope", "openid profile email groups offline_access audience:server:client_id:dex-k8s-authenticator")

	result := deviceCodeResponseData{}
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

func loadDeviceToken(i *installation.Installation, deviceCode string) (deviceTokenResponseData, error) {
	deviceCodeUrl := fmt.Sprintf("%s/device/token", i.AuthURL)

	formData := url.Values{}
	formData.Add("device_code", deviceCode)
	formData.Add("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

	result := deviceTokenResponseData{}
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

	tokenName := jwtName{}
	err = json.Unmarshal(tokenBody, &tokenName)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return strings.ToLower(strings.ReplaceAll(tokenName.Name, " ", ".")), nil
}
