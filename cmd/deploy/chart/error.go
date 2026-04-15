package chart

import "github.com/giantswarm/microerror"

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

var invalidFlagError = &microerror.Error{
	Kind: "invalidFlagError",
}

var applyAbortedError = &microerror.Error{
	Kind: "applyAbortedError",
}

var confirmationRequiredError = &microerror.Error{
	Kind: "confirmationRequiredError",
}
