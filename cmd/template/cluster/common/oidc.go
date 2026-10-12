package common

import (
	"context"
	"errors"
	"fmt"

	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
func ManagementClusterOIDC(ctx context.Context, ctrlClient client.Client, mcName string) (OIDCValues, error) {
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
	if values.IssuerURL == "" || values.ClientID == "" {
		return OIDCValues{}, fmt.Errorf("%w: the API server of management cluster %q trusts no complete OIDC issuer: KubeadmControlPlane %s/%s sets no oidc-issuer-url or oidc-client-id",
			ErrManagementClusterOIDC, kcp.GetName(), kcp.GetNamespace(), kcp.GetName())
	}

	return values, nil
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
