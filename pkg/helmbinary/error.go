package helmbinary

import (
	"github.com/giantswarm/microerror"
)

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

var argumentError = &microerror.Error{
	Kind: "argumentError",
}

var commandError = &microerror.Error{
	Kind: "commandError",
}
