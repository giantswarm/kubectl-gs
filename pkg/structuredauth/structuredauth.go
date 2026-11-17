// Package structuredauth reads the JWT issuers a cluster's API server trusts
// through Kubernetes structured authentication, from the cluster's
// KubeadmControlPlane.
package structuredauth

import (
	"encoding/base64"

	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

// ConfigPath is where the cluster chart writes the API server's
// AuthenticationConfiguration on control plane nodes.
const ConfigPath = "/etc/kubernetes/policies/auth-config.yaml"

// Issuer is one JWT authenticator of an AuthenticationConfiguration.
type Issuer struct {
	URL string
	// ClientID is the issuer's first audience.
	ClientID             string
	Audiences            []string
	CertificateAuthority string
	DiscoveryURL         string
	Username             ClaimMapping
	Groups               ClaimMapping
	// MapsUIDOrExtra tells whether the issuer maps the user's UID or extra
	// attributes.
	MapsUIDOrExtra       bool
	ClaimValidationRules int
	UserValidationRules  int
}

// ClaimMapping maps a token claim to a user attribute: either Claim with
// Prefix, or a CEL Expression.
type ClaimMapping struct {
	Claim      string
	Prefix     string
	Expression string
}

// authenticationConfiguration mirrors the relevant parts of the Kubernetes
// AuthenticationConfiguration API.
type authenticationConfiguration struct {
	JWT []jwtAuthenticator `json:"jwt"`
}

type jwtAuthenticator struct {
	Issuer               jwtIssuer     `json:"issuer"`
	ClaimMappings        claimMappings `json:"claimMappings"`
	ClaimValidationRules []object      `json:"claimValidationRules"`
	UserValidationRules  []object      `json:"userValidationRules"`
}

type jwtIssuer struct {
	URL                  string   `json:"url"`
	DiscoveryURL         string   `json:"discoveryURL"`
	Audiences            []string `json:"audiences"`
	CertificateAuthority string   `json:"certificateAuthority"`
}

type claimMappings struct {
	Username claimMapping `json:"username"`
	Groups   claimMapping `json:"groups"`
	UID      *object      `json:"uid"`
	Extra    []object     `json:"extra"`
}

type claimMapping struct {
	Claim      string `json:"claim"`
	Prefix     string `json:"prefix"`
	Expression string `json:"expression"`
}

// object is a JSON object whose content is not read.
type object = map[string]interface{}

// Issuers returns the JWT issuers of the AuthenticationConfiguration the
// KubeadmControlPlane writes to ConfigPath. See IssuersAt.
func Issuers(kcp *unstructured.Unstructured) ([]Issuer, error) {
	return IssuersAt(kcp, ConfigPath)
}

// IssuersAt returns the JWT issuers of the AuthenticationConfiguration the
// KubeadmControlPlane writes to path, in file order. Issuers without a URL
// are skipped. It returns no issuer and no error when the KubeadmControlPlane
// writes no such file inline.
func IssuersAt(kcp *unstructured.Unstructured, path string) ([]Issuer, error) {
	files, found, err := unstructured.NestedSlice(kcp.Object, "spec", "kubeadmConfigSpec", "files")
	if err != nil || !found {
		return nil, nil
	}

	for _, f := range files {
		fileMap, ok := f.(map[string]interface{})
		if !ok {
			continue
		}
		filePath, ok := fileMap["path"].(string)
		if !ok || filePath != path {
			continue
		}
		content, ok := fileMap["content"].(string)
		if !ok || content == "" {
			continue
		}

		switch encoding, _ := fileMap["encoding"].(string); encoding {
		case "":
		case "base64":
			decoded, err := base64.StdEncoding.DecodeString(content)
			if err != nil {
				return nil, microerror.Mask(err)
			}
			content = string(decoded)
		default:
			return nil, microerror.Maskf(unsupportedEncodingError, "file %s has encoding %q", path, encoding)
		}

		var authConfig authenticationConfiguration
		if err := yaml.Unmarshal([]byte(content), &authConfig); err != nil {
			return nil, microerror.Mask(err)
		}

		var issuers []Issuer
		for _, jwt := range authConfig.JWT {
			if jwt.Issuer.URL == "" {
				continue
			}
			clientID := ""
			if len(jwt.Issuer.Audiences) > 0 {
				clientID = jwt.Issuer.Audiences[0]
			}
			issuers = append(issuers, Issuer{
				URL:                  jwt.Issuer.URL,
				ClientID:             clientID,
				Audiences:            jwt.Issuer.Audiences,
				CertificateAuthority: jwt.Issuer.CertificateAuthority,
				DiscoveryURL:         jwt.Issuer.DiscoveryURL,
				Username:             ClaimMapping(jwt.ClaimMappings.Username),
				Groups:               ClaimMapping(jwt.ClaimMappings.Groups),
				MapsUIDOrExtra:       jwt.ClaimMappings.UID != nil || len(jwt.ClaimMappings.Extra) > 0,
				ClaimValidationRules: len(jwt.ClaimValidationRules),
				UserValidationRules:  len(jwt.UserValidationRules),
			})
		}
		return issuers, nil
	}

	return nil, nil
}
