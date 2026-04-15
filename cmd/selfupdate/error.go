package selfupdate

import (
	"github.com/giantswarm/microerror"
)

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

var updateCheckFailedError = &microerror.Error{
	Kind: "updateCheckFailedError",
}
