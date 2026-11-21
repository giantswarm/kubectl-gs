package common

import (
	"errors"
	"strings"
	"testing"

	"github.com/spf13/afero"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/creator"
)

const testNamespaceFile = "/repo/management-clusters/demomc/organizations/demoorg/namespace.yaml"

// Test_RequireMovedClusters checks that a workload cluster is refused in an
// organisation whose clusters' Kustomizations haven't moved into its
// namespace yet, as `gitops upgrade` leaves them.
func Test_RequireMovedClusters(t *testing.T) {
	testCases := []struct {
		name      string
		resources string
		expectErr string
	}{
		{name: "moved", resources: "resources:\n  - secrets\n  - ../workload-clusters\n"},
		{name: "not moved yet", resources: "resources: []\n", expectErr: "upgrade-fork.md#move-the-workload-cluster-kustomizations"},
		{name: "no namespace directory", expectErr: "kubectl gs gitops upgrade"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs := &afero.Afero{Fs: afero.NewMemMapFs()}
			_ = fs.WriteFile(testNamespaceFile, []byte("kind: Kustomization\n"), 0600)
			if tc.resources != "" {
				_ = fs.WriteFile(strings.TrimSuffix(testNamespaceFile, ".yaml")+"/kustomization.yaml", []byte(tc.resources), 0600)
			}

			err := RequireMovedClusters(fs, testNamespaceFile)
			if tc.expectErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
				return
			}
			if !errors.Is(err, creator.ValidationError) || !strings.Contains(err.Error(), tc.expectErr) {
				t.Fatalf("expected a validation error with %q, got: %v", tc.expectErr, err)
			}
		})
	}
}
