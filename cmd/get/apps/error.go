package apps

import "github.com/giantswarm/microerror"

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

var notFoundError = &microerror.Error{
	Kind: "notFoundError",
}
