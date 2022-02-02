package login

import "github.com/giantswarm/microerror"

var fileExistsError = &microerror.Error{
	Kind: "fileExistsError",
}

// IsFileExistsError asserts fileExistsError.
func IsFileExistsError(err error) bool {
	return microerror.Cause(err) == fileExistsError
}

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

// IsInvalidAuthConfiguration asserts invalidAuthConfigurationError.
func IsInvalidAuthConfiguration(err error) bool {
	return microerror.Cause(err) == invalidAuthConfigurationError
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

// IsCredentialRetrievalTimedOut asserts credentialRetrievalTimedOut.
func IsCredentialRetrievalTimedOut(err error) bool {
	return microerror.Cause(err) == credentialRetrievalTimedOut
}

var organizationNotFoundError = &microerror.Error{
	Kind: "organizationNotFoundError",
}

// IsOrganizationNotFound asserts organizationNotFoundError.
func IsOrganizationNotFound(err error) bool {
	return microerror.Cause(err) == organizationNotFoundError
}

var unknownOrganizationNamespaceError = &microerror.Error{
	Kind: "unknownOrganizationNamespaceError",
}

// IsUnknownOrganizationNamespace asserts unknownOrganizationNamespaceError.
func IsUnknownOrganizationNamespace(err error) bool {
	return microerror.Cause(err) == unknownOrganizationNamespaceError
}

var clusterNotFoundError = &microerror.Error{
	Kind: "clusterNotFoundError",
}

// IsClusterNotFound asserts clusterNotFoundError.
func IsClusterNotFound(err error) bool {
	return microerror.Cause(err) == clusterNotFoundError
}

var invalidReleaseVersionError = &microerror.Error{
	Kind: "invalidReleaseVersionError",
}

// IsInvalidReleaseVersion asserts invalidReleaseVersionError.
func IsInvalidReleaseVersion(err error) bool {
	return microerror.Cause(err) == invalidReleaseVersionError
}

var unsupportedReleaseVersionError = &microerror.Error{
	Kind: "unsupportedReleaseVersionError",
}

// IsUnsupportedReleaseVersion asserts unsupportedReleaseVersionError.
func IsUnsupportedReleaseVersion(err error) bool {
	return microerror.Cause(err) == unsupportedReleaseVersionError
}

var missingComponentError = &microerror.Error{
	Kind: "missingComponentError",
}

// IsMissingComponent asserts missingComponentError.
func IsMissingComponent(err error) bool {
	return microerror.Cause(err) == missingComponentError
}

var unsupportedProviderError = &microerror.Error{
	Kind: "unsupportedProviderError",
}

// IsUnsupportedProvider asserts unsupportedProviderError.
func IsUnsupportedProvider(err error) bool {
	return microerror.Cause(err) == unsupportedProviderError
}

var releaseNotFoundError = &microerror.Error{
	Kind: "releaseNotFoundError",
}

// IsReleaseNotFound asserts releaseNotFoundError.
func IsReleaseNotFound(err error) bool {
	return microerror.Cause(err) == releaseNotFoundError
}

var noOrganizationsError = &microerror.Error{
	Kind: "noOrganizationsError",
}

// IsNoOrganizations asserts noOrganizationsError.
func IsNoOrganizations(err error) bool {
	return microerror.Cause(err) == noOrganizationsError
}

var multipleClustersFoundError = &microerror.Error{
	Kind: "multipleClustersFoundError",
}

// IsMultipleClustersFound asserts multipleClustersFoundError.
func IsMultipleClustersFound(err error) bool {
	return microerror.Cause(err) == multipleClustersFoundError
}

var insufficientPermissionsError = &microerror.Error{
	Kind: "insufficientPermissionsError",
}

// IsInsufficientPermissions asserts insufficientPermissionsError.
func IsInsufficientPermissions(err error) bool {
	return microerror.Cause(err) == insufficientPermissionsError
}
