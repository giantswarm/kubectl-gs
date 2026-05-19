package oidc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/giantswarm/microerror"
	"golang.org/x/oauth2"
)

type Authenticator struct {
	provider     *gooidc.Provider
	clientConfig oauth2.Config
	challenge    string
	pkceVerifier string
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

// IsValidIDToken reports whether the given raw ID token JWT is valid
func IsValidIDToken(idToken string) bool {
	if idToken == "" {
		return false
	}

	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return false
	}

	payload, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}

	var claims map[string]json.RawMessage
	if err := json.Unmarshal(payload, &claims); err != nil {
		return false
	}

	var exp int64
	if err := json.Unmarshal(claims["exp"], &exp); err != nil || exp == 0 {
		return false
	}

	return time.Unix(exp, 0).After(time.Now().Add(5 * time.Minute))
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
		provider:     provider,
		clientConfig: oauthConfig,
		challenge:    challenge,
	}

	return a, nil
}

// GetPlainAuthURL returns the authorization URL without any Dex-specific parameters.
// This is used for direct OIDC flows (e.g., structured authentication) where the
// issuer is not Dex and doesn't understand connector_id or connector_filter.
//
// PKCE (RFC 7636) is enabled on this path because direct OIDC clients are
// typically registered as public clients (no client_secret) — e.g. an Okta
// SPA/Native app — which require PKCE for the token exchange to succeed.
func (a *Authenticator) GetPlainAuthURL() string {
	a.pkceVerifier = oauth2.GenerateVerifier()
	return a.clientConfig.AuthCodeURL(a.challenge, oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(a.pkceVerifier))
}

func (a *Authenticator) GetAuthURL(connectorID string) string {

	authURL := a.clientConfig.AuthCodeURL(a.challenge, oauth2.AccessTypeOffline)

	// connector_id is specific to dex (https://github.com/dexidp/dex) parameter.
	// It allows user directly select connector to use in authentication flow.
	authURLWithConnectorID := fmt.Sprintf("%s&connector_id=%s", authURL, connectorID)

	return authURLWithConnectorID
}

func (a *Authenticator) GetAuthSelectionURL(connectorType string) string {
	authURL := a.clientConfig.AuthCodeURL(a.challenge, oauth2.AccessTypeOffline)

	// connector_type is specific to GS. It is resolved in the custom Dex frontend.
	// It allows user filter the available connectors and only display the relevant ones
	if connectorType != "" {
		return fmt.Sprintf("%s&connector_filter=%s", authURL, connectorType)
	}

	return authURL
}

func (a *Authenticator) RenewToken(ctx context.Context, refreshToken string) (idToken string, rToken string, err error) {
	s := a.clientConfig.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken})
	t, err := s.Token()
	if err != nil {
		return "", "", microerror.Maskf(cannotRenewTokenError, "%s", err.Error())
	}

	idToken, err = ConvertTokenToRawIDToken(t)
	if err != nil {
		return "", "", microerror.Maskf(cannotRenewTokenError, "%s", err.Error())
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
		// Convert the authorization code into a token. Send the PKCE code
		// verifier when one was generated for this authentication (direct
		// OIDC flow); Dex flows leave it empty.
		var exchangeOpts []oauth2.AuthCodeOption
		if a.pkceVerifier != "" {
			exchangeOpts = append(exchangeOpts, oauth2.VerifierOption(a.pkceVerifier))
		}
		token, err = a.clientConfig.Exchange(ctx, code, exchangeOpts...)
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
