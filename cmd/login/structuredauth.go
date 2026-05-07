package login

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/giantswarm/k8sclient/v8/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// structuredAuthIssuer holds the selected issuer URL and client ID for the OIDC flow.
type structuredAuthIssuer struct {
	IssuerURL string
	ClientID  string
}

// authenticationConfiguration mirrors the relevant parts of the Kubernetes
// AuthenticationConfiguration API used in structured authentication.
type authenticationConfiguration struct {
	JWT []jwtAuthenticator `json:"jwt"`
}

type jwtAuthenticator struct {
	Issuer jwtIssuer `json:"issuer"`
}

type jwtIssuer struct {
	URL       string   `json:"url"`
	Audiences []string `json:"audiences"`
}

// detectStructuredAuth determines whether a workload cluster uses Kubernetes
// structured authentication. It returns a structuredAuthIssuer if detected, or
// nil if not (signalling fallback to client-cert).
//
// When issuerOverride and clientIDOverride are both provided the values are used
// directly and no KubeadmControlPlane resource is fetched from the MC.
func detectStructuredAuth(ctx context.Context, k8sClient k8sclient.Interface, clusterName, namespace, issuerOverride, clientIDOverride string) (*structuredAuthIssuer, error) {
	// If both issuer and client-id are provided via flags, use them directly
	// without needing management cluster access for KCP detection.
	if issuerOverride != "" && clientIDOverride != "" {
		return &structuredAuthIssuer{
			IssuerURL: issuerOverride,
			ClientID:  clientIDOverride,
		}, nil
	}

	// If only one of issuer/client-id is provided, we still need to fetch
	// from KCP to fill in the missing value.
	kcp, err := fetchKubeadmControlPlane(ctx, k8sClient, clusterName, namespace)
	if err != nil {
		if errors.IsNotFound(err) || meta.IsNoMatchError(err) {
			// KCP not found or CRD not installed — cluster doesn't use
			// KubeadmControlPlane. Fall through to client-cert.
			return nil, nil
		}
		return nil, microerror.Mask(err)
	}

	issuers, err := parseAuthenticationConfig(kcp)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	if len(issuers) == 0 {
		return nil, nil
	}

	// If issuer was provided via flag, find matching entry or use it directly.
	if issuerOverride != "" {
		for _, iss := range issuers {
			if iss.IssuerURL == issuerOverride {
				return &structuredAuthIssuer{
					IssuerURL: issuerOverride,
					ClientID:  iss.ClientID,
				}, nil
			}
		}
		// Issuer flag provided but not found in KCP; use it anyway with the
		// first audience from the first issuer as a best-effort fallback.
		return &structuredAuthIssuer{
			IssuerURL: issuerOverride,
			ClientID:  issuers[0].ClientID,
		}, nil
	}

	// If client-id was provided via flag, find the matching issuer.
	if clientIDOverride != "" {
		for _, iss := range issuers {
			if iss.ClientID == clientIDOverride {
				return &structuredAuthIssuer{
					IssuerURL: iss.IssuerURL,
					ClientID:  clientIDOverride,
				}, nil
			}
		}
		return nil, microerror.Maskf(structuredAuthIssuerNotFoundError,
			"no structured auth issuer found with client-id %q; available: %s",
			clientIDOverride, formatAvailableIssuers(issuers))
	}

	// Auto-selection: single issuer.
	if len(issuers) == 1 {
		return &issuers[0], nil
	}

	return nil, microerror.Maskf(structuredAuthMultipleIssuersError,
		"multiple OIDC issuers detected; use --%s or --%s to select one:\n%s",
		flagWCOIDCClientID, flagWCOIDCIssuer,
		formatAvailableIssuers(issuers))
}

// fetchKubeadmControlPlane retrieves the KubeadmControlPlane resource for
// the given cluster from the management cluster.
func fetchKubeadmControlPlane(ctx context.Context, k8sClient k8sclient.Interface, clusterName, namespace string) (*unstructured.Unstructured, error) {
	kcp := &unstructured.Unstructured{}
	kcp.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "controlplane.cluster.x-k8s.io",
		Version: "v1beta2",
		Kind:    "KubeadmControlPlane",
	})

	err := k8sClient.CtrlClient().Get(ctx, client.ObjectKey{
		Name:      clusterName,
		Namespace: namespace,
	}, kcp)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return kcp, nil
}

