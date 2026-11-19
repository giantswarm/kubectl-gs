package cluster

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/giantswarm/microerror"
	releasev1alpha1 "github.com/giantswarm/releases/sdk/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake" //nolint:staticcheck

	"github.com/giantswarm/kubectl-gs/v6/internal/key"
	"github.com/giantswarm/kubectl-gs/v6/internal/label"
	"github.com/giantswarm/kubectl-gs/v6/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/v6/pkg/data/domain/cluster"
	"github.com/giantswarm/kubectl-gs/v6/pkg/output"
	"github.com/giantswarm/kubectl-gs/v6/pkg/scheme"
	"github.com/giantswarm/kubectl-gs/v6/test/kubeconfig"
)

func Test_run(t *testing.T) {
	var testCases = []struct {
		name    string
		storage []runtime.Object
		flags   flag
	}{
		{
			name:    "update cluster with a scheduled time",
			storage: []runtime.Object{newCluster("abcd1", "default", "16.0.1"), newAWSCluster("abcd1", "default", "16.0.1")},
			flags:   flag{Name: "abcd1", ReleaseVersion: "16.1.0", ScheduledTime: "2022-01-01 01:00", Provider: "aws"},
		},
		{
			name:    "update cluster immediately",
			storage: []runtime.Object{newCluster("abcd1", "default", "16.0.1"), newAWSCluster("abcd1", "default", "16.0.1")},
			flags:   flag{Name: "abcd1", ReleaseVersion: "16.1.0", Provider: "aws"},
		},
	}

	for i, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprintf("case %d: %s", i, tc.name), func(t *testing.T) {
			var err error

			ctx := context.TODO()

			fakeKubeConfig := kubeconfig.CreateFakeKubeConfig()

			flag := &tc.flags
			flag.print = genericclioptions.NewPrintFlags("").WithDefaultOutput(output.TypeDefault)

			ctrlClient := newFakeClient(t, tc.storage...)
			out := new(bytes.Buffer)
			runner := &runner{
				commonConfig: commonconfig.New(genericclioptions.NewTestConfigFlags().WithClientConfig(fakeKubeConfig)),
				flag:         flag,
				stdout:       out,
				client:       ctrlClient,
				service:      cluster.New(cluster.Config{Client: ctrlClient}),
			}

			err = runner.run(ctx, nil, []string{})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func newCluster(name, namespace, targetRelease string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "cluster.x-k8s.io/v1beta2",
			"kind":       "Cluster",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
				"labels": map[string]interface{}{
					key.ClusterNameLabel: name,
					label.ReleaseVersion: "16.0.1",
				},
				"annotations": map[string]interface{}{
					"cluster.giantswarm.io/description": "fake-cluster",
				},
			},
		},
	}
}

func newAWSCluster(name, namespace, targetRelease string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "infrastructure.giantswarm.io/v1alpha3",
			"kind":       "AWSCluster",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
				"labels": map[string]interface{}{
					label.Cluster: name,
				},
				"annotations": map[string]interface{}{
					"cluster.giantswarm.io/description": "fake-cluster",
				},
			},
		},
	}
}

const capiValues = `global:
  metadata:
    description: "fake-cluster"
  release:
    version: %s
  providerSpecific:
    ami:
      version: 33.4.0
`

