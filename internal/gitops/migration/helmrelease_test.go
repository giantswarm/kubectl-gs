package migration

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/giantswarm/microerror"
	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/metadata"
)

const (
	testBase = "bases/clusters/capa/template"
	testWC   = testOrg + "/workload-clusters/demowc"

	appCR = `apiVersion: application.giantswarm.io/v1alpha1
kind: App
metadata:
  name: ${cluster_name}
spec:
  catalog: cluster
  name: cluster-aws
`
	clusterHelmRelease = `apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: ${cluster_name}
`
	baseKustomization = `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - cluster.yaml
`

	// The workload cluster files as kubectl-gs generated them at structure
	// version 1, for a cluster added with `--skip-mapi`.
	v1ClusterKustomization = `apiVersion: kustomize.config.k8s.io/v1beta1
commonLabels:
  giantswarm.io/managed-by: flux
kind: Kustomization
patches:
  - path: patch_cluster_userconfig.yaml
resources:
  - ../../../../../../../bases/clusters/capa/template
  - cluster_userconfig.yaml
`
	v1ClusterUserConfigPatch = `apiVersion: application.giantswarm.io/v1alpha1
kind: App
metadata:
  name: ${cluster_name}
  namespace: org-${organization}
spec:
  userConfig:
    configMap:
      name: demowc-cluster-userconfig
      namespace: org-demoorg
`
	v1AppsKustomization = `apiVersion: kustomize.config.k8s.io/v1beta1
commonLabels:
  giantswarm.io/cluster: ${cluster_name}
  giantswarm.io/managed-by: flux
kind: Kustomization
namespace: org-${organization}
patches:
  - path: patch_cluster_config.yaml
    target:
      kind: App
resources: []
`
	v1ClusterConfigPatch = `apiVersion: application.giantswarm.io/v1alpha1
kind: App
metadata:
  name: ignored
spec:
  kubeConfig:
    inCluster: false
`

	// The same files as kubectl-gs generates them at structure version 2.
	v2ClusterKustomization = `apiVersion: kustomize.config.k8s.io/v1beta1
commonLabels:
  giantswarm.io/managed-by: flux
kind: Kustomization
patches:
  - path: patch_cluster_userconfig.yaml
    target:
      group: helm.toolkit.fluxcd.io
      kind: HelmRelease
      # A regular expression matching the name the base gives the HelmRelease
      # before Flux substitutes the variables.
      name: \$\{cluster_name\}
resources:
  - ../../../../../../../bases/clusters/capa/template
  - cluster_userconfig.yaml
`
	v2ClusterUserConfigPatch = `- op: add
  path: /spec/valuesFrom/-
  value:
    kind: ConfigMap
    name: demowc-cluster-userconfig
    valuesKey: values
`
	v2AppsKustomization = `apiVersion: kustomize.config.k8s.io/v1beta1
commonLabels:
  giantswarm.io/cluster: ${cluster_name}
  giantswarm.io/managed-by: flux
kind: Kustomization
namespace: org-${organization}
resources: []
`
)

func writeFiles(t *testing.T, fs *afero.Afero, files map[string]string) {
	t.Helper()

	for path, data := range files {
		err := fs.WriteFile(filepath.Join(testRepo, path), []byte(data), 0600)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
	}
}

