package creator

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"
)

const (
	dryRunRepoPath = "/tmp/testpath"
)

func Test_Create(t *testing.T) {
	testCases := []struct {
		name           string
		creator        Creator
		expectedCreate map[string]string
		expectedDryRun string
	}{
		{
			name: "flawless management cluster (dry-run)",
			creator: Creator{
				dryRun: true,
				fsObjects: []*FsObject{
					NewFsObject("demomc", nil),
					NewFsObject(
						"demomc/demomc.yaml",
						[]byte(`apiVersion: kustomize.toolkit.fluxcd.io/v1beta2
kind: Kustomization
metadata:
  name: demomc-gitops
  namespace: default
spec:
  interval: 1m
  path: ./management-clusters/demomc
  prune: false
  serviceAccountName: automation
  sourceRef:
    kind: GitRepository
    name: gitops-demo
  timeout: 1m
`),
					),
					NewFsObject("demomc/.sops.keys", nil),
					NewFsObject("demomc/secrets", nil),
					NewFsObject("demomc/organizations", nil),
				},
				path: dryRunRepoPath,
			},
			expectedDryRun: "testdata/expected/files/dry-run/case-0-flawless.golden",
		},
		{
			name: "flawless organization (dry-run)",
			creator: Creator{
				dryRun: true,
				fsObjects: []*FsObject{
					NewFsObject("demoorg", nil),
					NewFsObject(
						"demoorg/demoorg.yaml",
						[]byte(`apiVersion: security.giantswarm.io/v1alpha1
kind: Organization
metadata:
  name: demoorg
spec: {}
`),
					),
					NewFsObject("demoorg/workload-clusters", nil),
					NewFsObject(
						"demoorg/workload-clusters/kustomization.yaml",
						[]byte(`apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources: []`),
					),
				},
				path: dryRunRepoPath,
			},
			expectedDryRun: "testdata/expected/files/dry-run/case-1-flawless.golden",
		},
		{
			name: "flawless workload cluster (dry-run)",
			creator: Creator{
				dryRun: true,
				fsObjects: []*FsObject{
					NewFsObject("workload-clusters", nil),
					NewFsObject(
						"workload-clusters/demowc.yaml",
						[]byte(`apiVersion: kustomize.toolkit.fluxcd.io/v1beta2
kind: Kustomization
metadata:
  name: demomc-clusters-demowc
  namespace: default
spec:
  interval: 1m
  path: "./management-clusters/demomc/organizations/demoorg/workload-clusters/demowc"
  postBuild:
    substitute:
      cluster_name: "demowc"
      organization: "demoorg"
      cluster_release: "0.13.0"
      default_apps_release: "0.6.0"
  prune: false
  serviceAccountName: automation
  sourceRef:
    kind: GitRepository
    name: gitops-demo
  timeout: 2m
`),
					),
					NewFsObject("workload-clusters/demowc", nil),
					NewFsObject("workload-clusters/demowc/apps", nil),
					NewFsObject("workload-clusters/demowc/cluster", nil),
					NewFsObject(
						"workload-clusters/demowc/cluster/kustomization.yaml",
						[]byte(`apiVersion: kustomize.config.k8s.io/v1beta1
commonLabels:
  giantswarm.io/managed-by: flux
kind: Kustomization
resources:
  - ../../../../../../../bases/cluster/capo`),
					),
				},
				path: dryRunRepoPath,
			},
			expectedDryRun: "testdata/expected/files/dry-run/case-2-flawless.golden",
		},
		{
			name: "flawless management cluster (create)",
			creator: Creator{
				dryRun: false,
				fsObjects: []*FsObject{
					NewFsObject("demomc", nil),
					NewFsObject(
						"demomc/demomc.yaml",
						[]byte(`apiVersion: kustomize.toolkit.fluxcd.io/v1beta2
kind: Kustomization
metadata:
  name: demomc-gitops
  namespace: default
spec:
  interval: 1m
  path: ./management-clusters/demomc
  prune: false
  serviceAccountName: automation
  sourceRef:
    kind: GitRepository
    name: gitops-demo
  timeout: 1m
`),
					),
					NewFsObject("demomc/.sops.keys", nil),
					NewFsObject("demomc/secrets", nil),
					NewFsObject("demomc/organizations", nil),
				},
				fs: &afero.Afero{Fs: afero.NewOsFs()},
			},
			expectedCreate: map[string]string{
				"demomc":               "",
				"demomc/demomc.yaml":   "testdata/expected/files/create/0-demomc.golden",
				"demomc/.sops.keys":    "",
				"demomc/secrets":       "",
				"demomc/organizations": "",
			},
		},
		{
			name: "flawless organization (create)",
			creator: Creator{
				dryRun: false,
				fsObjects: []*FsObject{
					NewFsObject("demoorg", nil),
					NewFsObject(
						"demoorg/demoorg.yaml",
						[]byte(`apiVersion: security.giantswarm.io/v1alpha1
kind: Organization
metadata:
  name: demoorg
spec: {}
`),
					),
					NewFsObject("demoorg/workload-clusters", nil),
					NewFsObject(
						"demoorg/workload-clusters/kustomization.yaml",
						[]byte(`apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources: []
`),
					),
				},
				fs: &afero.Afero{Fs: afero.NewOsFs()},
			},
			expectedCreate: map[string]string{
				"demoorg":                   "",
				"demoorg/demoorg.yaml":      "testdata/expected/files/create/1-demoorg.golden",
				"demoorg/workload-clusters": "",
				"demoorg/workload-clusters/kustomization.yaml": "testdata/expected/files/create/1-kustomization.golden",
			},
		},
		{
			name: "flawless workload cluster (create)",
			creator: Creator{
				dryRun: false,
				fsObjects: []*FsObject{
					NewFsObject("workload-clusters", nil),
					NewFsObject(
						"workload-clusters/demowc.yaml",
						[]byte(`apiVersion: kustomize.toolkit.fluxcd.io/v1beta2
kind: Kustomization
metadata:
  name: demomc-clusters-demowc
  namespace: default
spec:
  interval: 1m
  path: "./management-clusters/demomc/organizations/demoorg/workload-clusters/demowc"
  postBuild:
    substitute:
      cluster_name: "demomc"
      organization: "demoorg"
      cluster_release: "0.13.0"
      default_apps_release: "0.6.0"
  prune: false
  serviceAccountName: automation
  sourceRef:
    kind: GitRepository
    name: gitops-demo
  timeout: 2m
`),
					),
					NewFsObject("workload-clusters/demowc", nil),
					NewFsObject("workload-clusters/demowc/apps", nil),
					NewFsObject("workload-clusters/demowc/cluster", nil),
					NewFsObject(
						"workload-clusters/demowc/cluster/kustomization.yaml",
						[]byte(`apiVersion: kustomize.config.k8s.io/v1beta1
commonLabels:
  giantswarm.io/managed-by: flux
kind: Kustomization
resources:
  - ../../../../../../../bases/cluster/capo
`),
					),
				},
				fs: &afero.Afero{Fs: afero.NewOsFs()},
			},
			expectedCreate: map[string]string{
				"workload-clusters":                                   "",
				"workload-clusters/demowc.yaml":                       "testdata/expected/files/create/2-demowc.golden",
				"workload-clusters/demowc":                            "",
				"workload-clusters/demowc/apps":                       "",
				"workload-clusters/demowc/cluster":                    "",
				"workload-clusters/demowc/cluster/kustomization.yaml": "testdata/expected/files/create/2-kustomization.golden",
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("case %d: %s", i, tc.name), func(t *testing.T) {
			var err error

			var tmpDir string

			if !tc.creator.dryRun {
				tmpDir, err = ioutil.TempDir("", "creator-test")

				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}

				err = os.Mkdir(tmpDir+"/management-clusters", 0755)
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}

				tc.creator.path = fmt.Sprintf("%s/%s", tmpDir, tc.creator.path)
				defer os.RemoveAll(tmpDir)
			}

			out := new(bytes.Buffer)
			tc.creator.stdout = out

			err = tc.creator.Create()
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			if tc.expectedDryRun != "" {
				expected, err := ioutil.ReadFile(tc.expectedDryRun)
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}

				if !bytes.Equal(out.Bytes(), expected) {
					t.Fatalf("want matching files \n%s\n", cmp.Diff(string(expected), out.String()))
				}
			}

			if tc.expectedCreate == nil {
				return
			}

			for p, c := range tc.expectedCreate {
				if _, err := os.Stat(fmt.Sprintf("%s/%s", tc.creator.path, p)); err != nil {
					t.Fatalf("expected ni, got error: %s", err.Error())
				}

				if c == "" {
					continue
				}

				got, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", tc.creator.path, p))
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}

				expected, err := ioutil.ReadFile(c)
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}

				if !bytes.Equal(got, expected) {
					t.Fatalf("want matching files \n%s\n", cmp.Diff(string(expected), string(got)))
				}
			}

		})
	}
}
