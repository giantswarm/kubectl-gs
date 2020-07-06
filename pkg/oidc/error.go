package oidc

import (
	"github.com/giantswarm/microerror"
)

var invalidChallengeError = &microerror.Error{
	Kind: "invalidChallengeError",
}

// IsInvalidChallenge asserts invalidChallengeError.
func IsInvalidChallenge(err error) bool {
	return microerror.Cause(err) == invalidChallengeError
}
