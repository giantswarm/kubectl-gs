package deploy

import (
	"context"
	"fmt"

	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/kubectl-gs/v5/pkg/data/domain/catalog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *runner) listCatalogs(ctx context.Context) error {
	var catalogs *applicationv1alpha1.CatalogList

	err := RunWithSpinner("Listing catalogs", func() error {
		catalogs = &applicationv1alpha1.CatalogList{}
		// List catalogs in both default and giantswarm namespaces
		return r.ctrlClient.List(ctx, catalogs, &client.ListOptions{})
	})

	if err != nil {
		return err
	}

	if len(catalogs.Items) == 0 {
		fmt.Fprintf(r.stdout, "No catalogs found\n")
		return nil
	}

	output := ListCatalogsOutput(catalogs)
	fmt.Fprint(r.stdout, output)
	return nil
}

func (r *runner) getCatalogService() (catalog.Interface, error) {
	// Create a new catalog data service instance
	catalogConfig := catalog.Config{
		Client: r.ctrlClient,
	}
	catalogDataService, err := catalog.New(catalogConfig)
	if err != nil {
		return nil, err
	}

	return catalogDataService, nil
}
