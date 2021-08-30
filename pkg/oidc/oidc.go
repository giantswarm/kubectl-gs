package oidc

import (
	"context"
	"fmt"
	"strings"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/giantswarm/microerror"
	"golang.org/x/oauth2"
)

type Authenticator struct {
	provider     gooidc.Provider
	clientConfig oauth2.Config
	challenge    string
}

type UserInfo struct {
	Email         string
	EmailVerified bool
	IDToken       string
	RefreshToken  string
	IssuerURL     string
	Username      string
	Groups        []string
	ClientID      string
	ClientSecret  string
}

type Config struct {
	ClientID     string
	ClientSecret string
	Issuer       string
	RedirectURL  string
	AuthScopes   []string
}

type Claims struct {
	Email    string   `json:"email"`
	Verified bool     `json:"email_verified"`
	Groups   []string `json:"groups"`
}

func New(ctx context.Context, c Config) (*Authenticator, error) {
	provider, err := gooidc.NewProvider(ctx, c.Issuer)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	oauthConfig := oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  c.RedirectURL,
		Scopes:       c.AuthScopes,
	}

	challenge, err := GenerateChallenge()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	a := &Authenticator{
		provider:     *provider,
		clientConfig: oauthConfig,
		challenge:    challenge,
	}

	return a, nil
}

func (a *Authenticator) GetAuthURL(connectorID string) string {

	authURL := a.clientConfig.AuthCodeURL(a.challenge, oauth2.AccessTypeOffline)

	// connector_id is specific to dex (https://github.com/dexidp/dex) parameter.
	// It allows user directly select connector to use in authentication flow.
	authURLWithConnectorID := fmt.Sprintf("%s&connector_id=%s", authURL, connectorID)

	return authURLWithConnectorID
}

func (a *Authenticator) RenewToken(ctx context.Context, refreshToken string) (idToken string, rToken string, err error) {
	s := a.clientConfig.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken})
	t, err := s.Token()
	if err != nil {
		return "", "", microerror.Maskf(cannotRenewTokenError, err.Error())
	}

	idToken, err = ConvertTokenToRawIDToken(t)
	if err != nil {
		return "", "", microerror.Maskf(cannotRenewTokenError, err.Error())
	}
	rToken = t.RefreshToken

	return idToken, rToken, nil
}

func (a *Authenticator) HandleIssuerResponse(ctx context.Context, challenge string, code string) (UserInfo, error) {
	var err error

	if challenge != a.challenge {
		return UserInfo{}, microerror.Mask(invalidChallengeError)
	}

	var token *oauth2.Token
	{
		// Convert the authorization code into a token.
		token, err = a.clientConfig.Exchange(ctx, code)
		if err != nil {
			return UserInfo{}, microerror.Mask(err)
		}
	}

	rawIDToken, err := ConvertTokenToRawIDToken(token)
	if err != nil {
		return UserInfo{}, microerror.Mask(err)
	}

	var idToken *gooidc.IDToken
	{
		// Verify if ID Token is valid.
		oidcConfig := &gooidc.Config{
			ClientID: a.clientConfig.ClientID,
		}

		idToken, err = a.provider.Verifier(oidcConfig).Verify(ctx, rawIDToken)
		if err != nil {
			return UserInfo{}, microerror.Mask(err)
		}
	}

	claims := Claims{}
	{
		// Get the user's info.
		err = idToken.Claims(&claims)
		if err != nil {
			return UserInfo{}, microerror.Mask(err)
		}
	}

	var username string
	{
		username = strings.Split(claims.Email, "@")[0]
	}

	info := UserInfo{
		ClientID:      a.clientConfig.ClientID,
		ClientSecret:  a.clientConfig.ClientSecret,
		Email:         claims.Email,
		EmailVerified: claims.Verified,
		IDToken:       rawIDToken,
		RefreshToken:  token.RefreshToken,
		IssuerURL:     idToken.Issuer,
		Username:      username,
		Groups:        claims.Groups,
	}

	return info, nil
}
