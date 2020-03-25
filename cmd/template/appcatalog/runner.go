package appcatalog

import (
	"context"
	"io"
	"os"
	"text/template"

	"github.com/giantswarm/kubectl-gs/internal/key"
	appcatalog "github.com/giantswarm/kubectl-gs/pkg/template/appcatalog"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

type runner struct {
	flag   *flag
	logger micrologger.Logger
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
	config := appcatalog.Config{
		Description: r.flag.Description,
		Name:        r.flag.Name,
		URL:         r.flag.URL,
	}

	appCatalogCR, err := appcatalog.NewAppCatalogCR(config)

	appCatalogCRYaml, err := yaml.Marshal(appCatalogCR)
	if err != nil {
		return microerror.Mask(err)
	}

	type AppCatalogCROutput struct {
		AppCatalogCR string
	}

	appCatalogCROutput := AppCatalogCROutput{
		AppCatalogCR: string(appCatalogCRYaml),
	}

	t := template.Must(template.New("appCatalogCR").Parse(key.AppCatalogCRTemplate))

	err = t.Execute(os.Stdout, appCatalogCROutput)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
