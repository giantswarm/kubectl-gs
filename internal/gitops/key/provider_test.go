package key

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Test_ClusterBaseProviders checks the providers `gitops add base` writes a
// base for, which `gitops check --adopt` recognises too.
func Test_ClusterBaseProviders(t *testing.T) {
	if diff := cmp.Diff([]string{"capa", "capz", "vsphere"}, ClusterBaseProviders()); diff != "" {
		t.Errorf("providers (-want +got):\n%s", diff)
	}
}
