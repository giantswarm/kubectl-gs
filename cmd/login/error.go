package login

import "github.com/giantswarm/microerror"

var fileExistsError = &microerror.Error{
	Kind: "fileExistsError",
}

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

var invalidFlagError = &microerror.Error{
	Kind: "invalidFlagError",
}

var invalidAuthResult = &microerror.Error{
	Kind: "invalidAuthResult",
}

var unknownUrlError = &microerror.Error{
	Kind: "unknownUrlError",
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

var contextAlreadySelectedError = &microerror.Error{
	Kind: "contextAlreadySelectedError",
}

// IsContextAlreadySelected asserts contextAlreadySelectedError.
func IsContextAlreadySelected(err error) bool {
	return microerror.Cause(err) == contextAlreadySelectedError
}

var invalidAuthConfigurationError = &microerror.Error{
	Kind: "invalidAuthConfigurationError",
}

var newLoginRequiredError = &microerror.Error{
	Kind: "newLoginRequiredError",
}

// IsNewLoginRequired asserts newLoginRequiredError.
func IsNewLoginRequired(err error) bool {
	return microerror.Cause(err) == newLoginRequiredError
}

var authResponseTimedOutError = &microerror.Error{
	Kind: "authResponseTimedOutError",
}

// IsAuthResponseTimedOut asserts authResponseTimedOutError.
func IsAuthResponseTimedOut(err error) bool {
	return microerror.Cause(err) == authResponseTimedOutError
}

var credentialRetrievalTimedOut = &microerror.Error{
	Kind: "credentialRetrievalTimedOut",
}

var organizationNotFoundError = &microerror.Error{
	Kind: "organizationNotFoundError",
}

var unknownOrganizationNamespaceError = &microerror.Error{
	Kind: "unknownOrganizationNamespaceError",
}

var clusterNotFoundError = &microerror.Error{
	Kind: "clusterNotFoundError",
}

// IsClusterNotFound asserts clusterNotFoundError.
func IsClusterNotFound(err error) bool {
	return microerror.Cause(err) == clusterNotFoundError
}

var noOrganizationsError = &microerror.Error{
	Kind: "noOrganizationsError",
}

var multipleClustersFoundError = &microerror.Error{
	Kind: "multipleClustersFoundError",
}

var insufficientPermissionsError = &microerror.Error{
	Kind: "insufficientPermissionsError",
}

// IsInsufficientPermissions asserts insufficientPermissionsError.
func IsInsufficientPermissions(err error) bool {
	return microerror.Cause(err) == insufficientPermissionsError
}

var clusterAPINotReadyError = &microerror.Error{
	Kind: "clusterAPINotReadyError",
}

// IsClusterAPINotReady asserts clusterAPINotReadyError.
func IsClusterAPINotReady(err error) bool {
	return microerror.Cause(err) == clusterAPINotReadyError
}

var clusterAPINotKnownError = &microerror.Error{
	Kind: "clusterAPINotKnownError",
}

// IsClusterAPINotKnown asserts clusterAPINotKnownError.
func IsClusterAPINotKnown(err error) bool {
	return microerror.Cause(err) == clusterAPINotKnownError
}

var deviceAuthError = &microerror.Error{
	Kind: "deviceAuthError",
}

var structuredAuthMultipleIssuersError = &microerror.Error{
	Kind: "structuredAuthMultipleIssuersError",
}

var structuredAuthIssuerNotFoundError = &microerror.Error{
	Kind: "structuredAuthIssuerNotFoundError",
}

var structuredAuthCANotFoundError = &microerror.Error{
	Kind: "structuredAuthCANotFoundError",
}
