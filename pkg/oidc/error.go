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

var cannotDecodeTokenError = &microerror.Error{
	Kind: "cannotDecodeTokenError",
}

// IsCannotDecodeToken asserts cannotDecodeTokenError.
func IsCannotDecodeToken(err error) bool {
	return microerror.Cause(err) == cannotDecodeTokenError
}

var cannotRenewTokenError = &microerror.Error{
	Kind: "cannotRenewTokenError",
}

// IsCannotRenewToken asserts cannotRenewTokenError.
func IsCannotRenewToken(err error) bool {
	return microerror.Cause(err) == cannotRenewTokenError
}
