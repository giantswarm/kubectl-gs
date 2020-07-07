package login

import "github.com/giantswarm/microerror"

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var invalidFlagError = &microerror.Error{
	Kind: "invalidFlagError",
}

// IsInvalidFlag asserts invalidFlagError.
func IsInvalidFlag(err error) bool {
	return microerror.Cause(err) == invalidFlagError
}

var invalidAuthResult = &microerror.Error{
	Kind: "invalidAuthResult",
}

// IsInvalidAuthResult asserts invalidAuthResult.
func IsInvalidAuthResult(err error) bool {
	return microerror.Cause(err) == invalidAuthResult
}

var unknownUrlError = &microerror.Error{
	Kind: "unknownUrlError",
}

// IsUnknownUrl asserts unknownUrlError.
func IsUnknownUrl(err error) bool {
	return microerror.Cause(err) == unknownUrlError
}

var contextDoesNotExistError = &microerror.Error{
	Kind: "contextDoesNotExistError",
}

// IsContextDoesNotExist asserts contextDoesNotExistError.
func IsContextDoesNotExist(err error) bool {
	return microerror.Cause(err) == contextDoesNotExistError
}

var incorrectConfigurationError = &microerror.Error{
	Kind: "incorrectConfigurationError",
}

// IsIncorrectConfiguration asserts incorrectConfigurationError.
func IsIncorrectConfiguration(err error) bool {
	return microerror.Cause(err) == incorrectConfigurationError
}

var tokenRenewalFailedError = &microerror.Error{
	Kind: "tokenRenewalFailedError",
}

// IsTokenRenewalFailed asserts tokenRenewalFailedError.
func IsTokenRenewalFailed(err error) bool {
	return microerror.Cause(err) == tokenRenewalFailedError
}

var selectedContextNonCompatibleError = &microerror.Error{
	Kind: "selectedContextNonCompatibleError",
}

// IsSelectedTokenNonCompatible asserts selectedContextNonCompatibleError.
func IsSelectedTokenNonCompatible(err error) bool {
	return microerror.Cause(err) == selectedContextNonCompatibleError
}
