package wcluster

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

const (
	layoutBase       = "bases/capa"
	layoutMC         = "demomc"
	layoutOrg        = "demoorg"
	layoutRelease    = "36.1.0"
	layoutRepository = "gitops-demo"
	layoutWC         = "demowc"

	layoutWCsDir = "/repo/management-clusters/demomc/organizations/demoorg/workload-clusters"
	layoutWCDir  = layoutWCsDir + "/demowc"

	layoutSingleKustomization = `apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: demomc-clusters-demowc
  namespace: default
spec:
  path: "./management-clusters/demomc/organizations/demoorg/workload-clusters/demowc"
  postBuild:
    substitute:
      cluster_name: demowc
      organization: demoorg
`
)

type layoutCase struct {
	name string
	seed map[string]string

	expectKustomizations []string
	expectApplied        []string
	expectClusterPath    string
	expectAppsFile       bool
}

// layoutCases add a cluster with a base, and add a base to a cluster that
// already has a single Kustomization.
var layoutCases = []layoutCase{
	{
		name: "new cluster",
		seed: map[string]string{
			layoutWCsDir + "/kustomization.yaml": "resources: []\n",
		},
		expectKustomizations: []string{"demowc.yaml", "demowc-apps.yaml"},
		expectApplied:        []string{"apps/"},
		expectClusterPath:    "./management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/cluster",
		expectAppsFile:       true,
	},
	{
		name: "cluster with a single Kustomization",
		seed: map[string]string{
			layoutWCsDir + "/kustomization.yaml":     "resources:\n  - demowc.yaml\n",
			layoutWCsDir + "/demowc.yaml":            layoutSingleKustomization,
			layoutWCDir + "/kustomization.yaml":      "resources:\n  - apps/\n",
			layoutWCDir + "/apps/kustomization.yaml": "resources: []\n",
		},
		expectKustomizations: []string{"demowc.yaml"},
		expectApplied:        []string{"apps/", "cluster/"},
		expectClusterPath:    "./management-clusters/demomc/organizations/demoorg/workload-clusters/demowc",
	},
}

// Test_runner_run_layout checks which Kustomizations apply a cluster added
// with a base.
func Test_runner_run_layout(t *testing.T) {
	for _, tc := range layoutCases {
		t.Run(tc.name, func(t *testing.T) {
			fs := seedLayout(t, tc.seed)

			err := runLayout(fs)
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			checkLayout(t, fs, tc)
		})
	}
}

func seedLayout(t *testing.T, seed map[string]string) afero.Fs {
	t.Helper()

	fs := afero.NewMemMapFs()
	seed["/repo/bases/capa/cluster.yaml"] = ociRepository("release-aws")
	for path, data := range seed {
		err := afero.WriteFile(fs, path, []byte(data), 0600)
		if err != nil {
			t.Fatalf("unexpected error: %s", err.Error())
		}
	}

	return fs
}

func runLayout(fs afero.Fs) error {
	r := &runner{
		flag: &flag{
			Base:              layoutBase,
			ManagementCluster: layoutMC,
			Name:              layoutWC,
			Organization:      layoutOrg,
			Release:           layoutRelease,
			RepositoryName:    layoutRepository,
		},
		fs:        fs,
		ociClient: &fakeOCIClient{tags: map[string]bool{"charts/giantswarm/release-aws:" + layoutRelease: true}},
		stderr:    &bytes.Buffer{},
		stdout:    &bytes.Buffer{},
	}

	parent := &cobra.Command{}
	parent.PersistentFlags().String("local-path", "/repo", "")
	cmd := &cobra.Command{}
	parent.AddCommand(cmd)

	return r.run(context.Background(), cmd, nil)
}

func checkLayout(t *testing.T, fs afero.Fs, tc layoutCase) {
	t.Helper()

	if diff := cmp.Diff(tc.expectKustomizations, resources(t, fs, layoutWCsDir+"/kustomization.yaml")); diff != "" {
		t.Errorf("workload-clusters/kustomization.yaml resources (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(tc.expectApplied, resources(t, fs, layoutWCDir+"/kustomization.yaml")); diff != "" {
		t.Errorf("demowc/kustomization.yaml resources (-want +got):\n%s", diff)
	}

	var kustomization struct {
		Spec struct {
			Path string `json:"path"`
		} `json:"spec"`
	}
	readYAML(t, fs, layoutWCsDir+"/demowc.yaml", &kustomization)
	if kustomization.Spec.Path != tc.expectClusterPath {
		t.Errorf("expected demowc.yaml to apply %q, got %q", tc.expectClusterPath, kustomization.Spec.Path)
	}

	if exists(t, fs, layoutWCsDir+"/demowc-apps.yaml") != tc.expectAppsFile {
		t.Errorf("expected demowc-apps.yaml to exist: %t", tc.expectAppsFile)
	}
	if !exists(t, fs, layoutWCDir+"/cluster/kustomization.yaml") {
		t.Errorf("expected the cluster definition in cluster/")
	}
}

func exists(t *testing.T, fs afero.Fs, path string) bool {
	t.Helper()

	ok, err := afero.Exists(fs, path)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	return ok
}

func resources(t *testing.T, fs afero.Fs, path string) []string {
	t.Helper()

	var kustomization struct {
		Resources []string `json:"resources"`
	}
	readYAML(t, fs, path, &kustomization)

	return kustomization.Resources
}

func readYAML(t *testing.T, fs afero.Fs, path string, into any) {
	t.Helper()

	data, err := afero.ReadFile(fs, path)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}
	err = yaml.Unmarshal(data, into)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}
}