func Test_run_capi(t *testing.T) {
	var testCases = []struct {
		name          string
		labelVersion  string
		valuesVersion string
		releases      []string
		flags         flag
		wantVersion   string
		wantErr       *microerror.Error
	}{
		{
			name:          "update to an existing release",
			labelVersion:  "33.4.0",
			valuesVersion: "33.4.0",
			releases:      []string{"aws-33.4.0", "aws-34.5.0"},
			flags:         flag{Name: "abcd1", ReleaseVersion: "34.5.0"},
			wantVersion:   "34.5.0",
		},
		{
			name:          "values ahead of the Cluster label after a failed update",
			labelVersion:  "33.4.0",
			valuesVersion: "34.3.0",
			releases:      []string{"aws-33.4.0", "aws-34.5.0"},
			flags:         flag{Name: "abcd1", ReleaseVersion: "34.5.0"},
			wantVersion:   "34.5.0",
		},
		{
			name:          "a release that does not exist",
			labelVersion:  "33.4.0",
			valuesVersion: "33.4.0",
			releases:      []string{"aws-33.4.0", "aws-34.5.0"},
			flags:         flag{Name: "abcd1", ReleaseVersion: "34.3.0"},
			wantErr:       notFoundError,
		},
		{
			name:          "a scheduled update to a release that does not exist",
			labelVersion:  "33.4.0",
			valuesVersion: "33.4.0",
			releases:      []string{"aws-33.4.0"},
			flags:         flag{Name: "abcd1", ReleaseVersion: "34.3.0", ScheduledTime: "2022-01-01 01:00"},
			wantErr:       notFoundError,
		},
		{
			name:          "the current release",
			labelVersion:  "34.5.0",
			valuesVersion: "34.5.0",
			releases:      []string{"aws-34.5.0"},
			flags:         flag{Name: "abcd1", ReleaseVersion: "34.5.0"},
			wantErr:       notAllowedError,
		},
		{
			name:          "an older release",
			labelVersion:  "34.5.0",
			valuesVersion: "34.5.0",
			releases:      []string{"aws-33.4.0", "aws-34.5.0"},
			flags:         flag{Name: "abcd1", ReleaseVersion: "33.4.0"},
			wantErr:       notAllowedError,
		},
		{
			name:         "values without a release version",
			labelVersion: "33.4.0",
			releases:     []string{"aws-34.5.0"},
			flags:        flag{Name: "abcd1", ReleaseVersion: "34.5.0"},
			wantErr:      notFoundError,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("case %d: %s", i, tc.name), func(t *testing.T) {
			ctx := context.TODO()

			values := "global: {}\n"
			if tc.valuesVersion != "" {
				values = fmt.Sprintf(capiValues, tc.valuesVersion)
			}
			storage := []runtime.Object{
				newCAPICluster("abcd1", "default", tc.labelVersion),
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: "abcd1-userconfig", Namespace: "default"},
					Data:       map[string]string{"values": values},
				},
			}
			for _, name := range tc.releases {
				storage = append(storage, &releasev1alpha1.Release{ObjectMeta: metav1.ObjectMeta{Name: name}})
			}

			ctrlClient := newFakeClient(t, storage...)
			flag := &tc.flags
			flag.print = genericclioptions.NewPrintFlags("").WithDefaultOutput(output.TypeDefault)
			configFlags := genericclioptions.NewTestConfigFlags().WithClientConfig(kubeconfig.CreateFakeKubeConfig())

			out := new(bytes.Buffer)
			runner := &runner{
				commonConfig: commonconfig.New(configFlags),
				flag:         flag,
				stdout:       out,
				client:       ctrlClient,
				service:      cluster.New(cluster.Config{Client: ctrlClient}),
			}

			err := runner.run(ctx, nil, []string{})

			cm := &corev1.ConfigMap{}
			getErr := ctrlClient.Get(ctx, client.ObjectKey{Name: "abcd1-userconfig", Namespace: "default"}, cm)
			if getErr != nil {
				t.Fatal(getErr)
			}

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("want %s, got %v", tc.wantErr.Kind, err)
				}
				if cm.Data["values"] != values {
					t.Fatalf("values changed on error:\n%s", cm.Data["values"])
				}
				if out.Len() != 0 {
					t.Fatalf("printed on error: %q", out.String())
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if want := fmt.Sprintf(capiValues, tc.wantVersion); cm.Data["values"] != want {
				t.Fatalf("values:\n%s\nwant:\n%s", cm.Data["values"], want)
			}
		})
	}
}

func Test_setReleaseVersion(t *testing.T) {
	var testCases = []struct {
		name        string
		values      string
		wantCurrent string
		wantValues  string
	}{
		{
			name:        "plain",
			values:      "global:\n  release:\n    version: 33.4.0\n",
			wantCurrent: "33.4.0",
			wantValues:  "global:\n  release:\n    version: 34.5.0\n",
		},
		{
			name:        "double quoted, comment and other version keys kept",
			values:      "# cluster\nglobal:\n  apps:\n    version: 33.4.0\n  release:\n    version: \"33.4.0\" # pinned\n",
			wantCurrent: "33.4.0",
			wantValues:  "# cluster\nglobal:\n  apps:\n    version: 33.4.0\n  release:\n    version: \"34.5.0\" # pinned\n",
		},
		{
			name:        "single quoted, no trailing newline",
			values:      "global:\n  release:\n    version: '33.4.0'",
			wantCurrent: "33.4.0",
			wantValues:  "global:\n  release:\n    version: '34.5.0'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			current, values, err := setReleaseVersion(tc.values, "34.5.0")
			if err != nil {
				t.Fatal(err)
			}
			if current != tc.wantCurrent {
				t.Fatalf("current %q, want %q", current, tc.wantCurrent)
			}
			if values != tc.wantValues {
				t.Fatalf("values:\n%s\nwant:\n%s", values, tc.wantValues)
			}
		})
	}
}

