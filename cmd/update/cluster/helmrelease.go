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
	// ociRepositoryV1beta2GVK is the OCIRepository API before Flux 2.6. HelmRelease
	// v2, the first API with spec.chartRef, came with Flux 2.3.
	ociRepositoryV1beta2GVK = schema.GroupVersionKind{Group: "source.toolkit.fluxcd.io", Version: "v1beta2", Kind: "OCIRepository"}
)

// managerLabels are set by the Flux controllers on the resources they apply,
// which they revert to what their source says on their next reconciliation.
var managerLabels = []struct{ label, owner string }{
	{"kustomize.toolkit.fluxcd.io/name", "Flux Kustomization"},
	{"helm.toolkit.fluxcd.io/name", "HelmRelease"},
}

// ensureNotManaged fails when a Flux controller applies obj: a change made
// here would be reverted on its next reconciliation, and the command would
// report an update that doesn't happen.
func ensureNotManaged(obj client.Object, kind, field string) error {
	for _, m := range managerLabels {
		if owner, ok := obj.GetLabels()[m.label]; ok {
			return microerror.Maskf(notAllowedError, "%s '%s/%s' is managed by the %s '%s'. Change its %s in the source that %s applies.", kind, obj.GetNamespace(), obj.GetName(), m.owner, owner, field, m.owner)
		}
	}
	return nil
}

