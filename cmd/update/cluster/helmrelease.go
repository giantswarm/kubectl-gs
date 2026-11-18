package cluster

import (
	"context"
	"path"
	"strings"

	"github.com/giantswarm/microerror"
	"go.yaml.in/yaml/v4"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	helmReleaseGVK   = schema.GroupVersionKind{Group: "helm.toolkit.fluxcd.io", Version: "v2", Kind: "HelmRelease"}
	ociRepositoryGVK = schema.GroupVersionKind{Group: "source.toolkit.fluxcd.io", Version: "v1", Kind: "OCIRepository"}
)

// fluxKustomizationLabel is set by kustomize-controller on the resources a
// Flux Kustomization applies.
const fluxKustomizationLabel = "kustomize.toolkit.fluxcd.io/name"

// clusterHelmRelease returns the HelmRelease that deploys the cluster chart,
// or nil when the cluster is deployed another way. The cluster chart names
// the cluster after its Helm release, and the HelmRelease is named after it.
func clusterHelmRelease(ctx context.Context, c client.Client, capiCluster *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	hr := &unstructured.Unstructured{}
	hr.SetGroupVersionKind(helmReleaseGVK)
	err := c.Get(ctx, client.ObjectKey{Name: capiCluster.GetName(), Namespace: capiCluster.GetNamespace()}, hr)
	if apierrors.IsNotFound(err) || meta.IsNoMatchError(err) {
		return nil, nil
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	return hr, nil
}

// releaseChartUpdate returns the release version of a cluster deployed as a
// HelmRelease of a release-<provider> chart, and its OCIRepository with the
// tag set to target. The release chart is published with the version of the
// release it holds, and sets global.release.version itself.
//
// It returns no OCIRepository when the HelmRelease deploys another chart, such
// as cluster-<provider>. Its version is then not a release version, and the
// release is set in the values, as for an App CR. The chart version stays as
// it is, like an App CR with a version set by hand.
func releaseChartUpdate(ctx context.Context, c client.Client, hr *unstructured.Unstructured, target string) (*unstructured.Unstructured, string, error) {
	kind, _, _ := unstructured.NestedString(hr.Object, "spec", "chartRef", "kind")
	name, _, _ := unstructured.NestedString(hr.Object, "spec", "chartRef", "name")
	if kind != ociRepositoryGVK.Kind || name == "" {
		return nil, "", nil
	}
	namespace, _, _ := unstructured.NestedString(hr.Object, "spec", "chartRef", "namespace")
	if namespace == "" {
		namespace = hr.GetNamespace()
	}

	repo := &unstructured.Unstructured{}
	repo.SetGroupVersionKind(ociRepositoryGVK)
	err := c.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, repo)
	if err != nil {
		return nil, "", microerror.Mask(err)
	}

	url, _, _ := unstructured.NestedString(repo.Object, "spec", "url")
	if !strings.HasPrefix(path.Base(url), "release-") {
		return nil, "", nil
	}

	// Flux would revert a change made here on its next reconciliation.
	if kustomization, ok := repo.GetLabels()[fluxKustomizationLabel]; ok {
		return nil, "", microerror.Maskf(notAllowedError, "OCIRepository '%s/%s' is managed by the Flux Kustomization '%s'. Change its spec.ref.tag in the repository the Kustomization applies.", namespace, name, kustomization)
	}

	current, _, _ := unstructured.NestedString(repo.Object, "spec", "ref", "tag")
	if current == "" {
		return nil, "", microerror.Maskf(notFoundError, "OCIRepository '%s/%s' has no spec.ref.tag.", namespace, name)
	}

	err = ensureNoReleaseVersionValues(ctx, c, hr)
	if err != nil {
		return nil, "", microerror.Mask(err)
	}

	err = unstructured.SetNestedField(repo.Object, strings.TrimPrefix(target, "v"), "spec", "ref", "tag")
	if err != nil {
		return nil, "", microerror.Mask(err)
	}

	return repo, current, nil
}

// ensureNoReleaseVersionValues fails when the HelmRelease's values set
// global.release.version: they override the value the release chart sets,
// and the cluster would keep the components of that release. Secrets are
// not read.
func ensureNoReleaseVersionValues(ctx context.Context, c client.Client, hr *unstructured.Unstructured) error {
	values, _, _ := unstructured.NestedMap(hr.Object, "spec", "values")
	if hasReleaseVersion(values) {
		return microerror.Maskf(invalidConfigError, "HelmRelease '%s' sets %s in spec.values, which overrides the release chart's. Remove it first.", hr.GetName(), strings.Join(releaseVersionPath, "."))
	}

	valuesFrom, _, _ := unstructured.NestedSlice(hr.Object, "spec", "valuesFrom")
	for _, entry := range valuesFrom {
		ref, ok := entry.(map[string]interface{})
		if !ok || ref["kind"] != "ConfigMap" {
			continue
		}
		name, _ := ref["name"].(string)
		valuesKey, _ := ref["valuesKey"].(string)
		if valuesKey == "" {
			valuesKey = "values.yaml"
		}

		cm := &unstructured.Unstructured{}
		cm.SetAPIVersion("v1")
		cm.SetKind("ConfigMap")
		err := c.Get(ctx, client.ObjectKey{Name: name, Namespace: hr.GetNamespace()}, cm)
		if apierrors.IsNotFound(err) {
			continue
		} else if err != nil {
			return microerror.Mask(err)
		}

		data, _, _ := unstructured.NestedString(cm.Object, "data", valuesKey)
		var doc map[string]interface{}
		err = yaml.Unmarshal([]byte(data), &doc)
		if err != nil {
			return microerror.Maskf(invalidConfigError, "ConfigMap '%s' key %s is not valid YAML: %s", name, valuesKey, err)
		}
		if hasReleaseVersion(doc) {
			return microerror.Maskf(invalidConfigError, "ConfigMap '%s' sets %s, which overrides the release chart's. Remove it first.", name, strings.Join(releaseVersionPath, "."))
		}
	}

	return nil
}

func hasReleaseVersion(values map[string]interface{}) bool {
	var node interface{} = values
	for _, key := range releaseVersionPath {
		m, ok := node.(map[string]interface{})
		if !ok {
			return false
		}
		node, ok = m[key]
		if !ok {
			return false
		}
	}
	return node != nil && node != ""
}