func newCAPICluster(name, namespace, releaseVersion string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "cluster.x-k8s.io/v1beta2",
			"kind":       "Cluster",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
				"labels": map[string]interface{}{
					key.ClusterNameLabel:            name,
					label.ReleaseVersion:            releaseVersion,
					"cluster.x-k8s.io/watch-filter": "capi",
				},
			},
			"spec": map[string]interface{}{
				"infrastructureRef": map[string]interface{}{
					"apiGroup": "infrastructure.cluster.x-k8s.io",
					"kind":     "AWSCluster",
					"name":     name,
				},
			},
		},
	}
}

func newFakeClient(t *testing.T, object ...runtime.Object) client.Client {
	clientScheme, err := scheme.NewScheme()
	if err != nil {
		t.Fatalf("unexpected error: %s", microerror.Pretty(err, true))
	}
	return fake.NewClientBuilder().WithScheme(clientScheme).WithRuntimeObjects(object...).Build()
}

func newClusterHelmRelease(name, namespace, repo string, values map[string]interface{}, valuesFrom ...interface{}) *unstructured.Unstructured {
	hr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"spec": map[string]interface{}{
				"chartRef": map[string]interface{}{
					"kind": "OCIRepository",
					"name": repo,
				},
				"valuesFrom": append([]interface{}{
					map[string]interface{}{"kind": "ConfigMap", "name": name + "-userconfig", "valuesKey": "values"},
					map[string]interface{}{"kind": "Secret", "name": name + "-credentials"},
				}, valuesFrom...),
			},
		},
	}
	hr.SetGroupVersionKind(helmReleaseGVK)
	hr.SetName(name)
	hr.SetNamespace(namespace)
	if values != nil {
		_ = unstructured.SetNestedMap(hr.Object, values, "spec", "values")
	}
	return hr
}

func newClusterOCIRepository(name, namespace, chart string, ref map[string]interface{}, labels map[string]string) *unstructured.Unstructured {
	repo := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"spec": map[string]interface{}{
				"url": "oci://gsoci.azurecr.io/charts/giantswarm/" + chart,
				"ref": ref,
			},
		},
	}
	repo.SetGroupVersionKind(ociRepositoryGVK)
	repo.SetName(name)
	repo.SetNamespace(namespace)
	repo.SetLabels(labels)
	return repo
}

