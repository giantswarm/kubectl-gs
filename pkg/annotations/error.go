package annotations

import "github.com/giantswarm/microerror"

var invalidAnnotationSpecError = &microerror.Error{
	Kind: "invalidAnnotationSpecError",
}

func IsInvalidAnnotationSpec(err error) bool {
	return microerror.Cause(err) == invalidAnnotationSpecError
}

var invalidAnnotationKeyError = &microerror.Error{
	Kind: "invalidAnnotationKeyError",
}

func IsInvalidAnnotationKey(err error) bool {
	return microerror.Cause(err) == invalidAnnotationKeyError
}

var annotationsTooBigError = &microerror.Error{
	Kind: "invalidAnnotationValueError",
}

func IsAnnotationsTooBigError(err error) bool {
	return microerror.Cause(err) == annotationsTooBigError
}
