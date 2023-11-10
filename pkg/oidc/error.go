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

var cannotGetDeviceCodeError = &microerror.Error{
	Kind: "cannotGetDeviceCodeError",
}

// IsCannotGetDeviceCodeError asserts cannotGetDeviceCodeError.
func IsCannotGetDeviceCodeError(err error) bool {
	return microerror.Cause(err) == cannotGetDeviceCodeError
}

var cannotGetDeviceTokenError = &microerror.Error{
	Kind: "cannotGetDeviceTokenError",
}

// IsCannotGetDeviceTokenError asserts  cannotGetDeviceTokenError.
func IsCannotGetDeviceTokenError(err error) bool {
	return microerror.Cause(err) == cannotGetDeviceTokenError
}

var cannotParseJwtError = &microerror.Error{
	Kind: "cannotParseJwtError",
}

// IsCannotParseJwtError asserts cannotParseJwtError.
func IsCannotParseJwtError(err error) bool {
	return microerror.Cause(err) == cannotParseJwtError
}
