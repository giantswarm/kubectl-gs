package cluster

import (
	"github.com/giantswarm/microerror"
)

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

var templateFlagNotImplemented = &microerror.Error{
	Kind: "templateFlagsNotImplementedError",
}
