package errorprinter

import (
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/client-go/tools/clientcmd"
)

const notLoggedInHint = `Hint: this looks like you are not logged into a Giant Swarm management cluster.
Log in using 'kubectl gs login <management-cluster>'. The KUBECONFIG environment
variable or your current kubectl context must point to a management cluster, not
to a workload cluster. See https://docs.giantswarm.io/getting-started/management-cluster/`

// hint returns an actionable hint for errors whose cause is that the user is
// not logged into a management cluster, or has their kubectl context pointed
// somewhere else. For every other error it returns an empty string, so that
// unrelated failures are printed unchanged.
func hint(err error) string {
	if err == nil {
		return ""
	}

	if anyInChain(err, isNotLoggedIn) || messageLooksLikeNotLoggedIn(err.Error()) {
		return notLoggedInHint
	}

	return ""
}

// isNotLoggedIn reports whether a single error in the chain identifies a
// missing or misdirected management cluster connection.
func isNotLoggedIn(err error) bool {
	switch {
	// There is no kubeconfig at all, or no usable current context in it.
	case clientcmd.IsEmptyConfig(err), clientcmd.IsConfigurationInvalid(err):
		return true

	// Without a usable kubeconfig, client-go falls back to localhost:8080,
	// where nothing is listening.
	case utilnet.IsConnectionRefused(err):
		return true

	// The Giant Swarm CRDs are unknown to the API server, so this is a
	// workload cluster or not a Giant Swarm cluster at all.
	case meta.IsNoMatchError(err):
		return true

	// The kubeconfig exists but its token is no longer accepted. Note that we
	// deliberately do not treat 403 the same way: being logged in without the
	// required permissions is a different problem.
	case apierrors.IsUnauthorized(err):
		return true
	}

	return false
}

// messageLooksLikeNotLoggedIn is a fallback for the cases above, for error
// chains where an intermediate wrapper dropped the cause and the typed checks
// can therefore not see it.
func messageLooksLikeNotLoggedIn(message string) bool {
	message = strings.ToLower(message)

	return strings.Contains(message, "connection refused") ||
		strings.Contains(message, "no matches for kind")
}

// anyInChain reports whether pred holds for err or any error wrapped by it.
// We cannot rely on errors.As/errors.Is alone, because some of the predicates
// used above (the clientcmd ones in particular) type-assert on the error they
// are given instead of unwrapping it.
func anyInChain(err error, pred func(error) bool) bool {
	for err != nil {
		if pred(err) {
			return true
		}

		switch unwrappable := err.(type) {
		case interface{ Unwrap() error }:
			err = unwrappable.Unwrap()
		case interface{ Unwrap() []error }:
			for _, wrapped := range unwrappable.Unwrap() {
				if anyInChain(wrapped, pred) {
					return true
				}
			}

			return false
		default:
			return false
		}
	}

	return false
}
