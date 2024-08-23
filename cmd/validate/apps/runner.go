package apps

import (
	"context"
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/kubectl-gs/v5/pkg/app"
	"github.com/giantswarm/kubectl-gs/v5/pkg/commonconfig"
)

const (
	defaultNamespace = metav1.NamespaceDefault
)

type runner struct {
	commonConfig *commonconfig.CommonConfig
	flag         *flag
	logger       micrologger.Logger
	service      app.Interface
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

	namespace, _, err := r.commonConfig.GetNamespace()
	if err != nil {
		return microerror.Mask(err)
	}

	// If the namespace is empty, set it to "default".
	if namespace == "" {
		namespace = defaultNamespace
	}

	// BUT if we want all namespaces, set it to 'metav1.NamespaceAll', aka ""
	// again so the client gets all namespaces.
	if r.flag.AllNamespaces {
		namespace = metav1.NamespaceAll
	}

	labelSelector := r.flag.LabelSelector

	valuesSchemaFilePath := r.flag.ValuesSchemaFile

	var valuesSchema string
	if valuesSchemaFilePath != "" {
		valuesSchemaFile, err := os.ReadFile(valuesSchemaFilePath)
		if err != nil {
			return microerror.Mask(err)
		}

		valuesSchema = string(valuesSchemaFile)
	}

	{
		err = r.getService()
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

func (r *runner) getService() error {
	if r.service != nil {
		return nil
	}

	client, err := r.commonConfig.GetClient(r.logger)
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
