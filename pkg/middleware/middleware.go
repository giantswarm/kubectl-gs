package middleware

import (
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
)

type Middleware func(cmd *cobra.Command, args []string) error

// Compose runs a sequence of middleware, while maintaining
// the middleware interface, so it could be used in a cobra
// command.
func Compose(m ...Middleware) Middleware {
	return func(cmd *cobra.Command, args []string) error {
		var err error

		for _, middleware := range m {
			err = middleware(cmd, args)
			if err != nil {
				return microerror.Mask(err)
			}
		}

		return nil
	}
}
