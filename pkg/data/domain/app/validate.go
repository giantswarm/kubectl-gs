package app

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os/exec"
	"path"

	applicationv1alpha1 "github.com/giantswarm/apiextensions/v2/pkg/apis/application/v1alpha1"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"

	"github.com/giantswarm/microerror"
	"github.com/qri-io/jsonschema"

	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	valuesSchemaAnnotationKey = "application.giantswarm.io/values-schema"
)

// Validate attempts to validate the values of a deployed app (or multiple apps
// depending on how this command is invoked) against the schema for those values.
func (s *Service) Validate(ctx context.Context, options ValidateOptions) (ValidationResults, error) {
	var err error

	namespace := options.Namespace

	// If the namespace is empty, set it to "default".
	if namespace == "" {
		namespace = defaultNamespace
	}

	// BUT if we want all namespaces, set it to 'metav1.NamespaceAll', aka ""
	// again so the client gets all namespaces.
	if options.AllNamespaces {
		namespace = metav1.NamespaceAll
	}

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

func (s *Service) validateByName(ctx context.Context, name, namespace string, customValuesSchema *jsonschema.Schema) (ValidationResults, error) {
	results := ValidationResults{}

	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	app := &applicationv1alpha1.App{}
	err := s.client.K8sClient.CtrlClient().Get(ctx, namespacedName, app)
	if err != nil {
		return results, microerror.Mask(err)
	}

	valuesSchema, schemaValidationResult, err := s.validateApp(ctx, *app, customValuesSchema)
	if err != nil {
		results = append(results, &ValidationResult{
			App:          *app,
			ValuesSchema: nil,
			Err:          microerror.Mask(err),
		})

		return results, nil
	}

	results = append(results, &ValidationResult{
		App:              *app,
		ValuesSchema:     valuesSchema,
		ValidationErrors: *schemaValidationResult.Errs,
	})

	return results, nil

}

func (s *Service) validateMultiple(ctx context.Context, namespace string, labelSelector string, customValuesSchema *jsonschema.Schema) (ValidationResults, error) {
	var err error

	parsedSelector, err := labels.Parse(labelSelector)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	options := &runtimeClient.ListOptions{
		Namespace:     namespace,
		LabelSelector: parsedSelector,
	}

	results := ValidationResults{}
	apps := &applicationv1alpha1.AppList{}
	err = s.client.K8sClient.CtrlClient().List(ctx, apps, options)
	if err != nil {
		return nil, microerror.Mask(err)
	} else if len(apps.Items) == 0 {
		return nil, microerror.Mask(noResourcesError)
	}

	// Iterate over all apps and fetch the AppCatalog CR, index.yaml, and
	// corresponding values.schema.json if it is defined for that app's version.
	for _, app := range apps.Items {
		valuesSchema, schemaValidationResult, err := s.validateApp(ctx, app, customValuesSchema)
		if err != nil {
			results = append(results, &ValidationResult{
				App:          app,
				ValuesSchema: nil,
				Err:          microerror.Mask(err),
			})

			continue
		}

		// Append the result to the results array.
		results = append(results, &ValidationResult{
			App:              app,
			ValuesSchema:     valuesSchema,
			ValidationErrors: *schemaValidationResult.Errs,
		})
	}

	return results, nil
}

func (s *Service) validateApp(ctx context.Context, app applicationv1alpha1.App, customValuesSchema *jsonschema.Schema) (*jsonschema.Schema, *jsonschema.ValidationState, error) {
	catalogName := app.Spec.Catalog

	// Fetch the catalog's index.yaml if we haven't tried to yet.
	index, catalog, err := s.fetchCatalogIndex(ctx, catalogName)
	if err != nil {
		return nil, nil, microerror.Mask(err)
	}

	var valuesSchema *jsonschema.Schema
	if customValuesSchema != nil {
		valuesSchema = customValuesSchema
	} else {
		// Check if the catalog metadata defines a schema.values.json for the specific
		// version of this app and download it.
		// Q: Why not just check for schema.values.json in the tarball?
		// A: For now trying to stick to the spec as we made it. Didn't foresee
		// that I'd be downloading the Tarball anyways.
		valuesSchema, err = s.fetchValuesSchema(index.Entries[app.Spec.Name], app.Spec.Version)
		if err != nil {
			return nil, nil, microerror.Mask(err)
		}
	}

	// Use the helm binary to fetch and untar the chart tarball.
	// Why? There are actually a lot of little security details that go into
	// unpacking this tarball. Checkout https://github.com/helm/helm/blob/master/pkg/chart/loader/archive.go#L101
	url := findTarballURL(index.Entries[app.Spec.Name], app.Spec.Version)
	tmpDir, err := ioutil.TempDir("", app.Spec.Name+"-"+app.Spec.Version+"-")
	if err != nil {
		return nil, nil, microerror.Mask(err)
	}

	cmd := exec.Command("helm3", "pull", url, "--untar", "-d", tmpDir)
	_, err = cmd.CombinedOutput()
	if err != nil {
		return nil, nil, microerror.Maskf(commandError, "failed to execute: %s, %s", cmd.String(), err.Error())
	}

	// Gather all the values that we want to merge together. (1,2,3,4)
	// 1. Chart values (the starting point, what is in the values.yaml file of
	// the chart itself)
	valuesFilePath := path.Join(tmpDir, app.Spec.Name, "values.yaml")
	chartValuesYamlFile, err := ioutil.ReadFile(valuesFilePath)
	if err != nil {
		return nil, nil, microerror.Maskf(ioError, "failed to read: %s", valuesFilePath)
	}

	chartValues := make(map[string]interface{})
	err = yaml.Unmarshal(chartValuesYamlFile, &chartValues)
	if err != nil {
		return nil, nil, microerror.Maskf(ioError, "failed to unmarshal yaml file: %s", err.Error())
	}

	// Use MergeAll from the values package in app-operator to fetch and merge the
	// various values that users and admins can provide in the three configuration
	// levels and merges them together into a single result.
	// 2. Catalog values (configmap & secret)
	// 3. Cluster values (configmap & secret)
	// 4. User values (configmap & secret)
	providedValues, err := s.valuesService.MergeAll(ctx, app, *catalog)
	if err != nil {
		return nil, nil, microerror.Maskf(ioError, "failed fetch and/or merge user provided values: %s", err.Error())
	}

	// Finally, merge the user & admin provided values with the chart values.
	mergedData := mergeValues(chartValues, providedValues)

	// Validate the merged values against the jsonschema.
	schemaValidationResult := valuesSchema.Validate(ctx, mergedData)

	return valuesSchema, schemaValidationResult, nil
}

func (s *Service) fetchValuesSchema(entries ChartVersions, version string) (*jsonschema.Schema, error) {
	valuesSchemaURL := findValuesSchemaURL(entries, version)

	// Don't try to fetch something that isn't defined.
	if valuesSchemaURL == "" {
		return nil, microerror.Maskf(noSchemaError, "app has no url to 'values.schema.json' defined in the 'application.giantswarm.io/values-schema' annotation")
	}

	// Skip all the HTTP requests and Unmarshalling if we already fetched this schema.
	if result, isCached := s.schemaFetchResults[valuesSchemaURL]; isCached {
		return result.schema, nil
	}

	// Fetch the values.schema.json file.
	// nosec is applied here because this is not a web server. The only thing we do
	// with this response is attempt to unmarshal it into a jsonschema.Schema.
	resp, err := http.Get(valuesSchemaURL) // #nosec
	if err != nil {
		err = microerror.Maskf(fetchError, "unable to fetch values.schema.json, http error: %s", err.Error())
		s.schemaFetchResults[valuesSchemaURL] = SchemaFetchResult{
			err: err,
		}
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = microerror.Maskf(fetchError, "unable to fetch values.schema.json, error processing http response body: %s", err.Error())
		s.schemaFetchResults[valuesSchemaURL] = SchemaFetchResult{
			err: err,
		}
		return nil, err
	}

	// Unmarshal values.schema.json file into a jsonschema.Schema.
	valuesSchema := jsonschema.Schema{}
	err = json.Unmarshal(body, &valuesSchema)
	if err != nil {
		err = microerror.Maskf(fetchError, "unable to unmarshal values.schema.json: %s", err.Error())
		s.schemaFetchResults[valuesSchemaURL] = SchemaFetchResult{
			err: err,
		}
		return nil, err
	}

	// Cache the result.
	s.schemaFetchResults[valuesSchemaURL] = SchemaFetchResult{
		schema: &valuesSchema,
	}

	return &valuesSchema, nil
}

func (s *Service) fetchCatalogIndex(ctx context.Context, catalogName string) (*IndexFile, *applicationv1alpha1.AppCatalog, error) {
	var err error

	// Don't try to fetch something that is undefined.
	if catalogName == "" {
		return nil, nil, microerror.Maskf(fetchError, "app has empty catalog name")
	}

	// Skip all the HTTP requests and Unmarshalling if we already fetched this catalog.
	if result, isCached := s.catalogFetchResults[catalogName]; isCached {
		return result.index, result.catalog, nil
	}

	// Fetch the catalog.
	objKey := runtimeClient.ObjectKey{
		Name: catalogName,
	}

	catalog := &applicationv1alpha1.AppCatalog{}
	err = s.client.K8sClient.CtrlClient().Get(ctx, objKey, catalog)
	if err != nil {
		err = microerror.Maskf(fetchError, "unable to fetch AppCatalog: %s", err.Error())
		s.catalogFetchResults[catalogName] = CatalogFetchResult{
			err: err,
		}

		return nil, nil, err
	}

	// Error for catalogs where we for sure can't fetch the index because we don't
	// know about the storage type yet.
	if catalog.Spec.Storage.Type != "helm" {
		err = microerror.Maskf(fetchError, "unable to fetch index, storage type %q is not supported", catalog.Spec.Storage.Type)
		s.catalogFetchResults[catalogName] = CatalogFetchResult{
			err: err,
		}

		return nil, nil, err
	}

	// Error for catalogs where we for sure can't fetch the index because the
	// URL is missing in the AppCatalog CR.
	if catalog.Spec.Storage.URL == "" {
		err = microerror.Maskf(fetchError, "unable to fetch index, the URL for the helm repo's index.yaml is missing from 'Spec.Storage.URL' in the AppCatalog CR")
		s.catalogFetchResults[catalogName] = CatalogFetchResult{
			err: err,
		}

		return nil, nil, err
	}

	// Fetch the index.
	resp, err := http.Get(catalog.Spec.Storage.URL + "/index.yaml")
	if err != nil {
		err = microerror.Maskf(fetchError, "unable to fetch index, http request failed: %s", err.Error())
		s.catalogFetchResults[catalogName] = CatalogFetchResult{
			err: err,
		}

		return nil, nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = microerror.Maskf(fetchError, "unable to fetch index, error processing http response body: %s", err.Error())
		s.catalogFetchResults[catalogName] = CatalogFetchResult{
			err: err,
		}

		return nil, nil, err
	}

	index := &IndexFile{}
	err = yaml.Unmarshal(body, index)
	if err != nil {
		err = microerror.Maskf(fetchError, "unable to fetch index, error unmarshalling body: %s", err.Error())
		s.catalogFetchResults[catalogName] = CatalogFetchResult{
			err: err,
		}

		return nil, nil, err
	}

	// Cache the succesfull result.
	s.catalogFetchResults[catalogName] = CatalogFetchResult{
		catalog: catalog,
		index:   index,
		err:     nil,
	}

	// Return the AppCatalog CR and the unmarshalled index.yaml.
	return index, catalog, nil
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