// parseAuthenticationConfig extracts OIDC issuer information from the
// KubeadmControlPlane's files list. It looks for the auth-config.yaml file
// and parses its content as an AuthenticationConfiguration.
func parseAuthenticationConfig(kcp *unstructured.Unstructured) ([]structuredAuthIssuer, error) {
	files, found, err := unstructured.NestedSlice(kcp.Object, "spec", "kubeadmConfigSpec", "files")
	if err != nil || !found {
		return nil, nil
	}

	for _, f := range files {
		fileMap, ok := f.(map[string]interface{})
		if !ok {
			continue
		}
		path, ok := fileMap["path"].(string)
		if !ok || path != "/etc/kubernetes/policies/auth-config.yaml" {
			continue
		}
		content, ok := fileMap["content"].(string)
		if !ok || content == "" {
			continue
		}

		// Handle base64-encoded content (encoding field set to "base64").
		if encoding, _ := fileMap["encoding"].(string); encoding == "base64" {
			decoded, err := base64.StdEncoding.DecodeString(content)
			if err != nil {
				return nil, microerror.Mask(err)
			}
			content = string(decoded)
		}

		var authConfig authenticationConfiguration
		if err := yaml.Unmarshal([]byte(content), &authConfig); err != nil {
			return nil, microerror.Mask(err)
		}

		var issuers []structuredAuthIssuer
		for _, jwt := range authConfig.JWT {
			if jwt.Issuer.URL == "" {
				continue
			}
			clientID := ""
			if len(jwt.Issuer.Audiences) > 0 {
				clientID = jwt.Issuer.Audiences[0]
			}
			issuers = append(issuers, structuredAuthIssuer{
				IssuerURL: jwt.Issuer.URL,
				ClientID:  clientID,
			})
		}
		return issuers, nil
	}

	return nil, nil
}

// fetchClusterCA retrieves the CA certificate for the workload cluster.
// If caFilePath is provided, it reads the CA from that file instead of
// fetching from the management cluster ConfigMap.
func fetchClusterCA(ctx context.Context, k8sClient k8sclient.Interface, clusterName, namespace, caFilePath string) ([]byte, error) {
	if caFilePath != "" {
		caData, err := os.ReadFile(caFilePath)
		if err != nil {
			return nil, microerror.Maskf(structuredAuthCANotFoundError, "failed to read CA file %q: %s", caFilePath, err.Error())
		}
		return caData, nil
	}

	var cm v1.ConfigMap
	err := k8sClient.CtrlClient().Get(ctx, client.ObjectKey{
		Name:      fmt.Sprintf("%s-cluster-values", clusterName),
		Namespace: namespace,
	}, &cm)
	if err != nil {
		return nil, microerror.Maskf(structuredAuthCANotFoundError,
			"failed to fetch ConfigMap %s-cluster-values in namespace %s: %s",
			clusterName, namespace, err.Error())
	}

	if valuesData, ok := cm.Data["values"]; ok {
		ca := extractCAFromValues(valuesData)
		if ca != "" {
			return []byte(ca), nil
		}
	}

	if ca, ok := cm.Data["clusterCA"]; ok && ca != "" {
		return []byte(ca), nil
	}

	return nil, microerror.Maskf(structuredAuthCANotFoundError,
		"could not find cluster CA in ConfigMap %s-cluster-values", clusterName)
}

func extractCAFromValues(valuesData string) string {
	var values map[string]interface{}
	if err := yaml.Unmarshal([]byte(valuesData), &values); err != nil {
		return ""
	}

	if ca, ok := values["clusterCA"].(string); ok && ca != "" {
		return ca
	}

	if cluster, ok := values["cluster"].(map[string]interface{}); ok {
		if ca, ok := cluster["ca"].(string); ok && ca != "" {
			return ca
		}
	}

	return ""
}

func formatAvailableIssuers(issuers []structuredAuthIssuer) string {
	var parts []string
	for _, iss := range issuers {
		parts = append(parts, fmt.Sprintf("  - issuer: %s, client-id: %s", iss.IssuerURL, iss.ClientID))
	}
	return strings.Join(parts, "\n")
}
