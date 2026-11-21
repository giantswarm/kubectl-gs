package encryption

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

const (
	testMC  = "demomc"
	testOrg = "demoorg"
	testWC  = "demowc"
)

// Test_flag_EncryptionTarget checks the directory each kind of key encrypts
// without --target: an organisation's is namespace/secrets, as the
// organisation's kustomization.yaml applies only what goes into `default`.
func Test_flag_EncryptionTarget(t *testing.T) {
	testCases := []struct {
		name     string
		flag     flag
		expected string
	}{
		{name: "management cluster", flag: flag{ManagementCluster: testMC}, expected: defaultTarget},
		{name: "organization", flag: flag{ManagementCluster: testMC, Organization: testOrg}, expected: "namespace/secrets/"},
		{name: "organization with --target", flag: flag{ManagementCluster: testMC, Organization: testOrg, Target: defaultTarget}, expected: defaultTarget},
		{name: "workload cluster", flag: flag{ManagementCluster: testMC, Organization: testOrg, WorkloadCluster: testWC}, expected: defaultTarget},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.flag.EncryptionTarget()
			if got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

// Test_flag_TargetHelp checks that the help names the default of each kind
// of key, rather than one that is wrong for an organisation.
func Test_flag_TargetHelp(t *testing.T) {
	cmd := &cobra.Command{}
	(&flag{}).Init(cmd)

	usage := cmd.Flags().FlagUsages()
	if strings.Contains(usage, `(default "secrets/")`) {
		t.Errorf("expected no single default for --target, got:\n%s", usage)
	}
	if !strings.Contains(usage, "namespace/secrets/") {
		t.Errorf("expected the organization's default for --target, got:\n%s", usage)
	}
}