// clusterHelmRelease returns the HelmRelease that deploys the cluster chart,
// or nil when the cluster is deployed another way. The cluster chart names
// the cluster after its Helm release, and the HelmRelease is named after it.
// A user who may not read HelmReleases updates the cluster as an App CR one,
// as before HelmReleases were looked up.
func clusterHelmRelease(ctx context.Context, c client.Client, capiCluster *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	hr := &unstructured.Unstructured{}
	hr.SetGroupVersionKind(helmReleaseGVK)
	err := c.Get(ctx, client.ObjectKey{Name: capiCluster.GetName(), Namespace: capiCluster.GetNamespace()}, hr)
	if apierrors.IsNotFound(err) || meta.IsNoMatchError(err) || apierrors.IsForbidden(err) {
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

	repo, err := getOCIRepository(ctx, c, hr, client.ObjectKey{Name: name, Namespace: namespace})
	if err != nil {
		return nil, "", microerror.Mask(err)
	}

	url, _, _ := unstructured.NestedString(repo.Object, "spec", "url")
	if !strings.HasPrefix(path.Base(url), "release-") {
		return nil, "", nil
	}

	err = ensureNotManaged(repo, "OCIRepository", "spec.ref.tag")
	if err != nil {
		return nil, "", microerror.Mask(err)
	}

	// Flux follows a digest over a semver range, and both over a tag.
	for _, field := range []string{"digest", "semver"} {
		if v, _, _ := unstructured.NestedString(repo.Object, "spec", "ref", field); v != "" {
			return nil, "", microerror.Maskf(notAllowedError, "OCIRepository '%s/%s' sets spec.ref.%s %q, which Flux follows instead of spec.ref.tag, so changing the tag wouldn't update the cluster. Change spec.ref by hand.", namespace, name, field, v)
		}
	}

	current, _, _ := unstructured.NestedString(repo.Object, "spec", "ref", "tag")
	if current == "" {
		return nil, "", microerror.Maskf(notFoundError, "OCIRepository '%s/%s' has no spec.ref.tag.", namespace, name)
	}

	err = ensureOnlyUser(ctx, c, hr, repo)
	if err != nil {
		return nil, "", microerror.Mask(err)
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

// getOCIRepository returns the OCIRepository the HelmRelease takes its chart
// from, in the API version the management cluster serves.
func getOCIRepository(ctx context.Context, c client.Client, hr *unstructured.Unstructured, key client.ObjectKey) (*unstructured.Unstructured, error) {
	var err error
	for _, gvk := range []schema.GroupVersionKind{ociRepositoryGVK, ociRepositoryV1beta2GVK} {
		repo := &unstructured.Unstructured{}
		repo.SetGroupVersionKind(gvk)
		err = c.Get(ctx, key, repo)
		if meta.IsNoMatchError(err) {
			continue
		} else if apierrors.IsNotFound(err) {
			return nil, microerror.Maskf(notFoundError, "OCIRepository '%s/%s', which HelmRelease '%s/%s' takes its chart from, does not exist.", key.Namespace, key.Name, hr.GetNamespace(), hr.GetName())
		} else if err != nil {
			return nil, microerror.Mask(err)
		}
		return repo, nil
	}

	return nil, microerror.Mask(err)
}

// ensureOnlyUser fails when another HelmRelease takes its chart from the
// OCIRepository: changing its tag would update that release too. HelmReleases
// are looked for in the namespaces of the cluster's HelmRelease and of the
// OCIRepository.
func ensureOnlyUser(ctx context.Context, c client.Client, hr, repo *unstructured.Unstructured) error {
	namespaces := []string{hr.GetNamespace()}
	if repo.GetNamespace() != hr.GetNamespace() {
		namespaces = append(namespaces, repo.GetNamespace())
	}

	for _, namespace := range namespaces {
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(helmReleaseGVK.GroupVersion().WithKind(helmReleaseGVK.Kind + "List"))
		err := c.List(ctx, list, client.InNamespace(namespace))
		if err != nil {
			return microerror.Mask(err)
		}

		for _, other := range list.Items {
			if other.GetNamespace() == hr.GetNamespace() && other.GetName() == hr.GetName() {
				continue
			}
			kind, _, _ := unstructured.NestedString(other.Object, "spec", "chartRef", "kind")
			name, _, _ := unstructured.NestedString(other.Object, "spec", "chartRef", "name")
			refNamespace, _, _ := unstructured.NestedString(other.Object, "spec", "chartRef", "namespace")
			if refNamespace == "" {
				refNamespace = other.GetNamespace()
			}
			if kind == ociRepositoryGVK.Kind && name == repo.GetName() && refNamespace == repo.GetNamespace() {
				return microerror.Maskf(notAllowedError, "OCIRepository '%s/%s' is also the chart source of HelmRelease '%s/%s', which changing its tag would update too. Give the cluster's HelmRelease an OCIRepository of its own.", repo.GetNamespace(), repo.GetName(), other.GetNamespace(), other.GetName())
			}
		}
	}

	return nil
}

// ensureNoReleaseVersionValues fails when the HelmRelease's values set
// global.release.version: they override the value the release chart sets,
// and the cluster would keep the components of that release. Secrets are
// not read, but a valuesFrom targetPath is checked whatever its kind.
func ensureNoReleaseVersionValues(ctx context.Context, c client.Client, hr *unstructured.Unstructured) error {
	releaseVersion := strings.Join(releaseVersionPath, ".")

	values, _, _ := unstructured.NestedMap(hr.Object, "spec", "values")
	if hasReleaseVersion(values) {
		return microerror.Maskf(invalidConfigError, "HelmRelease '%s' sets %s in spec.values, which overrides the release chart's. Remove it first.", hr.GetName(), releaseVersion)
	}

	valuesFrom, _, _ := unstructured.NestedSlice(hr.Object, "spec", "valuesFrom")
	for _, entry := range valuesFrom {
		ref, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		kind, _ := ref["kind"].(string)
		name, _ := ref["name"].(string)

		// With a targetPath, the key holds the single value Flux sets at that
		// path, not a values document.
		if targetPath, _ := ref["targetPath"].(string); targetPath != "" {
			if targetPath == releaseVersion || strings.HasPrefix(releaseVersion, targetPath+".") {
				return microerror.Maskf(invalidConfigError, "HelmRelease '%s' sets %s from %s '%s' through targetPath %s, which overrides the release chart's. Remove it first.", hr.GetName(), releaseVersion, kind, name, targetPath)
			}
			continue
		}

		if kind != "ConfigMap" {
			continue
		}
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
			return microerror.Maskf(invalidConfigError, "ConfigMap '%s' sets %s, which overrides the release chart's. Remove it first.", name, releaseVersion)
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
