package organization

import (
	"io"
	"os"

	"github.com/giantswarm/kubectl-gs/pkg/commonconfig"
	"github.com/spf13/cobra"
)

const (
	name        = "organization"
	description = "Template Organization CR."
)

type Config struct {
	CommonConfig *commonconfig.CommonConfig
	Stderr       io.Writer
	Stdout       io.Writer
}

func New(config Config) (*cobra.Command, error) {
	if config.Stderr == nil {
		config.Stderr = os.Stderr
	}
	if config.Stdout == nil {
		config.Stdout = os.Stdout
	}

	f := &flag{}

	r := &runner{
		commonConfig: config.CommonConfig,
		flag:         f,
		stderr:       config.Stderr,
		stdout:       config.Stdout,
	}

	c := &cobra.Command{
		Use:   name,
		Short: description,
		Long:  description,
		RunE:  r.Run,
	}

	f.Init(c)

	return c, nil
}
