package cluster

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

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

func newClusterHelmRelease(name, namespace string, values map[string]interface{}) *unstructured.Unstructured {
	hr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"spec": map[string]interface{}{
				"chartRef": map[string]interface{}{
					"kind": "OCIRepository",
					"name": name,
				},
				"valuesFrom": []interface{}{
					map[string]interface{}{"kind": "ConfigMap", "name": name + "-userconfig", "valuesKey": "values"},
					map[string]interface{}{"kind": "Secret", "name": name + "-credentials"},
				},
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

func newClusterOCIRepository(name, namespace, chart, tag string, labels map[string]string) *unstructured.Unstructured {
	repo := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"spec": map[string]interface{}{
				"url": "oci://gsoci.azurecr.io/charts/giantswarm/" + chart,
				"ref": map[string]interface{}{"tag": tag},
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

	var testCases = []struct {
		name       string
		chart      string
		tag        string
		repoLabels map[string]string
		hrValues   map[string]interface{}
		userValues string
		releases   []string
		flags      flag
		wantTag    string
		wantValues string
		wantErr    *microerror.Error
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
			name:     "a scheduled update leaves the tag alone",
			chart:    "release-aws",
			tag:      "33.4.0",
			releases: []string{"aws-33.4.0", "aws-34.5.0"},
			flags:    flag{Name: "abcd1", ReleaseVersion: "34.5.0", ScheduledTime: "2022-01-01 01:00"},
			wantTag:  "33.4.0",
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
			name:     "an older release",
			chart:    "release-aws",
			tag:      "34.5.0",
			releases: []string{"aws-33.4.0", "aws-34.5.0"},
			flags:    flag{Name: "abcd1", ReleaseVersion: "33.4.0"},
			wantErr:  notAllowedError,
		},
		{
			name:       "an OCIRepository applied by a Flux Kustomization",
			chart:      "release-aws",
			tag:        "33.4.0",
			repoLabels: map[string]string{fluxKustomizationLabel: "clusters"},
			releases:   []string{"aws-34.5.0"},
			flags:      flag{Name: "abcd1", ReleaseVersion: "34.5.0"},
			wantErr:    notAllowedError,
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
			hrValues: map[string]interface{}{"global": map[string]interface{}{"release": map[string]interface{}{"version": "33.4.0"}}},
			releases: []string{"aws-34.5.0"},
			flags:    flag{Name: "abcd1", ReleaseVersion: "34.5.0"},
			wantErr:  invalidConfigError,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("case %d: %s", i, tc.name), func(t *testing.T) {
			ctx := context.TODO()

			values := tc.userValues
			if values == "" {
				values = userValues
			}
			storage := []runtime.Object{
				newCAPICluster("abcd1", "default", tc.tag),
				newClusterHelmRelease("abcd1", "default", tc.hrValues),
				newClusterOCIRepository("abcd1", "default", tc.chart, tc.tag, tc.repoLabels),
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

			repo := &unstructured.Unstructured{}
			repo.SetGroupVersionKind(ociRepositoryGVK)
			getErr := ctrlClient.Get(ctx, client.ObjectKey{Name: "abcd1", Namespace: "default"}, repo)
			if getErr != nil {
				t.Fatal(getErr)
			}
			tag, _, _ := unstructured.NestedString(repo.Object, "spec", "ref", "tag")

			cm := &corev1.ConfigMap{}
			getErr = ctrlClient.Get(ctx, client.ObjectKey{Name: "abcd1-userconfig", Namespace: "default"}, cm)
			if getErr != nil {
				t.Fatal(getErr)
			}
			wantValues := tc.wantValues
			if wantValues == "" {
				wantValues = values
			}
			if cm.Data["values"] != wantValues {
				t.Fatalf("user values:\n%s\nwant:\n%s", cm.Data["values"], wantValues)
			}

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("want %s, got %v", tc.wantErr.Kind, err)
				}
				if tag != tc.tag {
					t.Fatalf("tag changed on error to %q", tag)
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
