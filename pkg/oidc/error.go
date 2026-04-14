package oidc

import (
	"github.com/giantswarm/microerror"
)

var invalidChallengeError = &microerror.Error{
	Kind: "invalidChallengeError",
}

var cannotDecodeTokenError = &microerror.Error{
	Kind: "cannotDecodeTokenError",
}

var cannotRenewTokenError = &microerror.Error{
	Kind: "cannotRenewTokenError",
}

var cannotGetDeviceCodeError = &microerror.Error{
	Kind: "cannotGetDeviceCodeError",
}

var cannotGetDeviceTokenError = &microerror.Error{
	Kind: "cannotGetDeviceTokenError",
}

var cannotParseJwtError = &microerror.Error{
	Kind: "cannotParseJwtError",
}

var authorizationPendingError = &microerror.Error{
	Kind: "authorizationPendingError",
}

// IsAuthorizationPendingError asserts authorizationPendingError.
func IsAuthorizationPendingError(err error) bool {
	return microerror.Cause(err) == authorizationPendingError
}

var tooManyAuthRequestsError = &microerror.Error{
	Kind: "tooManyAuthRequestsError",
}

// IsTooManyAuthRequestsError asserts tooManyAuthRequestsError.
func IsTooManyAuthRequestsError(err error) bool {
	return microerror.Cause(err) == tooManyAuthRequestsError
}
