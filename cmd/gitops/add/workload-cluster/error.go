package wcluster

import "github.com/giantswarm/microerror"

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

var invalidFlagsError = &microerror.Error{
	Kind: "invalidFlagsError",
}
