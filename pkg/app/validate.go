package app

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"

	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/xeipuuv/gojsonschema"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/v2/pkg/data/domain/app"
	appdata "github.com/giantswarm/kubectl-gs/v2/pkg/data/domain/app"
	catalogdata "github.com/giantswarm/kubectl-gs/v2/pkg/data/domain/catalog"
	"github.com/giantswarm/kubectl-gs/v2/pkg/helmbinary"
)

const (
	valuesSchemaAnnotationKey = "application.giantswarm.io/values-schema"
)

// Validate attempts to validate the values of a deployed app (or multiple apps
// depending on how this command is invoked) against the schema for those values.
func (s *Service) Validate(ctx context.Context, options ValidateOptions) (ValidationResults, error) {
	var err error

	namespace := options.Namespace
	selector := options.LabelSelector

	var results ValidationResults
	if options.Name != "" {
		results, err = s.validateByName(ctx, options.Name, namespace, options.ValuesSchema)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	} else {
		results, err = s.validateMultiple(ctx, namespace, selector, options.ValuesSchema)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return results, nil
}

func (s *Service) validateByName(ctx context.Context, name, namespace string, customValuesSchema string) (ValidationResults, error) {
	results := ValidationResults{}

	options := appdata.GetOptions{
		Namespace: namespace,
		Name:      name,
	}

	appResource, err := s.AppDataService.Get(ctx, options)
	if app.IsNotFound(err) {
		return nil, microerror.Maskf(notFoundError, fmt.Sprintf("An app '%s/%s' cannot be found.\n", options.Namespace, options.Name))
	} else if err != nil {
		return results, microerror.Mask(err)
	}

	var appCR *applicationv1alpha1.App

	switch a := appResource.(type) {
	case *app.App:
		appCR = a.CR
	default:
		return results, microerror.Maskf(invalidTypeError, "unexpected type %T found", a)
	}

	valuesSchema, schemaValidationResult, err := s.ValidateApp(ctx, appCR, customValuesSchema, nil)
	if err != nil {
		results = append(results, &ValidationResult{
			App:          appCR,
			ValuesSchema: "",
			Err:          microerror.Mask(err),
		})

		return results, nil
	}

	results = append(results, &ValidationResult{
		App:              appCR,
		ValuesSchema:     valuesSchema,
		ValidationErrors: schemaValidationResult.Errors(),
	})

	return results, nil
}

func (s *Service) validateMultiple(ctx context.Context, namespace string, labelSelector string, customValuesSchema string) (ValidationResults, error) {
	var err error
	results := ValidationResults{}

	options := appdata.GetOptions{
		Namespace:     namespace,
		LabelSelector: labelSelector,
	}

	appResource, err := s.AppDataService.Get(ctx, options)
	if err != nil {
		return results, microerror.Mask(err)
	}

	var apps []*applicationv1alpha1.App

	switch a := appResource.(type) {
	case *app.Collection:
		for _, appItem := range a.Items {
			apps = append(apps, appItem.CR)
		}
	default:
		return results, microerror.Maskf(invalidTypeError, "unexpected type %T found", a)
	}

	if len(apps) == 0 {
		return results, microerror.Mask(noResourcesError)
	}

	// Iterate over all apps and fetch the Catalog CR, index.yaml, and
	// corresponding values.schema.json if it is defined for that app's version.
	for _, app := range apps {
		valuesSchema, schemaValidationResult, err := s.ValidateApp(ctx, app, customValuesSchema, nil)
		if err != nil {
			results = append(results, &ValidationResult{
				App:          app,
				ValuesSchema: "",
				Err:          microerror.Mask(err),
			})

			continue
		}

		// Append the result to the results array if there are any.
		if schemaValidationResult != nil {
			results = append(results, &ValidationResult{
				App:              app,
				ValuesSchema:     valuesSchema,
				ValidationErrors: schemaValidationResult.Errors(),
			})
		}
	}

	return results, nil
}

func (s *Service) ValidateApp(ctx context.Context, app *applicationv1alpha1.App, customValuesSchema string, yamlData map[string]interface{}) (string, *gojsonschema.Result, error) {
	catalogName := app.Spec.Catalog
	catalogNamespace := app.Spec.CatalogNamespace

	// Fetch the catalog's index.yaml if we haven't tried to yet.
	index, catalog, err := s.fetchCatalogIndex(ctx, catalogName, catalogNamespace)
	if err != nil {
		return "", nil, microerror.Mask(err)
	}

	var valuesSchema string
	if customValuesSchema != "" {
		valuesSchema = customValuesSchema
	} else {
		// Check if the catalog metadata defines a schema.values.json for the specific
		// version of this app and download it.
		// Q: Why not just check for schema.values.json in the tarball?
		// A: For now trying to stick to the spec as we made it. Didn't foresee
		// that I'd be downloading the Tarball anyways.
		valuesSchema, err = s.fetchValuesSchema(index.Entries[app.Spec.Name], app.Spec.Version)
		if err != nil {
			return "", nil, microerror.Mask(err)
		}
	}

	// Fetch and untar the chart tarball.
	pullOptions := helmbinary.PullOptions{
		URL: findTarballURL(index.Entries[app.Spec.Name], app.Spec.Version),
	}
	tmpDir, err := s.HelmbinaryService.Pull(ctx, pullOptions)
	if err != nil {
		return "", nil, microerror.Mask(err)
	}

	// if no user defined manifest data is given, discover and merge
	// all the app-platform managed values for schema validation
	if yamlData == nil {

		// Gather all the values that we want to merge together. (1,2,3,4)
		// 1. Chart values (the starting point, what is in the values.yaml file of
		// the chart itself)
		valuesFilePath := path.Join(tmpDir, app.Spec.Name, "values.yaml")
		chartValuesYamlFile, err := os.ReadFile(valuesFilePath)
		if err != nil {
			return "", nil, microerror.Maskf(ioError, "failed to read: %s", valuesFilePath)
		}

		chartValues := make(map[string]interface{})
		err = yaml.Unmarshal(chartValuesYamlFile, &chartValues)
		if err != nil {
			return "", nil, microerror.Maskf(ioError, "failed to unmarshal yaml file: %s", err.Error())
		}

		// Use MergeAll from the values package in the app library repo to fetch and
		// merge the various values that users and admins can provide in the three
		// configuration levels and merges them together into a single result.
		// 2. Catalog values (configmap & secret)
		// 3. Cluster values (configmap & secret)
		// 4. User values (configmap & secret)
		providedValues, err := s.ValuesService.MergeAll(ctx, *app, *catalog)
		if err != nil {
			return "", nil, microerror.Maskf(ioError, "failed fetch and/or merge user provided values: %s", err.Error())
		}

		// Finally, merge the user & admin provided values with the chart values.
		yamlData = mergeValues(chartValues, providedValues)
	}

	// Validate the merged values against the schema using gojsonschema.
	result, err := ValidateSchema(valuesSchema, yamlData)
	if err != nil {
		return "", nil, microerror.Mask(err)
	}

	return valuesSchema, result, nil
}

func (s *Service) fetchValuesSchema(entries ChartVersions, version string) (string, error) {
	valuesSchemaURL := findValuesSchemaURL(entries, version)

	// Don't try to fetch something that isn't defined.
	if valuesSchemaURL == "" {
		return "", microerror.Maskf(noSchemaError, "app has no url to 'values.schema.json' defined in the 'application.giantswarm.io/values-schema' annotation")
	}

	// Skip all the HTTP requests and Unmarshalling if we already fetched this schema.
	if result, isCached := s.SchemaFetchResults[valuesSchemaURL]; isCached {
		return result.schema, nil
	}

	// Fetch the values.schema.json file.
	// nosec is applied here because this is not a web server. The only thing we do
	// with this response is attempt to unmarshal it into a jsonschema.Schema.
	resp, err := http.Get(valuesSchemaURL) // #nosec
	if err != nil {
		err = microerror.Maskf(fetchError, "unable to fetch values.schema.json, http error: %s", err.Error())
		s.SchemaFetchResults[valuesSchemaURL] = SchemaFetchResult{
			err: err,
		}
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		err = microerror.Maskf(fetchError, "unable to fetch values.schema.json, error processing http response body: %s", err.Error())
		s.SchemaFetchResults[valuesSchemaURL] = SchemaFetchResult{
			err: err,
		}
		return "", err
	}

	// Cache the result.
	s.SchemaFetchResults[valuesSchemaURL] = SchemaFetchResult{
		schema: string(body),
	}

	return string(body), nil
}

func (s *Service) fetchCatalogIndex(ctx context.Context, catalogName, catalogNamespace string) (*IndexFile, *applicationv1alpha1.Catalog, error) {
	var err error

	// Don't try to fetch something that is undefined.
	if catalogName == "" {
		return nil, nil, microerror.Maskf(fetchError, "app has empty catalog name")
	}

	// Skip all the HTTP requests and Unmarshalling if we already fetched this catalog.
	if result, isCached := s.CatalogFetchResults[catalogName]; isCached {
		return result.index, result.catalog, nil
	}

	// Fetch the catalog.
	options := catalogdata.GetOptions{
		Name:      catalogName,
		Namespace: catalogNamespace,
	}

	catalogResource, err := s.CatalogDataService.Get(ctx, options)
	if err != nil {
		err = microerror.Maskf(fetchError, "unable to fetch Catalog: %s", err.Error())
		s.CatalogFetchResults[catalogName] = CatalogFetchResult{
			err: err,
		}

		return nil, nil, err
	}

	var catalog *applicationv1alpha1.Catalog

	switch c := catalogResource.(type) {
	case *catalogdata.Catalog:
		catalog = c.CR
	default:
		return nil, nil, microerror.Maskf(invalidTypeError, "unexpected type %T found", c)
	}

	// Pick a repository with type="helm", since we don't know how to fetch
	// indexes from other storage types yet.
	var catalogURL string
	var foundHelmRepository bool
	for _, repo := range catalog.Spec.Repositories {
		if repo.Type == "helm" {
			foundHelmRepository = true
			catalogURL = repo.URL
			break
		}
	}
	// Legacy: use deprecated .spec.storage in case .spec.repositories is empty.
	if catalogURL == "" && !foundHelmRepository && catalog.Spec.Storage.Type == "helm" {
		catalogURL = catalog.Spec.Storage.URL
		foundHelmRepository = true
	}
	// Error for catalogs where we for sure can't fetch the index because we don't
	// know about the storage type yet.
	if !foundHelmRepository {
		err = microerror.Maskf(fetchError, "unable to fetch index, only \"helm\" storage type is supported")
		s.CatalogFetchResults[catalogName] = CatalogFetchResult{
			err: err,
		}

		return nil, nil, err
	}

	// Error for catalogs where we for sure can't fetch the index because the
	// URL is missing in the Catalog CR.
	if catalogURL == "" {
		err = microerror.Maskf(fetchError, "unable to fetch index, the URL for the helm repo's index.yaml is missing from 'Spec.Repositories[].URL' in the Catalog CR")
		s.CatalogFetchResults[catalogName] = CatalogFetchResult{
			err: err,
		}

		return nil, nil, err
	}

	// Fetch the index.
	resp, err := http.Get(catalogURL + "/index.yaml")
	if err != nil {
		err = microerror.Maskf(fetchError, "unable to fetch index, http request failed: %s", err.Error())
		s.CatalogFetchResults[catalogName] = CatalogFetchResult{
			err: err,
		}

		return nil, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		err = microerror.Maskf(fetchError, "unable to fetch index, error processing http response body: %s", err.Error())
		s.CatalogFetchResults[catalogName] = CatalogFetchResult{
			err: err,
		}

		return nil, nil, err
	}

	index := &IndexFile{}
	err = yaml.Unmarshal(body, index)
	if err != nil {
		err = microerror.Maskf(fetchError, "unable to fetch index, error unmarshalling body: %s", err.Error())
		s.CatalogFetchResults[catalogName] = CatalogFetchResult{
			err: err,
		}

		return nil, nil, err
	}

	// Cache the succesfull result.
	s.CatalogFetchResults[catalogName] = CatalogFetchResult{
		catalog: catalog,
		index:   index,
		err:     nil,
	}

	// Return the Catalog CR and the unmarshalled index.yaml.
	return index, catalog, nil
}

func ValidateSchema(valuesSchema string, yamlData map[string]interface{}) (*gojsonschema.Result, error) {
	// Validate the merged values against the schema using gojsonschema.
	schemaLoader := gojsonschema.NewStringLoader(valuesSchema)
	documentLoader := gojsonschema.NewGoLoader(yamlData)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return result, nil
}

func findValuesSchemaURL(entries ChartVersions, version string) string {
	for _, entry := range entries {
		_, hasValuesSchema := entry.Annotations[valuesSchemaAnnotationKey]

		if entry.Version == version && hasValuesSchema {

			return entry.Annotations[valuesSchemaAnnotationKey]
		}
	}

	return ""
}

func findTarballURL(entries ChartVersions, version string) string {
	for _, entry := range entries {
		if entry.Version == version {
			return entry.URLs[0]
		}
	}

	return ""
}

// mergeValues implements the merge logic. It performs a deep merge. If a value
// is present in both then the source map is preferred.
//
// Logic is based on the upstream logic implemented by Helm.
// https://github.com/helm/helm/blob/240e539cec44e2b746b3541529d41f4ba01e77df/cmd/helm/install.go#L358
func mergeValues(dest, src map[string]interface{}) map[string]interface{} {
	for k, v := range src {
		if _, exists := dest[k]; !exists {
			// If the key doesn't exist already. Set the key to that value.
			dest[k] = v
			continue
		}

		nextMap, ok := v.(map[string]interface{})
		if !ok {
			// If it isn't another map. Overwrite the value.
			dest[k] = v
			continue
		}

		// Edge case: If the key exists in the destination but isn't a map.
		destMap, ok := dest[k].(map[string]interface{})
		if !ok {
			// If the source map has a map for this key. Prefer that value.
			dest[k] = v
			continue
		}

		// If we got to this point. It is a map in both so merge them.
		dest[k] = mergeValues(destMap, nextMap)
	}

	return dest
}
