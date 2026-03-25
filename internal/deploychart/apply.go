package deploychart

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/yaml"
)

const fieldManager = "kubectl-gs"

// knownResources maps Kind to plural resource name for the resource types
// this command creates.
var knownResources = map[string]string{
	"OCIRepository": "ocirepositories",
	"HelmRelease":   "helmreleases",
	"Secret":        "secrets",
}

// ApplyOptions controls the apply behavior.
type ApplyOptions struct {
	DryRun bool
}

// ApplyManifest applies a YAML manifest to the cluster using server-side apply.
// If dryRun is true, uses dry-run=server and does not persist.
func ApplyManifest(ctx context.Context, dynClient dynamic.Interface, manifestYAML []byte, opts ApplyOptions) error {
	// Convert YAML to JSON (required by unstructured).
	jsonData, err := yaml.YAMLToJSON(manifestYAML)
	if err != nil {
		return fmt.Errorf("converting manifest to JSON: %w", err)
	}

	obj := &unstructured.Unstructured{}
	if err := json.Unmarshal(jsonData, &obj.Object); err != nil {
		return fmt.Errorf("parsing manifest: %w", err)
	}

	gvr, err := ResourceGVR(obj.GetAPIVersion(), obj.GetKind())
	if err != nil {
		return err
	}

	patchOpts := metav1.PatchOptions{
		FieldManager: fieldManager,
	}
	if opts.DryRun {
		patchOpts.DryRun = []string{"All"}
	}

	force := true
	patchOpts.Force = &force

	data, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}

	_, err = dynClient.Resource(gvr).Namespace(obj.GetNamespace()).Patch(
		ctx, obj.GetName(), types.ApplyPatchType, data, patchOpts,
	)
	if err != nil {
		return fmt.Errorf("applying %s %s/%s: %w", obj.GetKind(), obj.GetNamespace(), obj.GetName(), err)
	}

	return nil
}

// GetExistingResource fetches a resource from the cluster.
// Returns nil, nil if the resource does not exist.
func GetExistingResource(ctx context.Context, dynClient dynamic.Interface, gvr schema.GroupVersionResource, namespace, name string) (*unstructured.Unstructured, error) {
	obj, err := dynClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting %s %s/%s: %w", gvr.Resource, namespace, name, err)
	}
	return obj, nil
}

// ResourceGVR derives the GroupVersionResource from an apiVersion and kind.
func ResourceGVR(apiVersion, kind string) (schema.GroupVersionResource, error) {
	resource, ok := knownResources[kind]
	if !ok {
		return schema.GroupVersionResource{}, fmt.Errorf("unknown resource kind %q", kind)
	}

	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("parsing apiVersion %q: %w", apiVersion, err)
	}

	return gv.WithResource(resource), nil
}