func Test_helmReleases(t *testing.T) {
	baseLayer := metadata.Layer{Kind: metadata.LayerClusterBase, Path: testBase, StructureVersion: 1}
	wcLayer := metadata.Layer{Kind: metadata.LayerWorkloadCluster, Path: testWC, StructureVersion: 1}
	appLayer := metadata.Layer{Kind: metadata.LayerApp, Path: testWC + "/apps/hello", StructureVersion: 1}

	migratedBase := map[string]string{
		testBase + "/kustomization.yaml": baseKustomization,
		testBase + "/cluster.yaml":       clusterHelmRelease,
	}
	v1WC := map[string]string{
		testWC + "/cluster/kustomization.yaml":            v1ClusterKustomization,
		testWC + "/cluster/patch_cluster_userconfig.yaml": v1ClusterUserConfigPatch,
		testWC + "/cluster/cluster_userconfig.yaml":       "apiVersion: v1\nkind: ConfigMap\n",
		testWC + "/apps/kustomization.yaml":               v1AppsKustomization,
		testWC + "/apps/patch_cluster_config.yaml":        v1ClusterConfigPatch,
	}

	testCases := []struct {
		name           string
		files          []map[string]string
		layer          metadata.Layer
		expectErr      *microerror.Error
		expectErrMsg   string
		expectFiles    map[string]string
		expectRemoved  []string
		expectNChanges int
	}{
		{
			name:         "cluster base with an App CR",
			files:        []map[string]string{{testBase + "/kustomization.yaml": baseKustomization, testBase + "/cluster.yaml": appCR}},
			layer:        baseLayer,
			expectErr:    appCRError,
			expectErrMsg: testBase + "/cluster.yaml",
		},
		{
			name:  "cluster base with a HelmRelease",
			files: []map[string]string{migratedBase},
			layer: baseLayer,
		},
		{
			name:         "app with an App CR",
			files:        []map[string]string{{testWC + "/apps/hello/appcr.yaml": appCR}},
			layer:        appLayer,
			expectErr:    appCRError,
			expectErrMsg: "app-platform-deprecation",
		},
		{
			name:  "app with a HelmRelease",
			files: []map[string]string{{testWC + "/apps/hello/helm-release.yaml": clusterHelmRelease}},
			layer: appLayer,
		},
		{
			name:  "workload cluster",
			files: []map[string]string{migratedBase, v1WC},
			layer: wcLayer,
			expectFiles: map[string]string{
				testWC + "/cluster/kustomization.yaml":            v2ClusterKustomization,
				testWC + "/cluster/patch_cluster_userconfig.yaml": v2ClusterUserConfigPatch,
				testWC + "/apps/kustomization.yaml":               v2AppsKustomization,
			},
			expectRemoved:  []string{testWC + "/apps/patch_cluster_config.yaml"},
			expectNChanges: 3,
		},
		{
			name: "workload cluster with a user config patch targeting the App",
			files: []map[string]string{migratedBase, v1WC, {
				testWC + "/cluster/kustomization.yaml": strings.Replace(v1ClusterKustomization, "  - path: patch_cluster_userconfig.yaml\n", "  - path: patch_cluster_userconfig.yaml\n    target:\n      kind: App\n", 1),
			}},
			layer: wcLayer,
			expectFiles: map[string]string{
				testWC + "/cluster/kustomization.yaml":            v2ClusterKustomization,
				testWC + "/cluster/patch_cluster_userconfig.yaml": v2ClusterUserConfigPatch,
			},
			expectNChanges: 3,
		},
		{
			name: "workload cluster with a user config Secret in the patch",
			files: []map[string]string{migratedBase, v1WC, {
				testWC + "/cluster/patch_cluster_userconfig.yaml": v1ClusterUserConfigPatch + "    secret:\n      name: demowc-cluster-usersecret\n      namespace: org-demoorg\n",
			}},
			layer:        wcLayer,
			expectErr:    unexpectedFileError,
			expectErrMsg: "by hand",
		},
		{
			name: "workload cluster already migrated",
			files: []map[string]string{migratedBase, {
				testWC + "/cluster/kustomization.yaml":            v2ClusterKustomization,
				testWC + "/cluster/patch_cluster_userconfig.yaml": v2ClusterUserConfigPatch,
				testWC + "/apps/kustomization.yaml":               v2AppsKustomization,
			}},
			layer: wcLayer,
			expectFiles: map[string]string{
				testWC + "/cluster/kustomization.yaml": v2ClusterKustomization,
				testWC + "/apps/kustomization.yaml":    v2AppsKustomization,
			},
		},
		{
			name: "workload cluster with mapi",
			files: []map[string]string{migratedBase, {
				testWC + "/mapi/apps/kustomization.yaml":        v1AppsKustomization,
				testWC + "/mapi/apps/patch_cluster_config.yaml": v1ClusterConfigPatch,
			}},
			layer:          wcLayer,
			expectFiles:    map[string]string{testWC + "/mapi/apps/kustomization.yaml": v2AppsKustomization},
			expectRemoved:  []string{testWC + "/mapi/apps/patch_cluster_config.yaml"},
			expectNChanges: 2,
		},
		{
			name: "workload cluster on a cluster base with an App CR",
			files: []map[string]string{v1WC, {
				testBase + "/kustomization.yaml": baseKustomization,
				testBase + "/cluster.yaml":       appCR,
			}},
			layer:        wcLayer,
			expectErr:    appCRError,
			expectErrMsg: "release-<provider> HelmRelease",
		},
		{
			name: "workload cluster with an App",
			files: []map[string]string{migratedBase, v1WC, {
				testWC + "/apps/kustomization.yaml":        strings.Replace(v1AppsKustomization, "resources: []", "resources:\n  - hello/appcr.yaml", 1),
				testWC + "/apps/hello/appcr.yaml":          appCR,
				testWC + "/apps/hello/configmap.yaml":      "apiVersion: v1\nkind: ConfigMap\n",
				testWC + "/apps/patch_cluster_config.yaml": v1ClusterConfigPatch,
			}},
			layer:        wcLayer,
			expectErr:    appCRError,
			expectErrMsg: testWC + "/apps/hello/appcr.yaml",
		},
		{
			name: "workload cluster with an edited user config patch",
			files: []map[string]string{migratedBase, v1WC, {
				testWC + "/cluster/patch_cluster_userconfig.yaml": "apiVersion: application.giantswarm.io/v1alpha1\nkind: App\nmetadata:\n  name: ${cluster_name}\nspec:\n  version: 1.0.0\n",
			}},
			layer:        wcLayer,
			expectErr:    unexpectedFileError,
			expectErrMsg: "by hand",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs := &afero.Afero{Fs: afero.NewMemMapFs()}
			for _, files := range tc.files {
				writeFiles(t, fs, files)
			}

			changes, err := helmReleases.Apply(fs, testRepo, tc.layer)
			if tc.expectErr != nil {
				if microerror.Cause(err) != tc.expectErr {
					t.Fatalf("expected %s, got: %v", tc.expectErr.Kind, err)
				}
				if !strings.Contains(err.Error(), tc.expectErrMsg) {
					t.Fatalf("expected error %q to contain %q", err.Error(), tc.expectErrMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if len(changes) != tc.expectNChanges {
				t.Errorf("expected %d changes, got: %q", tc.expectNChanges, changes)
			}
			for path, expected := range tc.expectFiles {
				got, err := fs.ReadFile(filepath.Join(testRepo, path))
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
				if diff := cmp.Diff(expected, string(got)); diff != "" {
					t.Errorf("unexpected %s (-want +got):\n%s", path, diff)
				}
			}
			for _, path := range tc.expectRemoved {
				if exists(t, fs, filepath.Join(testRepo, path)) {
					t.Errorf("expected %s to be removed", path)
				}
			}

			// Running it again changes nothing.
			changes, err = helmReleases.Apply(fs, testRepo, tc.layer)
			if err != nil {
				t.Fatalf("unexpected error on the second run: %s", err)
			}
			if len(changes) != 0 {
				t.Errorf("expected no changes on the second run, got: %q", changes)
			}
		})
	}
}
