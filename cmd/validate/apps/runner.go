package apps

import (
	"context"
	"io"
	"io/ioutil"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/pkg/data/domain/app"
)

type runner struct {
	flag    *flag
	logger  micrologger.Logger
	service app.Interface
	stdout  io.Writer
	stderr  io.Writer
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

	namespace, _, err := r.flag.config.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return microerror.Mask(err)
	}

	allNamespaces := r.flag.AllNamespaces
	labelSelector := r.flag.LabelSelector

	valuesSchemaFilePath := r.flag.ValuesSchemaFile

	var valuesSchema string
	if valuesSchemaFilePath != "" {
		valuesSchemaFile, err := ioutil.ReadFile(valuesSchemaFilePath)
		if err != nil {
			return microerror.Mask(err)
		}

		valuesSchema = string(valuesSchemaFile)
	}

	config := commonconfig.New(r.flag.config)
	{
		err = r.getService(config)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var results app.ValidationResults
	{
		options := app.ValidateOptions{}
		{
			if len(args) > 0 {
				options.Name = args[0]
			}

			options.Namespace = namespace
			options.AllNamespaces = allNamespaces
			options.LabelSelector = labelSelector
			options.ValuesSchema = valuesSchema
		}

		results, err = r.service.Validate(ctx, options)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	err = r.printOutput(results)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *runner) getService(config *commonconfig.CommonConfig) error {
	if r.service != nil {
		return nil
	}

	client, err := config.GetClient(r.logger)
	if err != nil {
		return microerror.Mask(err)
	}

	serviceConfig := app.Config{
		Client: client,
		Logger: r.logger,
	}
	r.service, err = app.New(serviceConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
