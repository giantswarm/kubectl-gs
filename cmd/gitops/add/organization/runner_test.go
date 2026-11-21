package org

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

const (
	runnerMCFile        = "/repo/management-clusters/demomc/demomc.yaml"
	runnerNamespaceFile = "/repo/management-clusters/demomc/organizations/demoorg/namespace.yaml"
	runnerMC            = `apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: demomc-gitops
  namespace: default
spec:
  sourceRef:
    kind: GitRepository
    name: mc-repo
`
)

// Test_runner_RepositoryName checks the GitRepository the organisation's
// namespace Kustomization reads: --repository-name, or else the management
// cluster's.
func Test_runner_RepositoryName(t *testing.T) {
	testCases := []struct {
		name          string
		flag          string
		mc            string
		expected      string
		expectInvalid bool
	}{
		{name: "from the management cluster", mc: runnerMC, expected: "name: mc-repo"},
		{name: "from the flag", flag: "flag-repo", mc: runnerMC, expected: "name: flag-repo"},
		{name: "neither", expectInvalid: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			if tc.mc != "" {
				err := afero.WriteFile(fs, runnerMCFile, []byte(tc.mc), 0600)
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
			}

			err := runOrganization(fs, tc.flag)
			if tc.expectInvalid {
				if !errors.Is(err, invalidFlagsError) {
					t.Fatalf("expected invalid flags error, got: %v", err)
				}
				return
			}

			checkNamespaceFile(t, fs, err, tc.expected)
		})
	}
}

func runOrganization(fs afero.Fs, repository string) error {
	r := &runner{
		flag:   &flag{ManagementCluster: testMC, Name: testOrg, RepositoryName: repository},
		fs:     fs,
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
	}
	parent := &cobra.Command{}
	parent.PersistentFlags().String("local-path", "/repo", "")
	cmd := &cobra.Command{}
	parent.AddCommand(cmd)

	return r.run(context.Background(), cmd, nil)
}

func checkNamespaceFile(t *testing.T, fs afero.Fs, err error, expected string) {
	t.Helper()

	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	data, err := afero.ReadFile(fs, runnerNamespaceFile)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !strings.Contains(string(data), expected) {
		t.Errorf("expected %q in namespace.yaml, got:\n%s", expected, data)
	}
}
