package cluster

import (
	"github.com/giantswarm/microerror"
)

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

var invalidFlagError = &microerror.Error{
	Kind: "invalidFlagError",
}

var notFoundError = &microerror.Error{
	Kind: "notFoundError",
}

var notAllowedError = &microerror.Error{
	Kind: "notAllowedError",
}
