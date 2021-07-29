package labels

import "github.com/giantswarm/microerror"

var invalidLabelSpecError = &microerror.Error{
	Kind: "invalidLabelSpecError",
}

func IsInvalidLabelSpec(err error) bool {
	return microerror.Cause(err) == invalidLabelSpecError
}

var invalidLabelKeyError = &microerror.Error{
	Kind: "invalidLabelKeyError",
}

func IsInvalidLabelKey(err error) bool {
	return microerror.Cause(err) == invalidLabelKeyError
}

var invalidLabelValueError = &microerror.Error{
	Kind: "invalidLabelValueError",
}

func IsInvalidLabelValue(err error) bool {
	return microerror.Cause(err) == invalidLabelValueError
}
