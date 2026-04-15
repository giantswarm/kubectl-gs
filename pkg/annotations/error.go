package annotations

import "github.com/giantswarm/microerror"

var invalidAnnotationSpecError = &microerror.Error{
	Kind: "invalidAnnotationSpecError",
}

var invalidAnnotationKeyError = &microerror.Error{
	Kind: "invalidAnnotationKeyError",
}

var annotationsTooBigError = &microerror.Error{
	Kind: "invalidAnnotationValueError",
}
