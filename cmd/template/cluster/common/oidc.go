package common

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/kubectl-gs/v6/pkg/structuredauth"
)

// ManagementClusterNamespace holds the management cluster's own Cluster API
// resources.
const ManagementClusterNamespace = "org-giantswarm"

// ErrManagementClusterOIDC wraps every reason the OIDC issuer the management
// cluster trusts cannot be read.
var ErrManagementClusterOIDC = errors.New("cannot read the OIDC issuer the management cluster trusts")

// IsManagementClusterOIDC asserts ErrManagementClusterOIDC.
func IsManagementClusterOIDC(err error) bool {
	return errors.Is(err, ErrManagementClusterOIDC)
}

// OIDCValues is the cluster chart's global.controlPlane.oidc: the OIDC
// issuer the workload cluster's API server trusts.
type OIDCValues struct {
	IssuerURL     string `json:"issuerUrl"`
	CAPem         string `json:"caPem,omitempty"`
	ClientID      string `json:"clientId"`
	UsernameClaim string `json:"usernameClaim,omitempty"`
	GroupsClaim   string `json:"groupsClaim,omitempty"`
}

// kubeadmControlPlaneVersions are tried in order: v1beta2 is the version
// current Cluster API serves, v1beta1 the one older management clusters do.
var kubeadmControlPlaneVersions = []string{"v1beta2", "v1beta1"}

// ManagementClusterIssuer is the OIDC issuer read from the management
// cluster, with what the user should be told about how it was chosen.
type ManagementClusterIssuer struct {
	Values OIDCValues
	// ManagementCluster is the name of the management cluster's
	// KubeadmControlPlane.
	ManagementCluster string
	// Source says where the issuer was read: the API server's oidc-*
	// arguments, or the AuthenticationConfiguration file.
	Source string
	// ClientIDOverride is set when the client ID asked for is none the
	// management cluster's issuer accepts: it lists those it accepts.
	ClientIDOverride []string
	// Skipped lists the other structured authentication issuers, which are
	// not copied.
	Skipped []string
}

// ManagementClusterOIDC reads the OIDC issuer the management cluster's own
// API server trusts for its Dex, from the management cluster's
// KubeadmControlPlane in the org-giantswarm namespace.
//
// mcName names the management cluster. When it is empty, the only
// KubeadmControlPlane in org-giantswarm is used.
//
// With structured authentication the issuers are in the
// AuthenticationConfiguration file the authentication-config argument names,
// instead of the oidc-* arguments. Only the management cluster's Dex,
// https://dex.<management cluster domain>, is copied; see dexIssuer.
//
// clientID, when given, is the client ID the workload cluster uses. It selects
// one of the client IDs the management cluster's Dex accepts, or else
// replaces them, which ClientIDOverride reports.
func ManagementClusterOIDC(ctx context.Context, ctrlClient client.Client, mcName, clientID string) (ManagementClusterIssuer, error) {
	kcp, err := getManagementClusterControlPlane(ctx, ctrlClient, mcName)
	if err != nil {
		return ManagementClusterIssuer{}, microerror.Mask(err)
	}

	args, err := apiServerExtraArgs(kcp)
	if err != nil {
		return ManagementClusterIssuer{}, microerror.Mask(err)
	}

	if args["oidc-issuer-url"] != "" && args["oidc-client-id"] != "" {
		result := ManagementClusterIssuer{
			Values: OIDCValues{
				IssuerURL:     args["oidc-issuer-url"],
				ClientID:      args["oidc-client-id"],
				UsernameClaim: args["oidc-username-claim"],
				GroupsClaim:   args["oidc-groups-claim"],
			},
			ManagementCluster: kcp.GetName(),
			Source:            "the API server's oidc-* arguments",
		}
		if clientID != "" && clientID != result.Values.ClientID {
			result.ClientIDOverride = []string{result.Values.ClientID}
			result.Values.ClientID = clientID
		}
		return result, nil
	}

	path := args["authentication-config"]
	if path == "" {
		return ManagementClusterIssuer{}, fmt.Errorf("%w: the API server of management cluster %q trusts no complete OIDC issuer: KubeadmControlPlane %s/%s sets no oidc-issuer-url or oidc-client-id",
			ErrManagementClusterOIDC, kcp.GetName(), kcp.GetNamespace(), kcp.GetName())
	}

	issuers, err := structuredauth.IssuersAt(kcp, path)
	if err != nil {
		return ManagementClusterIssuer{}, fmt.Errorf("%w: KubeadmControlPlane %s/%s: cannot read %s: %w",
			ErrManagementClusterOIDC, kcp.GetNamespace(), kcp.GetName(), path, err)
	}
	if len(issuers) == 0 {
		// Either the file is not written inline (e.g. it comes from a
		// Secret) or it lists no issuer: the KubeadmControlPlane does not
		// tell which issuers the API server trusts.
		return ManagementClusterIssuer{}, fmt.Errorf("%w: KubeadmControlPlane %s/%s writes no JWT issuer inline to %s, the file the API server's authentication-config argument names",
			ErrManagementClusterOIDC, kcp.GetNamespace(), kcp.GetName(), path)
	}

	dexURL, err := managementClusterDexURL(kcp)
	if err != nil {
		return ManagementClusterIssuer{}, microerror.Mask(err)
	}

	result, err := dexIssuer(issuers, dexURL, clientID)
	if err != nil {
		return ManagementClusterIssuer{}, fmt.Errorf("%w: the API server of management cluster %q: %s: %w", ErrManagementClusterOIDC, kcp.GetName(), path, err)
	}
	result.ManagementCluster = kcp.GetName()
	result.Source = path
	return result, nil
}

