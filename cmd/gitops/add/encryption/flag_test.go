package encryption

import "testing"

const (
	defaultTarget = "secrets/"
	testMC        = "demomc"
	testOrg       = "demoorg"
	testWC        = "demowc"
)

// Test_flag_EncryptionTarget checks that an organisation's key encrypts
// namespace/secrets unless --target says otherwise: the organisation's
// kustomization.yaml applies only what goes into `default`.
func Test_flag_EncryptionTarget(t *testing.T) {
	testCases := []struct {
		name     string
		flag     flag
		changed  bool
		expected string
	}{
		{name: "management cluster", flag: flag{ManagementCluster: testMC, Target: defaultTarget}, expected: defaultTarget},
		{name: "organization", flag: flag{ManagementCluster: testMC, Organization: testOrg, Target: defaultTarget}, expected: "namespace/secrets/"},
		{name: "organization with --target", flag: flag{ManagementCluster: testMC, Organization: testOrg, Target: defaultTarget}, changed: true, expected: defaultTarget},
		{name: "workload cluster", flag: flag{ManagementCluster: testMC, Organization: testOrg, WorkloadCluster: testWC, Target: defaultTarget}, expected: defaultTarget},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.flag.EncryptionTarget(tc.changed)
			if got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}
