package app

import (
	"context"
	"fmt"
	"sort"
	"strings"

	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/microerror"
	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
	"github.com/muesli/reflow/wordwrap"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/giantswarm/kubectl-gs/v2/pkg/data/domain/catalog"
)

func (r *runner) promptCatalog() (*applicationv1alpha1.Catalog, error) {
	ctx := context.Background()
	o := catalog.GetOptions{}
	resource, err := r.catalogService.Get(ctx, o)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	catalogs := resource.(*catalog.Collection)
	entries := catalogs.Items
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].CR.GetName() > entries[j].CR.GetName()
	})

	idx, err := fuzzyfinder.Find(entries, func(i int) string {
		return entries[i].CR.GetName()
	}, fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
		if i == -1 {
			return ""
		}
		return fmt.Sprintf("   %s\n\n%s", entries[i].CR.Spec.Title, wordwrap.String(entries[i].CR.Spec.Description, h))
	}))

	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return entries[idx].CR, nil
}

func (r *runner) promptCatalogEntries(catalogName, appName, appVersion string) (*applicationv1alpha1.AppCatalogEntry, error) {
	var err error

	ctx := context.Background()
	o := catalog.ListOptions{}
	var selector []string
	{
		if len(catalogName) > 0 {
			selector = append(selector, fmt.Sprintf("application.giantswarm.io/catalog=%s", catalogName))
		}
		if len(appName) > 0 {
			selector = append(selector, fmt.Sprintf("app.kubernetes.io/name=%s", appName))
		}
		if len(appVersion) > 0 {
			selector = append(selector, fmt.Sprintf("app.kubernetes.io/version=%s", appVersion))
		}

	}
	var labelSelector labels.Selector
	{
		labelSelector, err = labels.Parse(strings.Join(selector, ","))
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}
	o.LabelSelector = labelSelector

	catalogEntries, err := r.catalogService.ListCatalogEntries(ctx, o)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	entries := catalogEntries.(*catalog.CatalogEntryList).Items
	sort.Slice(entries, func(i, j int) bool {
		return entries[j].Spec.DateUpdated.Before(entries[i].Spec.DateUpdated)
	})

	idx, err := fuzzyfinder.Find(entries, func(i int) string {
		return fmt.Sprintf("%s/%s@%s", entries[i].Spec.Catalog.Name, entries[i].Spec.AppName, entries[i].Spec.Version)
	}, fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
		if i == -1 {
			return ""
		}
		return fmt.Sprintf("   %s\n\n%s\n\nURL: %s\nUpdated: %s\nUpstream version: %s",
			entries[i].Spec.AppName,
			wordwrap.String(entries[i].Spec.Chart.Description, h),
			entries[i].Spec.Chart.Home,
			entries[i].Spec.DateUpdated.Format("2006-01-02 15:04:05"),
			entries[i].Spec.AppVersion,
		)
	}))
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return &entries[idx], nil
}
