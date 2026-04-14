package creator

import (
	"github.com/giantswarm/microerror"
)

var ValidationError = &microerror.Error{
	Kind: "validationError",
}
