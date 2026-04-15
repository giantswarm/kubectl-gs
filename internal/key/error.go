package key

import (
	"github.com/giantswarm/microerror"
)

// unmashalToMapFailedError is used when a YAML appCatalog values  can't be unmarshalled into map[string]interface{}.
var unmashalToMapFailedError = &microerror.Error{
	Kind: "unmashalToMapFailedError",
	Desc: "Could not unmarshal YAML into a map[string]interface{} structure. Seems like the YAML is invalid.",
}
