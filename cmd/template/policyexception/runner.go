package policyexception

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/v2/pkg/commonconfig"
	template "github.com/giantswarm/kubectl-gs/v2/pkg/template/policyexception"
)

type runner struct {
	commonConfig *commonconfig.CommonConfig
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
	err = r.run(ctx, cmd, args)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *runner) run(ctx context.Context, cmd *cobra.Command, args []string) error {
	var err error

	config := template.Config{
		Name: r.flag.Draft,
	}

	client, err := r.commonConfig.GetClient(r.logger)
	if err != nil {
		return microerror.Mask(err)
	}

	var outputWriter io.Writer
	{
		if r.flag.Output == "" {
			outputWriter = r.stdout
		} else {
			f, err := os.Create(r.flag.Output)
			if err != nil {
				return microerror.Mask(err)
			}
			defer f.Close()

			outputWriter = f
		}
	}

	policyexceptionCR, err := template.NewPolicyExceptionCR(config, client.CtrlClient())
	if err != nil {
		return microerror.Mask(err)
	}

	policyexceptionCRYaml, err := yaml.Marshal(policyexceptionCR)
	if err != nil {
		return microerror.Mask(err)
	}

	_, err = fmt.Fprint(outputWriter, string(policyexceptionCRYaml))
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
