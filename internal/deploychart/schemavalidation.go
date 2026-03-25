package deploychart

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"sigs.k8s.io/yaml"
)

const (
	// ValuesSchemaAnnotation is the OCI manifest annotation that contains
	// the URL to a chart's values.schema.json file.
	ValuesSchemaAnnotation = "io.giantswarm.application.values-schema"

	schemaFetchTimeout = 30 * time.Second

	// maxSchemaSize is the maximum size of a fetched schema document (10 MB).
	maxSchemaSize = 10 * 1024 * 1024
)

// ValidateValuesAgainstSchema fetches a JSON Schema from schemaURL and validates
// the given values against it. External $ref URLs (http/https) are resolved
// automatically. Returns nil if validation passes.
func ValidateValuesAgainstSchema(values map[string]any, schemaURL string) error {
	compiler := jsonschema.NewCompiler()
	compiler.UseLoader(newSchemeURLLoader())

	schema, err := compiler.Compile(schemaURL)
	if err != nil {
		return fmt.Errorf("compiling values schema: %w", err)
	}

	// Convert values to JSON via UnmarshalJSON to get the right types
	// (json.Number for numbers, etc.).
	valuesYAML, err := yaml.Marshal(values)
	if err != nil {
		return fmt.Errorf("marshaling values: %w", err)
	}
	valuesJSON, err := yaml.YAMLToJSON(valuesYAML)
	if err != nil {
		return fmt.Errorf("converting values to JSON: %w", err)
	}
	valuesDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(valuesJSON))
	if err != nil {
		return fmt.Errorf("parsing values as JSON: %w", err)
	}

	err = schema.Validate(valuesDoc)
	if err == nil {
		return nil
	}

	validationErr, ok := err.(*jsonschema.ValidationError)
	if !ok {
		return fmt.Errorf("validating values against schema: %w", err)
	}

	// Collect errors, separating root-level from nested.
	bold := color.New(color.Bold).SprintFunc()
	var rootErrors, nestedErrors []string
	for _, detail := range validationErr.BasicOutput().Errors {
		if detail.Error == nil {
			continue
		}
		if detail.InstanceLocation == "" {
			rootErrors = append(rootErrors, fmt.Sprintf("  - %s: %s", bold("(root)"), detail.Error))
		} else {
			loc := strings.ReplaceAll(detail.InstanceLocation, "/", ".")
			nestedErrors = append(nestedErrors, fmt.Sprintf("  - %s: %s", bold(loc), detail.Error))
		}
	}

	var b strings.Builder
	b.WriteString("Values validation failed:\n \n")
	for _, e := range rootErrors {
		_, _ = fmt.Fprintf(&b, "%s\n", e)
	}
	for _, e := range nestedErrors {
		_, _ = fmt.Fprintf(&b, "%s\n", e)
	}
	return fmt.Errorf("%s", b.String())
}

// newSchemeURLLoader returns a URL loader that supports file, http, and https schemes.
func newSchemeURLLoader() *jsonschema.SchemeURLLoader {
	return &jsonschema.SchemeURLLoader{
		"file":  jsonschema.FileLoader{},
		"http":  newHTTPURLLoader(),
		"https": newHTTPURLLoader(),
	}
}

// httpURLLoader implements jsonschema.URLLoader for http/https URLs.
type httpURLLoader http.Client

func (l *httpURLLoader) Load(url string) (any, error) {
	client := (*http.Client)(l)
	resp, err := client.Get(url) //nolint:gosec // URL comes from chart schema $ref — expected to be an external HTTP resource.
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("%s returned status code %d", url, resp.StatusCode)
	}
	defer func() { _ = resp.Body.Close() }()

	return jsonschema.UnmarshalJSON(io.LimitReader(resp.Body, maxSchemaSize))
}

func newHTTPURLLoader() *httpURLLoader {
	return &httpURLLoader{
		Timeout: schemaFetchTimeout,
	}
}
