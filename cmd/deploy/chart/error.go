package chart

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

var applyAbortedError = &microerror.Error{
	Kind: "applyAbortedError",
}

// IsApplyAborted asserts applyAbortedError.
func IsApplyAborted(err error) bool {
	return microerror.Cause(err) == applyAbortedError
}

var confirmationRequiredError = &microerror.Error{
	Kind: "confirmationRequiredError",
}

// IsConfirmationRequired asserts confirmationRequiredError.
func IsConfirmationRequired(err error) bool {
	return microerror.Cause(err) == confirmationRequiredError
}

var resourceNotFoundError = &microerror.Error{
	Kind: "resourceNotFoundError",
}

// IsResourceNotFound asserts resourceNotFoundError.
func IsResourceNotFound(err error) bool {
	return microerror.Cause(err) == resourceNotFoundError
}
