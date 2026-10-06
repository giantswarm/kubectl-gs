package login

import (
	"fmt"
	"os"
	"testing"
)

// TestMain points KUBECONFIG at a scratch file, so a test that loads the
// default kubeconfig instead of its own file never rewrites the developer's
// ~/.kube/config (clientcmd.ModifyConfig drops every context it was not given).
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "kubectl-gs-login-test-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	err = os.Setenv("KUBECONFIG", fmt.Sprintf("%s/config.yaml", dir))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}
