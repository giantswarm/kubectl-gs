package callbackserver

import (
	"github.com/giantswarm/microerror"
)

var timedOutError = &microerror.Error{
	Kind: "timedOutError",
}

// IsTimedOut asserts timedOutError.
func IsTimedOut(err error) bool {
	return microerror.Cause(err) == timedOutError
}
