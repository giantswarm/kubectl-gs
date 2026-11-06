package wcluster

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// fakeOCIClient is a minimal ociregistry.Client fake. tags holds the
// "repository:tag" keys that exist.
type fakeOCIClient struct {
	tags map[string]bool
}

func (f *fakeOCIClient) ListTags(_ context.Context, _, _ string) ([]string, error) { return nil, nil }

func (f *fakeOCIClient) TagExists(_ context.Context, _, repository, tag string) (bool, error) {
	return f.tags[repository+":"+tag], nil
}

func (f *fakeOCIClient) GetManifestAnnotations(_ context.Context, _, _, _ string) (map[string]string, error) {
	return nil, nil
}

func (f *fakeOCIClient) Close(_ context.Context) {}

func Test_runner_run(t *testing.T) {
	const (
		clusterDir     = "/repo/management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/cluster"
		userConfigFile = clusterDir + "/cluster_userconfig.yaml"
	)

	testCases := []struct {
		name               string
		base               string
		release            string
		clusterUserConfig  string
		expectInvalidFlags string
		expectInvalid      string
		expectStderr       string
		expectUserConfig   bool
	}{
		{
			name:               "missing base",
			base:               "bases/nope",
			expectInvalidFlags: `--base "bases/nope" not found under "/repo"`,
		},
		{
			name:         "overlay base",
			base:         "bases/overlay",
			expectStderr: "is not checked against the gsoci.azurecr.io registry",
		},
		{
			name:              "overlay base with user config",
			base:              "bases/overlay",
			clusterUserConfig: "/values.yaml",
			expectStderr:      "is not checked against the gsoci.azurecr.io registry",
			expectUserConfig:  true,
		},
		{
			name:              "release chart base with user config",
			base:              "bases/capa",
			clusterUserConfig: "/values.yaml",
			expectUserConfig:  true,
		},
		{
			name:    "release with v prefix",
			base:    "bases/capa",
			release: "v36.1.0",
		},
		{
			name:               "release not published",
			base:               "bases/capa",
			release:            "36.0.0-test",
			expectInvalidFlags: "--release 36.0.0-test is not published as a release-aws chart",
		},
		{
			name:               "vSphere release before the first HelmRelease one",
			base:               "bases/vsphere",
			release:            "36.0.0",
			expectInvalidFlags: "the release-vsphere chart needs at least 36.1.0",
		},
		{
			name:          "cluster App base",
			base:          "bases/app",
			expectInvalid: "kubectl-gs no longer supports",
		},
		{
			name:          "release overlay of a cluster App base",
			base:          "bases/release",
			expectInvalid: "kubectl-gs no longer supports",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			seed := map[string]string{
				"/repo/bases/app/cluster.yaml":           app("cluster-aws"),
				"/repo/bases/capa/cluster.yaml":          ociRepository("release-aws"),
				"/repo/bases/vsphere/cluster.yaml":       ociRepository("release-vsphere"),
				"/repo/bases/overlay/kustomization.yaml": "resources:\n  - ../capa\n",
				"/repo/bases/release/kustomization.yaml": strings.Replace(releaseOverlay, "../template", "../app", 1),
				"/repo/management-clusters/demomc/organizations/demoorg/workload-clusters/kustomization.yaml": "apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\nresources: []\n",
				"/values.yaml": "testKey: testValue\n",
			}
			for path, data := range seed {
				err := afero.WriteFile(fs, path, []byte(data), 0600)
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}
			}

			release := tc.release
			if release == "" {
				release = "36.1.0"
			}

			var stderr bytes.Buffer
			r := &runner{
				flag: &flag{
					Base:              tc.base,
					ClusterUserConfig: tc.clusterUserConfig,
					ManagementCluster: "demomc",
					Name:              "demowc",
					Organization:      "demoorg",
					Release:           release,
					RepositoryName:    "gitops-demo",
					SkipMAPI:          true,
				},
				fs: fs,
				ociClient: &fakeOCIClient{tags: map[string]bool{
					"charts/giantswarm/release-aws:36.1.0":     true,
					"charts/giantswarm/release-vsphere:36.0.0": true,
				}},
				stderr: &stderr,
				stdout: &bytes.Buffer{},
			}

			parent := &cobra.Command{}
			parent.PersistentFlags().String("local-path", "/repo", "")
			cmd := &cobra.Command{}
			parent.AddCommand(cmd)

			err := r.run(context.Background(), cmd, nil)
			if tc.expectInvalidFlags != "" || tc.expectInvalid != "" {
				expectedErr, expectedMsg := invalidFlagsError, tc.expectInvalidFlags
				if tc.expectInvalid != "" {
					expectedErr, expectedMsg = invalidConfigError, tc.expectInvalid
				}
				if microerror.Cause(err) != expectedErr {
					t.Fatalf("expected %s, got: %v", expectedErr.Kind, err)
				}
				if !strings.Contains(err.Error(), expectedMsg) {
					t.Fatalf("expected error %q to contain %q", err.Error(), expectedMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			if tc.expectStderr == "" && stderr.Len() > 0 {
				t.Errorf("expected no output on stderr, got %q", stderr.String())
			}
			if !strings.Contains(stderr.String(), tc.expectStderr) {
				t.Errorf("expected stderr %q to contain %q", stderr.String(), tc.expectStderr)
			}

			userConfig, err := afero.ReadFile(fs, userConfigFile)
			if !tc.expectUserConfig {
				if err == nil {
					t.Fatalf("expected no %s, got:\n%s", userConfigFile, userConfig)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}
			if !strings.Contains(string(userConfig), "testKey: testValue") {
				t.Errorf("expected the user config, got:\n%s", userConfig)
			}
			if strings.Contains(string(userConfig), "release") {
				t.Errorf("expected the user config without a release, got:\n%s", userConfig)
			}
		})
	}
}
