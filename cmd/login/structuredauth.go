package login

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strconv"
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
// structured authentication.
//
// Return value semantics:
//   - (resolved, nil, nil): a single issuer was resolved (via flags, single
//     entry in KCP, or flag-matched entry).
//   - (nil, candidates, nil): multiple issuers were found and no flag was
//     provided to disambiguate; the caller decides how to pick one (e.g.
//     interactive prompt) or surface an error.
//   - (nil, nil, nil): the cluster does not use structured authentication
//     (caller should fall back to client-cert).
//   - (nil, nil, err): an actual error occurred.
//
// When issuerOverride and clientIDOverride are both provided the values are used
// directly and no KubeadmControlPlane resource is fetched from the MC.
func detectStructuredAuth(ctx context.Context, k8sClient k8sclient.Interface, clusterName, namespace, issuerOverride, clientIDOverride string) (*structuredAuthIssuer, []structuredAuthIssuer, error) {
	// If both issuer and client-id are provided via flags, use them directly
	// without needing management cluster access for KCP detection.
	if issuerOverride != "" && clientIDOverride != "" {
		return &structuredAuthIssuer{
			IssuerURL: issuerOverride,
			ClientID:  clientIDOverride,
		}, nil, nil
	}

	// If only one of issuer/client-id is provided, we still need to fetch
	// from KCP to fill in the missing value.
	kcp, err := fetchKubeadmControlPlane(ctx, k8sClient, clusterName, namespace)
	if err != nil {
		if errors.IsNotFound(err) || meta.IsNoMatchError(err) {
			// KCP not found or CRD not installed — cluster doesn't use
			// KubeadmControlPlane. Fall through to client-cert.
			return nil, nil, nil
		}
		return nil, nil, microerror.Mask(err)
	}

	issuers, err := parseAuthenticationConfig(kcp)
	if err != nil {
		return nil, nil, microerror.Mask(err)
	}

	if len(issuers) == 0 {
		return nil, nil, nil
	}

	// If issuer was provided via flag, find matching entry or use it directly.
	if issuerOverride != "" {
		for _, iss := range issuers {
			if iss.IssuerURL == issuerOverride {
				return &structuredAuthIssuer{
					IssuerURL: issuerOverride,
					ClientID:  iss.ClientID,
				}, nil, nil
			}
		}
		// Issuer flag provided but not found in KCP; use it anyway with the
		// first audience from the first issuer as a best-effort fallback.
		return &structuredAuthIssuer{
			IssuerURL: issuerOverride,
			ClientID:  issuers[0].ClientID,
		}, nil, nil
	}

	// If client-id was provided via flag, find the matching issuer.
	if clientIDOverride != "" {
		for _, iss := range issuers {
			if iss.ClientID == clientIDOverride {
				return &structuredAuthIssuer{
					IssuerURL: iss.IssuerURL,
					ClientID:  clientIDOverride,
				}, nil, nil
			}
		}
		return nil, nil, microerror.Maskf(structuredAuthIssuerNotFoundError,
			"no OIDC issuer matches --%s %q; available issuers:\n%s",
			flagWCOIDCClientID, clientIDOverride, formatAvailableIssuers(issuers))
	}

	// Auto-selection: single issuer.
	if len(issuers) == 1 {
		return &issuers[0], nil, nil
	}

	// Multiple issuers and no flag to disambiguate. Let the caller decide
	// how to pick one (interactive prompt or scripted error).
	return nil, issuers, nil
}

// pickIssuerInteractive prints a numbered menu of issuers and reads a
// selection from the provided reader. Used when stdin is a TTY.
func pickIssuerInteractive(issuers []structuredAuthIssuer, in io.Reader, out io.Writer) (*structuredAuthIssuer, error) {
	if len(issuers) == 0 {
		return nil, microerror.Maskf(structuredAuthIssuerNotFoundError, "no OIDC issuers to choose from")
	}
	if len(issuers) == 1 {
		return &issuers[0], nil
	}

	_, _ = fmt.Fprintln(out, "Multiple OIDC issuers are configured for this cluster:")
	for i, iss := range issuers {
		_, _ = fmt.Fprintf(out, "  %d) issuer: %s, client-id: %s\n", i+1, iss.IssuerURL, iss.ClientID)
	}
	_, _ = fmt.Fprintf(out, "Select an issuer [1-%d]: ", len(issuers))

	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, microerror.Mask(err)
		}
		return nil, microerror.Maskf(structuredAuthIssuerNotFoundError, "no issuer selected")
	}

	line := strings.TrimSpace(scanner.Text())
	n, err := strconv.Atoi(line)
	if err != nil || n < 1 || n > len(issuers) {
		return nil, microerror.Maskf(structuredAuthIssuerNotFoundError,
			"invalid selection %q; expected a number between 1 and %d",
			line, len(issuers))
	}

	return &issuers[n-1], nil
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
			"could not auto-detect the API server CA for cluster %q; pass --%s to provide it directly: %s",
			clusterName, flagWCOIDCCAFile, err.Error())
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
		"could not auto-detect the API server CA for cluster %q; pass --%s to provide it directly",
		clusterName, flagWCOIDCCAFile)
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