// managementClusterDexURL returns the URL of the management cluster's Dex,
// https://dex.<management cluster domain>. The cluster chart always lists
// api.<management cluster domain> among the API server's certificate SANs,
// where the management cluster domain is its name followed by its base
// domain.
func managementClusterDexURL(kcp *unstructured.Unstructured) (string, error) {
	sans, _, err := unstructured.NestedStringSlice(kcp.Object, "spec", "kubeadmConfigSpec", "clusterConfiguration", "apiServer", "certSANs")
	if err != nil {
		return "", microerror.Mask(err)
	}

	prefix := "api." + kcp.GetName() + "."
	for _, san := range sans {
		if strings.HasPrefix(san, prefix) && len(san) > len(prefix) {
			return "https://dex." + strings.TrimPrefix(san, "api."), nil
		}
	}

	return "", fmt.Errorf("%w: cannot tell the URL of management cluster %q's Dex: the API server certificate SANs of KubeadmControlPlane %s/%s list no %s<base domain>",
		ErrManagementClusterOIDC, kcp.GetName(), kcp.GetNamespace(), kcp.GetName(), prefix)
}

// dexIssuer returns the structured authentication issuer of the management
// cluster's Dex, at dexURL. It fails rather than copy any other issuer, and
// fails when the workload cluster's OIDC values cannot express Dex exactly
// (see valuesGaps).
//
// A clientID that is one of Dex's audiences selects it. Any other clientID
// replaces them, unless another issuer has it as an audience: then the user
// most likely meant that issuer, which is not copied. Without a clientID, Dex
// must have a single audience.
func dexIssuer(issuers []structuredauth.Issuer, dexURL, clientID string) (ManagementClusterIssuer, error) {
	var dex *structuredauth.Issuer
	var others []structuredauth.Issuer
	for i, iss := range issuers {
		if dex == nil && strings.TrimSuffix(iss.URL, "/") == dexURL {
			dex = &issuers[i]
			continue
		}
		others = append(others, iss)
	}

	if dex == nil {
		return ManagementClusterIssuer{}, fmt.Errorf("no JWT issuer is the management cluster's Dex, %s. The file lists:\n%s",
			dexURL, issuerList(issuers))
	}

	if gaps := valuesGaps(*dex); len(gaps) > 0 {
		return ManagementClusterIssuer{}, fmt.Errorf("the management cluster's Dex, %s, cannot be copied to the workload cluster's OIDC settings: it sets %s",
			dex.URL, strings.Join(gaps, ", "))
	}

	result := ManagementClusterIssuer{
		Values: OIDCValues{
			IssuerURL:     dex.URL,
			CAPem:         dex.CertificateAuthority,
			UsernameClaim: dex.Username.Claim,
			GroupsClaim:   dex.Groups.Claim,
		},
	}
	for _, iss := range others {
		result.Skipped = append(result.Skipped, iss.URL)
	}

	switch {
	case clientID != "" && slices.Contains(dex.Audiences, clientID):
		result.Values.ClientID = clientID
	case clientID != "":
		for _, iss := range others {
			if slices.Contains(iss.Audiences, clientID) {
				return ManagementClusterIssuer{}, fmt.Errorf("client ID %q belongs to %s, not to the management cluster's Dex, %s, whose client IDs are %s. Only Dex is copied; to trust %s, name it with --oidc-issuer-url",
					clientID, iss.URL, dex.URL, strings.Join(dex.Audiences, ", "), iss.URL)
			}
		}
		result.Values.ClientID = clientID
		result.ClientIDOverride = dex.Audiences
	case len(dex.Audiences) == 1:
		result.Values.ClientID = dex.Audiences[0]
	default:
		return ManagementClusterIssuer{}, fmt.Errorf("the management cluster's Dex, %s, accepts several client IDs: %s. Select one with --oidc-client-id",
			dex.URL, strings.Join(dex.Audiences, ", "))
	}

	return result, nil
}

// issuerList lists the issuers' URLs, one per line.
func issuerList(issuers []structuredauth.Issuer) string {
	var lines []string
	for _, iss := range issuers {
		lines = append(lines, "  - "+iss.URL)
	}
	return strings.Join(lines, "\n")
}

