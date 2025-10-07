package catalogs

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/giantswarm/kubectl-gs/v5/pkg/commonconfig"
	catalogdata "github.com/giantswarm/kubectl-gs/v5/pkg/data/domain/catalog"
	"github.com/giantswarm/kubectl-gs/v5/pkg/output"
)

type runner struct {
	commonConfig *commonconfig.CommonConfig
	flag         *flag
	logger       micrologger.Logger
	fs           afero.Fs

	service catalogdata.Interface

	stdout io.Writer
	stderr io.Writer
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

	{
		err = r.getService()
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var name, selector string
	{
		if len(args) > 0 {
			name = strings.ToLower(args[0])
			selector = fmt.Sprintf("application.giantswarm.io/catalog=%s,latest=true", name)
		}
	}

	var namespace string
	{
		if r.flag.AllNamespaces {
			namespace = metav1.NamespaceAll
		} else {
			namespace, _, err = r.commonConfig.GetNamespace()
			if err != nil {
				return microerror.Mask(err)
			}
		}
	}

	var labelSelector labels.Selector
	{
		labelSelector, err = labels.Parse(selector)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var catalogResource catalogdata.Resource
	{

		options := catalogdata.GetOptions{
			AllNamespaces: r.flag.AllNamespaces,
			Name:          name,
			Namespace:     namespace,
			LabelSelector: labelSelector,
		}
		catalogResource, err = r.service.Get(ctx, options)
		if catalogdata.IsNotFound(err) {
			return microerror.Maskf(notFoundError, "A catalog '%s/%s' cannot be found.\n", options.Namespace, options.Name)
		} else if catalogdata.IsNoMatch(err) {
			r.printNoMatchOutput()
			return nil
		} else if catalogdata.IsNoResources(err) && output.IsOutputDefault(r.flag.print.OutputFormat) {
			r.printNoResourcesOutput()
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	err = r.printOutput(catalogResource, r.flag.MaxColWidth)
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

	serviceConfig := catalogdata.Config{
		Client: client.CtrlClient(),
	}
	r.service, err = catalogdata.New(serviceConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
