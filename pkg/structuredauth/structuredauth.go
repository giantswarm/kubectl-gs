// Package structuredauth reads the JWT issuers a cluster's API server trusts
// through Kubernetes structured authentication, from the cluster's
// KubeadmControlPlane.
package structuredauth

import (
	"encoding/base64"
	"encoding/json"
	"maps"
	"slices"
	"strings"

	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

// ConfigPath is where the cluster chart writes the API server's
// AuthenticationConfiguration on control plane nodes.
const ConfigPath = "/etc/kubernetes/policies/auth-config.yaml"

// Issuer is one JWT authenticator of an AuthenticationConfiguration.
type Issuer struct {
	URL                  string
	Audiences            []string
	CertificateAuthority string
	DiscoveryURL         string
	EgressSelectorType   string
	Username             ClaimMapping
	Groups               ClaimMapping
	// MapsUIDOrExtra tells whether the issuer maps the user's UID or extra
	// attributes.
	MapsUIDOrExtra       bool
	ClaimValidationRules int
	UserValidationRules  int
	// UnknownFields lists, as dotted paths, the fields of the authenticator
	// that knownFields does not list, e.g. ones a later Kubernetes version
	// added.
	UnknownFields []string
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
	JWT []json.RawMessage `json:"jwt"`
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
	EgressSelectorType   string   `json:"egressSelectorType"`
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

// knownFields lists, per object of a JWT authenticator (as a dotted path from
// the authenticator), the fields the AuthenticationConfiguration v1 API
// defines. The content of the validation rules and of the uid and extra
// mappings is not checked: Issuer only records whether they are set.
var knownFields = map[string][]string{
	"":                       {"issuer", "claimValidationRules", "claimMappings", "userValidationRules"},
	"issuer":                 {"url", "discoveryURL", "certificateAuthority", "audiences", "audienceMatchPolicy", "egressSelectorType"},
	"claimMappings":          {"username", "groups", "uid", "extra"},
	"claimMappings.username": {"claim", "prefix", "expression"},
	"claimMappings.groups":   {"claim", "prefix", "expression"},
}

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
		for _, raw := range authConfig.JWT {
			var jwt jwtAuthenticator
			if err := json.Unmarshal(raw, &jwt); err != nil {
				return nil, microerror.Mask(err)
			}
			if jwt.Issuer.URL == "" {
				continue
			}
			unknown, err := unknownFields(raw)
			if err != nil {
				return nil, microerror.Mask(err)
			}
			issuers = append(issuers, Issuer{
				URL:                  jwt.Issuer.URL,
				Audiences:            jwt.Issuer.Audiences,
				CertificateAuthority: jwt.Issuer.CertificateAuthority,
				DiscoveryURL:         jwt.Issuer.DiscoveryURL,
				EgressSelectorType:   jwt.Issuer.EgressSelectorType,
				Username:             ClaimMapping(jwt.ClaimMappings.Username),
				Groups:               ClaimMapping(jwt.ClaimMappings.Groups),
				MapsUIDOrExtra:       jwt.ClaimMappings.UID != nil || len(jwt.ClaimMappings.Extra) > 0,
				ClaimValidationRules: len(jwt.ClaimValidationRules),
				UserValidationRules:  len(jwt.UserValidationRules),
				UnknownFields:        unknown,
			})
		}
		return issuers, nil
	}

	return nil, nil
}

// unknownFields returns the fields of the JWT authenticator that knownFields
// does not list, as sorted dotted paths.
func unknownFields(raw json.RawMessage) ([]string, error) {
	var authenticator object
	if err := json.Unmarshal(raw, &authenticator); err != nil {
		return nil, microerror.Mask(err)
	}

	var unknown []string
	for _, path := range slices.Sorted(maps.Keys(knownFields)) {
		obj := authenticator
		if path != "" {
			for _, key := range strings.Split(path, ".") {
				obj, _ = obj[key].(object)
			}
		}
		for _, field := range slices.Sorted(maps.Keys(obj)) {
			if !slices.Contains(knownFields[path], field) {
				unknown = append(unknown, strings.TrimPrefix(path+"."+field, "."))
			}
		}
	}
	return unknown, nil
}
