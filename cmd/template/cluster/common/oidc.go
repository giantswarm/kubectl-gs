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

// ManagementClusterOIDC reads the OIDC issuer the management cluster's own
// API server trusts (its Dex) from the management cluster's
// KubeadmControlPlane in the org-giantswarm namespace.
//
// mcName names the management cluster. When it is empty, the only
// KubeadmControlPlane in org-giantswarm is used.
//
// With structured authentication the issuers are in the
// AuthenticationConfiguration file the authentication-config argument names,
// instead of the oidc-* arguments; see selectIssuer for how clientID, when
// given, selects one.
func ManagementClusterOIDC(ctx context.Context, ctrlClient client.Client, mcName, clientID string) (OIDCValues, error) {
	kcp, err := getManagementClusterControlPlane(ctx, ctrlClient, mcName)
	if err != nil {
		return OIDCValues{}, microerror.Mask(err)
	}

	args, err := apiServerExtraArgs(kcp)
	if err != nil {
		return OIDCValues{}, microerror.Mask(err)
	}

	values := OIDCValues{
		IssuerURL:     args["oidc-issuer-url"],
		ClientID:      args["oidc-client-id"],
		UsernameClaim: args["oidc-username-claim"],
		GroupsClaim:   args["oidc-groups-claim"],
	}
	if values.IssuerURL != "" && values.ClientID != "" {
		return values, nil
	}

	path := args["authentication-config"]
	if path == "" {
		return OIDCValues{}, fmt.Errorf("%w: the API server of management cluster %q trusts no complete OIDC issuer: KubeadmControlPlane %s/%s sets no oidc-issuer-url or oidc-client-id",
			ErrManagementClusterOIDC, kcp.GetName(), kcp.GetNamespace(), kcp.GetName())
	}

	issuers, err := structuredauth.IssuersAt(kcp, path)
	if err != nil {
		return OIDCValues{}, fmt.Errorf("%w: KubeadmControlPlane %s/%s: cannot read %s: %w",
			ErrManagementClusterOIDC, kcp.GetNamespace(), kcp.GetName(), path, err)
	}
	if len(issuers) == 0 {
		return OIDCValues{}, fmt.Errorf("%w: the API server of management cluster %q trusts no complete OIDC issuer: KubeadmControlPlane %s/%s sets no JWT issuer in %s",
			ErrManagementClusterOIDC, kcp.GetName(), kcp.GetNamespace(), kcp.GetName(), path)
	}

	issuer, err := selectIssuer(issuers, clientID)
	if err != nil {
		return OIDCValues{}, fmt.Errorf("%w: the API server of management cluster %q: %s: %w", ErrManagementClusterOIDC, kcp.GetName(), path, err)
	}

	return OIDCValues{
		IssuerURL:     issuer.URL,
		CAPem:         issuer.CertificateAuthority,
		ClientID:      issuer.ClientID,
		UsernameClaim: issuer.Username.Claim,
		GroupsClaim:   issuer.Groups.Claim,
	}, nil
}

// selectIssuer picks the structured authentication issuer the workload
// cluster trusts. Only issuers that the workload cluster's OIDC values express
// exactly are eligible (see valuesGaps). With one eligible issuer, it is taken
// whatever clientID is: clientID then only overrides its client ID, as with a
// management cluster that uses the oidc-* arguments. With several, clientID
// selects the one with that audience.
func selectIssuer(issuers []structuredauth.Issuer, clientID string) (structuredauth.Issuer, error) {
	var eligible []structuredauth.Issuer
	var ineligible []string
	for _, iss := range issuers {
		if gaps := valuesGaps(iss); len(gaps) > 0 {
			ineligible = append(ineligible, fmt.Sprintf("%s (%s)", iss.URL, strings.Join(gaps, ", ")))
			continue
		}
		eligible = append(eligible, iss)
	}

	switch len(eligible) {
	case 0:
		return structuredauth.Issuer{}, fmt.Errorf("no JWT issuer can be copied to the workload cluster's OIDC settings: %s",
			strings.Join(ineligible, "; "))
	case 1:
		return eligible[0], nil
	}

	var matching []structuredauth.Issuer
	var available []string
	for _, iss := range eligible {
		if clientID != "" && slices.Contains(iss.Audiences, clientID) {
			matching = append(matching, iss)
		}
		available = append(available, fmt.Sprintf("%s (audiences %s)", iss.URL, strings.Join(iss.Audiences, ", ")))
	}
	if len(matching) == 1 {
		return matching[0], nil
	}

	return structuredauth.Issuer{}, fmt.Errorf("%d JWT issuers can be copied to the workload cluster's OIDC settings; select one by its audience with --oidc-client-id: %s",
		len(eligible), strings.Join(available, "; "))
}

// valuesGaps lists what of the issuer the workload cluster's OIDC values
// (global.controlPlane.oidc: issuer URL, CA, client ID, username and groups
// claims, without prefixes) cannot express. Copying such an issuer would make
// the workload cluster accept other tokens, or map them to other identities,
// than the management cluster does.
func valuesGaps(iss structuredauth.Issuer) []string {
	var gaps []string
	if iss.ClientID == "" {
		gaps = append(gaps, "no audience")
	}
	if iss.DiscoveryURL != "" {
		gaps = append(gaps, "discovery URL")
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
