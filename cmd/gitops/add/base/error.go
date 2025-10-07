package app

import "github.com/giantswarm/microerror"

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

var invalidFlagError = &microerror.Error{
	Kind: "invalidFlagsError",
}

var invalidProviderError = &microerror.Error{
	Kind: "invalidProviderError",
}