// valuesGaps lists what of the issuer the workload cluster's OIDC values
// (global.controlPlane.oidc: issuer URL, CA, client ID, username and groups
// claims, without prefixes) cannot express. Copying such an issuer would make
// the workload cluster accept other tokens, or map them to other identities,
// than the management cluster does.
func valuesGaps(iss structuredauth.Issuer) []string {
	var gaps []string
	if len(iss.Audiences) == 0 {
		gaps = append(gaps, "no audience")
	}
	if iss.DiscoveryURL != "" {
		gaps = append(gaps, "discovery URL")
	}
	if iss.EgressSelectorType != "" {
		gaps = append(gaps, "egress selector type")
	}
	if iss.ClaimValidationRules > 0 {
		gaps = append(gaps, "claim validation rules")
	}
	if iss.UserValidationRules > 0 {
		gaps = append(gaps, "user validation rules")
	}
	if iss.Username.Expression != "" {
		gaps = append(gaps, "username CEL expression")
	} else if iss.Username.Claim != "email" {
		// Without a prefix, the API server prefixes every username claim
		// but email with the issuer URL.
		gaps = append(gaps, fmt.Sprintf("username claim %q instead of email", iss.Username.Claim))
	}
	if iss.Username.Prefix != "" {
		gaps = append(gaps, "username prefix")
	}
	if iss.Groups.Expression != "" {
		gaps = append(gaps, "groups CEL expression")
	}
	if iss.Groups.Prefix != "" {
		gaps = append(gaps, "groups prefix")
	}
	if iss.MapsUIDOrExtra {
		gaps = append(gaps, "uid or extra mappings")
	}
	if len(iss.UnknownFields) > 0 {
		// A field kubectl-gs does not know may restrict which tokens the
		// issuer accepts.
		gaps = append(gaps, "unknown fields "+strings.Join(iss.UnknownFields, ", "))
	}
	return gaps
}

func getManagementClusterControlPlane(ctx context.Context, ctrlClient client.Client, mcName string) (*unstructured.Unstructured, error) {
	var lastErr error
	for _, version := range kubeadmControlPlaneVersions {
		gvk := schema.GroupVersionKind{Group: "controlplane.cluster.x-k8s.io", Version: version, Kind: "KubeadmControlPlane"}

		kcp, err := findControlPlane(ctx, ctrlClient, gvk, mcName)
		if meta.IsNoMatchError(err) {
			lastErr = err
			continue
		}
		return kcp, microerror.Mask(err)
	}

	return nil, fmt.Errorf("%w: the cluster the kubeconfig context points at serves no KubeadmControlPlane resource: %w", ErrManagementClusterOIDC, lastErr)
}

func findControlPlane(ctx context.Context, ctrlClient client.Client, gvk schema.GroupVersionKind, mcName string) (*unstructured.Unstructured, error) {
	if mcName != "" {
		kcp := &unstructured.Unstructured{}
		kcp.SetGroupVersionKind(gvk)
		err := ctrlClient.Get(ctx, client.ObjectKey{Namespace: ManagementClusterNamespace, Name: mcName}, kcp)
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("%w: no KubeadmControlPlane %s/%s found: the kubeconfig context does not point at management cluster %q", ErrManagementClusterOIDC, ManagementClusterNamespace, mcName, mcName)
		} else if err != nil {
			return nil, unreachable(err)
		}
		return kcp, nil
	}

	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(gvk.GroupVersion().WithKind(gvk.Kind + "List"))
	err := ctrlClient.List(ctx, list, client.InNamespace(ManagementClusterNamespace))
	if err != nil {
		return nil, unreachable(err)
	}

	switch len(list.Items) {
	case 0:
		return nil, fmt.Errorf("%w: no KubeadmControlPlane found in namespace %s: the kubeconfig context does not point at a management cluster", ErrManagementClusterOIDC, ManagementClusterNamespace)
	case 1:
		return &list.Items[0], nil
	default:
		return nil, fmt.Errorf("%w: %d KubeadmControlPlanes found in namespace %s: name the management cluster with --management-cluster", ErrManagementClusterOIDC, len(list.Items), ManagementClusterNamespace)
	}
}

// unreachable wraps a failed request, keeping the RESTMapper's no-match error
// recognisable for the version fallback.
func unreachable(err error) error {
	if meta.IsNoMatchError(err) {
		return err
	}
	return fmt.Errorf("%w: the management cluster did not answer: %w", ErrManagementClusterOIDC, err)
}

// apiServerExtraArgs returns the API server's extra arguments. v1beta2 lists
// them as name/value pairs, v1beta1 as a map.
func apiServerExtraArgs(kcp *unstructured.Unstructured) (map[string]string, error) {
	raw, _, err := unstructured.NestedFieldNoCopy(kcp.Object, "spec", "kubeadmConfigSpec", "clusterConfiguration", "apiServer", "extraArgs")
	if err != nil {
		return nil, microerror.Mask(err)
	}

	args := map[string]string{}
	switch extraArgs := raw.(type) {
	case map[string]interface{}:
		for name, value := range extraArgs {
			args[name] = fmt.Sprint(value)
		}
	case []interface{}:
		for _, item := range extraArgs {
			arg, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			name, _ := arg["name"].(string)
			value, _ := arg["value"].(string)
			args[name] = value
		}
	}

	return args, nil
}
