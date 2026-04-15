package labels

import "github.com/giantswarm/microerror"

var invalidLabelSpecError = &microerror.Error{
	Kind: "invalidLabelSpecError",
}

var invalidLabelKeyError = &microerror.Error{
	Kind: "invalidLabelKeyError",
}

var invalidLabelValueError = &microerror.Error{
	Kind: "invalidLabelValueError",
}
