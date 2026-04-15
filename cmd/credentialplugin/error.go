package credentialplugin

import "github.com/giantswarm/microerror"

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

var credentialPluginError = &microerror.Error{
	Kind: "credentialPluginError",
}
