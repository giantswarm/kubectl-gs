package apps

import (
	"github.com/giantswarm/microerror"
)

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

var invalidFlagError = &microerror.Error{
	Kind: "invalidFlagError",
}
