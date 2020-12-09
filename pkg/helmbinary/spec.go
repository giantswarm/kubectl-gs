package helmbinary

// helmbinary is a package that helps with executing the `helm` binary on the
// users computer.

import (
	"context"
)

// PullOptions are the parameters that the Pull method takes.
type PullOptions struct {
	URL string
}

// Interface represents the contract for the helmbinary service.
// Using this instead of a regular 'struct' makes mocking the
// service in tests much simpler.
type Interface interface {
	Pull(context.Context, PullOptions) (tmpDir string, err error)
}