func Test_run_helmRelease(t *testing.T) {
	const userValues = "global:\n  metadata:\n    name: abcd1\n"
	releaseVersion := map[string]interface{}{"global": map[string]interface{}{"release": map[string]interface{}{"version": "33.4.0"}}}

	var testCases = []struct {
		name string
		// labelVersion is the release the Cluster reports.
		labelVersion string
		chart        string
		tag          string
		// ref replaces the OCIRepository's spec.ref {tag: tag}.
		ref        map[string]interface{}
		repoLabels map[string]string
		// noRepo leaves the OCIRepository out.
		noRepo     bool
		hrValues   map[string]interface{}
		valuesFrom []interface{}
		// otherHelmRelease takes its chart from the cluster's OCIRepository too.
		otherHelmRelease bool
		// servedV1beta2 reads the OCIRepository as v1beta2 after the run, as the
		// interceptor of the case serves it only so.
		servedV1beta2 bool
		userValues    string
		userLabels    map[string]string
		releases      []string
		flags         flag
		interceptor   interceptor.Funcs
		wantTag       string
		wantValues    string
		wantErr       *microerror.Error
	}{
		{
			name:     "update to an existing release",
			chart:    "release-aws",
			tag:      "33.4.0",
			releases: []string{"aws-33.4.0", "aws-34.5.0"},
			flags:    flag{Name: "abcd1", ReleaseVersion: "34.5.0"},
			wantTag:  "34.5.0",
		},
		{
			name:     "a scheduled update, which nothing carries out for a release chart",
			chart:    "release-aws",
			tag:      "33.4.0",
			releases: []string{"aws-33.4.0", "aws-34.5.0"},
			flags:    flag{Name: "abcd1", ReleaseVersion: "34.5.0", ScheduledTime: "2022-01-01 01:00"},
			wantErr:  notAllowedError,
		},
		{
			name:     "a release that does not exist",
			chart:    "release-aws",
			tag:      "33.4.0",
			releases: []string{"aws-33.4.0"},
			flags:    flag{Name: "abcd1", ReleaseVersion: "34.5.0"},
			wantErr:  notFoundError,
		},
		{
			name:         "an older release",
			labelVersion: "34.5.0",
			chart:        "release-aws",
			tag:          "34.5.0",
			releases:     []string{"aws-33.4.0", "aws-34.5.0"},
			flags:        flag{Name: "abcd1", ReleaseVersion: "33.4.0"},
			wantErr:      notAllowedError,
		},
		{
			name:       "an OCIRepository applied by a Flux Kustomization",
			chart:      "release-aws",
			tag:        "33.4.0",
			repoLabels: map[string]string{"kustomize.toolkit.fluxcd.io/name": "clusters"},
			releases:   []string{"aws-34.5.0"},
			flags:      flag{Name: "abcd1", ReleaseVersion: "34.5.0"},
			wantErr:    notAllowedError,
		},
		{
			name:       "an OCIRepository applied by a HelmRelease",
			chart:      "release-aws",
			tag:        "33.4.0",
			repoLabels: map[string]string{"helm.toolkit.fluxcd.io/name": "clusters"},
			releases:   []string{"aws-34.5.0"},
			flags:      flag{Name: "abcd1", ReleaseVersion: "34.5.0"},
			wantErr:    notAllowedError,
		},
		{
			name:     "an OCIRepository that follows a semver range",
			chart:    "release-aws",
			tag:      "33.4.0",
			ref:      map[string]interface{}{"tag": "33.4.0", "semver": ">=33.0.0"},
			releases: []string{"aws-34.5.0"},
			flags:    flag{Name: "abcd1", ReleaseVersion: "34.5.0"},
			wantErr:  notAllowedError,
		},
		{
			name:     "an OCIRepository pinned to a digest",
			chart:    "release-aws",
			tag:      "33.4.0",
			ref:      map[string]interface{}{"tag": "33.4.0", "digest": "sha256:0000000000000000000000000000000000000000000000000000000000000000"},
			releases: []string{"aws-34.5.0"},
			flags:    flag{Name: "abcd1", ReleaseVersion: "34.5.0"},
			wantErr:  notAllowedError,
		},
		{
			name:             "an OCIRepository another HelmRelease takes its chart from",
			chart:            "release-aws",
			tag:              "33.4.0",
			otherHelmRelease: true,
			releases:         []string{"aws-34.5.0"},
			flags:            flag{Name: "abcd1", ReleaseVersion: "34.5.0"},
			wantErr:          notAllowedError,
		},
		{
			name:     "a missing OCIRepository",
			chart:    "release-aws",
			tag:      "33.4.0",
			noRepo:   true,
			releases: []string{"aws-34.5.0"},
			flags:    flag{Name: "abcd1", ReleaseVersion: "34.5.0"},
			wantErr:  notFoundError,
		},
		{
			name:          "an OCIRepository served as v1beta2 only, before Flux 2.6",
			chart:         "release-aws",
			tag:           "33.4.0",
			servedV1beta2: true,
			releases:      []string{"aws-33.4.0", "aws-34.5.0"},
			flags:         flag{Name: "abcd1", ReleaseVersion: "34.5.0"},
			interceptor: interceptor.Funcs{
				Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					gvk := obj.GetObjectKind().GroupVersionKind()
					if gvk == ociRepositoryGVK {
						return &meta.NoKindMatchError{GroupKind: ociRepositoryGVK.GroupKind(), SearchedVersions: []string{"v1"}}
					}
					if gvk == ociRepositoryV1beta2GVK {
						// The fake client stores the object as v1.
						u := obj.(*unstructured.Unstructured)
						u.SetGroupVersionKind(ociRepositoryGVK)
						err := c.Get(ctx, key, u, opts...)
						u.SetGroupVersionKind(ociRepositoryV1beta2GVK)
						return err
					}
					return c.Get(ctx, key, obj, opts...)
				},
				Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
					if u, ok := obj.(*unstructured.Unstructured); ok && u.GroupVersionKind() == ociRepositoryV1beta2GVK {
						u.SetGroupVersionKind(ociRepositoryGVK)
					}
					return c.Update(ctx, obj, opts...)
				},
			},
			wantTag: "34.5.0",
		},
		{
			name:       "user values that set the release version",
			chart:      "release-aws",
			tag:        "33.4.0",
			userValues: fmt.Sprintf(capiValues, "33.4.0"),
			releases:   []string{"aws-34.5.0"},
			flags:      flag{Name: "abcd1", ReleaseVersion: "34.5.0"},
			wantErr:    invalidConfigError,
		},
		{
			name:     "inline values that set the release version",
			chart:    "release-aws",
			tag:      "33.4.0",
			hrValues: releaseVersion,
			releases: []string{"aws-34.5.0"},
			flags:    flag{Name: "abcd1", ReleaseVersion: "34.5.0"},
			wantErr:  invalidConfigError,
		},
		{
			name:       "a targetPath that sets the release version",
			chart:      "release-aws",
			tag:        "33.4.0",
			valuesFrom: []interface{}{map[string]interface{}{"kind": "Secret", "name": "release", "targetPath": "global.release.version"}},
			releases:   []string{"aws-34.5.0"},
			flags:      flag{Name: "abcd1", ReleaseVersion: "34.5.0"},
			wantErr:    invalidConfigError,
		},
		{
			name:       "a targetPath above the release version",
			chart:      "release-aws",
			tag:        "33.4.0",
			valuesFrom: []interface{}{map[string]interface{}{"kind": "ConfigMap", "name": "release", "targetPath": "global.release"}},
			releases:   []string{"aws-34.5.0"},
			flags:      flag{Name: "abcd1", ReleaseVersion: "34.5.0"},
			wantErr:    invalidConfigError,
		},
		{
			name:       "a targetPath elsewhere, whose key holds a single value",
			chart:      "release-aws",
			tag:        "33.4.0",
			valuesFrom: []interface{}{map[string]interface{}{"kind": "ConfigMap", "name": "description", "valuesKey": "text", "targetPath": "global.metadata.description"}},
			releases:   []string{"aws-33.4.0", "aws-34.5.0"},
			flags:      flag{Name: "abcd1", ReleaseVersion: "34.5.0"},
			wantTag:    "34.5.0",
		},
		{
			name:       "a cluster chart sets the release in the values and keeps the chart version",
			chart:      "cluster-aws",
			tag:        "3.6.0",
			userValues: fmt.Sprintf(capiValues, "33.4.0"),
			releases:   []string{"aws-33.4.0", "aws-34.5.0"},
			flags:      flag{Name: "abcd1", ReleaseVersion: "34.5.0"},
			wantTag:    "3.6.0",
			wantValues: fmt.Sprintf(capiValues, "34.5.0"),
		},
		{
			name:     "a cluster chart without a release version in the values",
			chart:    "cluster-aws",
			tag:      "3.6.0",
			releases: []string{"aws-34.5.0"},
			flags:    flag{Name: "abcd1", ReleaseVersion: "34.5.0"},
			wantErr:  notFoundError,
		},
		{
			name:       "a cluster chart with values a Flux Kustomization applies",
			chart:      "cluster-aws",
			tag:        "3.6.0",
			userValues: fmt.Sprintf(capiValues, "33.4.0"),
			userLabels: map[string]string{"kustomize.toolkit.fluxcd.io/name": "clusters"},
			releases:   []string{"aws-34.5.0"},
			flags:      flag{Name: "abcd1", ReleaseVersion: "34.5.0"},
			wantErr:    notAllowedError,
		},
		{
			name:       "a scheduled update of values a Flux Kustomization applies, which it doesn't change",
			chart:      "cluster-aws",
			tag:        "3.6.0",
			userValues: fmt.Sprintf(capiValues, "33.4.0"),
			userLabels: map[string]string{"kustomize.toolkit.fluxcd.io/name": "clusters"},
			releases:   []string{"aws-34.5.0"},
			flags:      flag{Name: "abcd1", ReleaseVersion: "34.5.0", ScheduledTime: "2022-01-01 01:00"},
			wantTag:    "3.6.0",
		},
		{
			name:       "a user who may not read HelmReleases updates the values, as before",
			chart:      "release-aws",
			tag:        "33.4.0",
			userValues: fmt.Sprintf(capiValues, "33.4.0"),
			releases:   []string{"aws-33.4.0", "aws-34.5.0"},
			flags:      flag{Name: "abcd1", ReleaseVersion: "34.5.0"},
			interceptor: interceptor.Funcs{
				Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					if obj.GetObjectKind().GroupVersionKind() == helmReleaseGVK {
						return apierrors.NewForbidden(helmReleaseGVK.GroupVersion().WithResource("helmreleases").GroupResource(), key.Name, errors.New("no access"))
					}
					return c.Get(ctx, key, obj, opts...)
				},
			},
			wantTag:    "33.4.0",
			wantValues: fmt.Sprintf(capiValues, "34.5.0"),
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("case %d: %s", i, tc.name), func(t *testing.T) {
			ctx := context.TODO()

			values := tc.userValues
			if values == "" {
				values = userValues
			}
			labelVersion := tc.labelVersion
			if labelVersion == "" {
				labelVersion = "33.4.0"
			}
			ref := tc.ref
			if ref == nil {
				ref = map[string]interface{}{"tag": tc.tag}
			}
			storage := []runtime.Object{
				newCAPICluster("abcd1", "default", labelVersion),
				newClusterHelmRelease("abcd1", "default", "abcd1", tc.hrValues, tc.valuesFrom...),
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: "abcd1-userconfig", Namespace: "default", Labels: tc.userLabels},
					Data:       map[string]string{"values": values},
				},
			}
			if !tc.noRepo {
				storage = append(storage, newClusterOCIRepository("abcd1", "default", tc.chart, ref, tc.repoLabels))
			}
			if tc.otherHelmRelease {
				storage = append(storage, newClusterHelmRelease("efgh2", "default", "abcd1", nil))
			}
			for _, name := range tc.releases {
				storage = append(storage, &releasev1alpha1.Release{ObjectMeta: metav1.ObjectMeta{Name: name}})
			}

			clientScheme, err := scheme.NewScheme()
			if err != nil {
				t.Fatal(err)
			}
			ctrlClient := fake.NewClientBuilder().WithScheme(clientScheme).WithRuntimeObjects(storage...).WithInterceptorFuncs(tc.interceptor).Build()
			flag := &tc.flags
			flag.print = genericclioptions.NewPrintFlags("").WithDefaultOutput(output.TypeDefault)
			configFlags := genericclioptions.NewTestConfigFlags().WithClientConfig(kubeconfig.CreateFakeKubeConfig())

			out := new(bytes.Buffer)
			runner := &runner{
				commonConfig: commonconfig.New(configFlags),
				flag:         flag,
				stdout:       out,
				client:       ctrlClient,
				service:      cluster.New(cluster.Config{Client: ctrlClient}),
			}

			err = runner.run(ctx, nil, []string{})

			var tag string
			if !tc.noRepo {
				repo := &unstructured.Unstructured{}
				repo.SetGroupVersionKind(ociRepositoryGVK)
				if tc.servedV1beta2 {
					repo.SetGroupVersionKind(ociRepositoryV1beta2GVK)
				}
				getErr := ctrlClient.Get(ctx, client.ObjectKey{Name: "abcd1", Namespace: "default"}, repo)
				if getErr != nil {
					t.Fatal(getErr)
				}
				tag, _, _ = unstructured.NestedString(repo.Object, "spec", "ref", "tag")
			}

			cm := &corev1.ConfigMap{}
			getErr := ctrlClient.Get(ctx, client.ObjectKey{Name: "abcd1-userconfig", Namespace: "default"}, cm)
			if getErr != nil {
				t.Fatal(getErr)
			}
			wantValues := tc.wantValues
			if wantValues == "" || tc.wantErr != nil {
				wantValues = values
			}
			if cm.Data["values"] != wantValues {
				t.Fatalf("user values:\n%s\nwant:\n%s", cm.Data["values"], wantValues)
			}

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("want %s, got %v", tc.wantErr.Kind, err)
				}
				if tag != tc.tag && !tc.noRepo {
					t.Fatalf("tag changed on error to %q", tag)
				}
				if out.Len() != 0 {
					t.Fatalf("printed on error: %q", out.String())
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tag != tc.wantTag {
				t.Fatalf("tag %q, want %q", tag, tc.wantTag)
			}
		})
	}
}
