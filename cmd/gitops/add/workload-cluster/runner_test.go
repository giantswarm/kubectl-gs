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

func Test_runner_run(t *testing.T) {
	const (
		clusterDir     = "/repo/management-clusters/demomc/organizations/demoorg/workload-clusters/demowc/cluster"
		userConfigFile = clusterDir + "/cluster_userconfig.yaml"
	)

	testCases := []struct {
		name              string
		base              string
		clusterUserConfig string
		expectInvalid     string
		expectStderr      string
		expectUserConfig  []string
	}{
		{
			name:          "missing base",
			base:          "bases/nope",
			expectInvalid: `--base "bases/nope" not found under "/repo"`,
		},
		{
			name:         "overlay base",
			base:         "bases/overlay",
			expectStderr: "Set the release in the base's values.",
		},
		{
			name:              "overlay base with user config",
			base:              "bases/overlay",
			clusterUserConfig: "/values.yaml",
			expectStderr:      "they must set global.release.version themselves",
			expectUserConfig:  []string{"testKey: testValue"},
		},
		{
			name:              "cluster chart base with user config",
			base:              "bases/capa",
			clusterUserConfig: "/values.yaml",
			expectUserConfig:  []string{"testKey: testValue", "version: 36.1.0"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			seed := map[string]string{
				"/repo/bases/capa/cluster.yaml":          app("cluster-aws"),
				"/repo/bases/overlay/kustomization.yaml": "resources:\n  - ../capa\n",
				"/repo/management-clusters/demomc/organizations/demoorg/workload-clusters/kustomization.yaml": "apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\nresources: []\n",
				"/values.yaml": "testKey: testValue\n",
			}
			for path, data := range seed {
				err := afero.WriteFile(fs, path, []byte(data), 0600)
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}
			}

			var stderr bytes.Buffer
			r := &runner{
				flag: &flag{
					Base:              tc.base,
					ClusterUserConfig: tc.clusterUserConfig,
					ManagementCluster: "demomc",
					Name:              "demowc",
					Organization:      "demoorg",
					Release:           "36.1.0",
					RepositoryName:    "gitops-demo",
					SkipMAPI:          true,
				},
				fs:     fs,
				stderr: &stderr,
				stdout: &bytes.Buffer{},
			}

			parent := &cobra.Command{}
			parent.PersistentFlags().String("local-path", "/repo", "")
			cmd := &cobra.Command{}
			parent.AddCommand(cmd)

			err := r.run(context.Background(), cmd, nil)
			if tc.expectInvalid != "" {
				if microerror.Cause(err) != invalidFlagsError {
					t.Fatalf("expected invalid flags error, got: %v", err)
				}
				if !strings.Contains(err.Error(), tc.expectInvalid) {
					t.Fatalf("expected error %q to contain %q", err.Error(), tc.expectInvalid)
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
			if len(tc.expectUserConfig) == 0 {
				if err == nil {
					t.Fatalf("expected no %s, got:\n%s", userConfigFile, userConfig)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}
			for _, s := range tc.expectUserConfig {
				if !strings.Contains(string(userConfig), s) {
					t.Errorf("expected %q in the user config, got:\n%s", s, userConfig)
				}
			}
			if len(tc.expectUserConfig) == 1 && strings.Contains(string(userConfig), "release") {
				t.Errorf("expected the user config without a release, got:\n%s", userConfig)
			}
		})
	}
}
