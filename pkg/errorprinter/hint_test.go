package errorprinter

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"syscall"
	"testing"

	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/clientcmd"
)

// connectionRefused builds the error chain client-go produces when it falls
// back to localhost:8080 because no kubeconfig is available.
func connectionRefused() error {
	return &url.Error{
		Op:  "Get",
		URL: "http://localhost:8080/api",
		Err: &net.OpError{
			Op:  "dial",
			Net: "tcp",
			Err: os.NewSyscallError("connect", syscall.ECONNREFUSED),
		},
	}
}

func TestHint(t *testing.T) {
	testCases := []struct {
		name        string
		err         error
		expectsHint bool
	}{
		{
			name:        "case 0: no error",
			err:         nil,
			expectsHint: false,
		},
		{
			name:        "case 1: empty kubeconfig",
			err:         clientcmd.ErrEmptyConfig,
			expectsHint: true,
		},
		{
			name:        "case 2: no context chosen",
			err:         clientcmd.ErrNoContext,
			expectsHint: true,
		},
		{
			name:        "case 3: connection refused on the localhost fallback",
			err:         connectionRefused(),
			expectsHint: true,
		},
		{
			name:        "case 4: connection refused, wrapped by microerror",
			err:         microerror.Mask(fmt.Errorf("failed to get server groups: %w", connectionRefused())),
			expectsHint: true,
		},
		{
			name: "case 5: Giant Swarm CRD unknown to the API server",
			err: &meta.NoKindMatchError{
				GroupKind:        schema.GroupKind{Group: "application.giantswarm.io", Kind: "AppCatalogEntry"},
				SearchedVersions: []string{"v1alpha1"},
			},
			expectsHint: true,
		},
		{
			name: "case 6: Giant Swarm CRD unknown, wrapped by microerror",
			err: microerror.Mask(&meta.NoKindMatchError{
				GroupKind:        schema.GroupKind{Group: "application.giantswarm.io", Kind: "AppCatalogEntry"},
				SearchedVersions: []string{"v1alpha1"},
			}),
			expectsHint: true,
		},
		{
			name:        "case 7: expired token",
			err:         apierrors.NewUnauthorized("Unauthorized"),
			expectsHint: true,
		},
		{
			name:        "case 8: generic error",
			err:         errors.New("something went wrong"),
			expectsHint: false,
		},
		{
			name:        "case 9: invalid flag",
			err:         microerror.Maskf(&microerror.Error{Kind: "invalidFlagError"}, "--name must not be empty"),
			expectsHint: false,
		},
		{
			name:        "case 10: resource not found on a reachable cluster",
			err:         apierrors.NewNotFound(schema.GroupResource{Group: "cluster.x-k8s.io", Resource: "clusters"}, "test1234"),
			expectsHint: false,
		},
		{
			name: "case 11: missing permissions on a reachable cluster",
			err: apierrors.NewForbidden(schema.GroupResource{Group: "cluster.x-k8s.io", Resource: "clusters"},
				"test1234", errors.New("not allowed")),
			expectsHint: false,
		},
		{
			name:        "case 12: connection reset is not a login problem",
			err:         &url.Error{Op: "Get", URL: "https://api.example.com/api", Err: os.NewSyscallError("read", syscall.ECONNRESET)},
			expectsHint: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := hint(tc.err) != ""
			if got != tc.expectsHint {
				t.Fatalf("hint(%v) returned a hint == %t, expected %t", tc.err, got, tc.expectsHint)
			}
		})
	}
}

// TestHint_aggregate makes sure we also look into aggregated errors, which is
// how clientcmd reports an invalid kubeconfig.
func TestHint_aggregate(t *testing.T) {
	err := fmt.Errorf("outer: %w", errors.Join(errors.New("unrelated"), clientcmd.ErrEmptyConfig))

	if hint(err) == "" {
		t.Fatalf("expected a hint for an aggregated error containing %v", clientcmd.ErrEmptyConfig)
	}
}

// TestFormat_withHint makes sure the hint ends up in the printed output,
// separated from the error message itself.
func TestFormat_withHint(t *testing.T) {
	ep := New(Config{DisableColors: true})

	message := ep.Format(microerror.Mask(fmt.Errorf("failed to get server groups: %w", connectionRefused())))

	expected := "Hint: this looks like you are not logged into a Giant Swarm management cluster."
	if !strings.Contains(message, expected) {
		t.Fatalf("expected formatted output to contain %q, got:\n%s", expected, message)
	}
	if !strings.Contains(message, "get server groups") {
		t.Fatalf("expected formatted output to keep the original message, got:\n%s", message)
	}
}
