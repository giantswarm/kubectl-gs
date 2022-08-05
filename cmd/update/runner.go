package update

import (
	"context"
	"io"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/giantswarm/kubectl-gs/pkg/commonconfig"
)

type runner struct {
	commonConfig *commonconfig.CommonConfig
	configFlags  *genericclioptions.RESTClientGetter
	flag         *flag
	logger       micrologger.Logger
	stdout       io.Writer
	stderr       io.Writer
}

func (r *runner) Run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	err := r.flag.Validate()
	if err != nil {
		return microerror.Mask(err)
	}

	r.commonConfig = commonconfig.New(*r.configFlags)
	err = r.run(ctx, cmd, args)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *runner) run(ctx context.Context, cmd *cobra.Command, args []string) error {
	err := cmd.Help()
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
