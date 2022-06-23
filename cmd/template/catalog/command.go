package catalog

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"
)

const (
	name        = "catalog"
	description = "Template Catalog CR."
	example     = `
    Basic catalog with a single repository:

    kubectl-gs template catalog --name my-catalog --namespace default --logo https://example.com/img.jpg --description 'Custom catalog' --type helm --url https://example.com/helm-catalog/

    Catalog with a multiple repository mirrors:

    kubectl-gs template catalog --name my-catalog --namespace default --logo https://example.com/img.jpg --description 'Custom catalog' --type helm --url https://example.com/helm-catalog/ --type helm --url https://example.com/helm-mirror/ --type oci --url oci://example.com/oci-registry/
`
)

type Config struct {
	Logger micrologger.Logger
	Stderr io.Writer
	Stdout io.Writer
}

func New(config Config) (*cobra.Command, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.Stderr == nil {
		config.Stderr = os.Stderr
	}
	if config.Stdout == nil {
		config.Stdout = os.Stdout
	}

	f := &flag{}

	r := &runner{
		flag:   f,
		logger: config.Logger,
		stderr: config.Stderr,
		stdout: config.Stdout,
	}

	c := &cobra.Command{
		Use:     name,
		Short:   description,
		Long:    description,
		Example: example,
		RunE:    r.Run,
	}

	f.Init(c)

	return c, nil
}
