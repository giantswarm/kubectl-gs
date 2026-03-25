package deploychart

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidateValuesAgainstSchema(t *testing.T) {
	schema := map[string]any{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type":    "object",
		"properties": map[string]any{
			"replicaCount": map[string]any{
				"type": "integer",
			},
			"image": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"repository": map[string]any{"type": "string"},
					"tag":        map[string]any{"type": "string"},
				},
				"required": []string{"repository"},
			},
		},
		"required": []string{"replicaCount"},
	}

	schemaJSON, _ := json.Marshal(schema)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(schemaJSON)
	}))
	defer srv.Close()

	tests := []struct {
		name       string
		values     map[string]any
		wantErr    bool
		errContain string
	}{
		{
			name: "valid values",
			values: map[string]any{
				"replicaCount": 3,
				"image": map[string]any{
					"repository": "nginx",
					"tag":        "latest",
				},
			},
		},
		{
			name: "missing required field",
			values: map[string]any{
				"image": map[string]any{
					"repository": "nginx",
				},
			},
			wantErr:    true,
			errContain: "replicaCount",
		},
		{
			name: "wrong type",
			values: map[string]any{
				"replicaCount": "not-a-number",
			},
			wantErr:    true,
			errContain: "replicaCount",
		},
		{
			name: "nested missing required",
			values: map[string]any{
				"replicaCount": 1,
				"image":        map[string]any{},
			},
			wantErr:    true,
			errContain: "repository",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateValuesAgainstSchema(tc.values, srv.URL+"/schema.json")
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tc.errContain != "" && !strings.Contains(err.Error(), tc.errContain) {
					t.Errorf("error %q should contain %q", err.Error(), tc.errContain)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateValuesAgainstSchema_SchemaNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	err := ValidateValuesAgainstSchema(map[string]any{"key": "value"}, srv.URL+"/schema.json")
	if err == nil {
		t.Fatal("expected error for 404 schema")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error should mention 404, got: %v", err)
	}
}

func TestValidateValuesAgainstSchema_InvalidSchema(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		errPart string
	}{
		{
			name:    "not JSON at all",
			body:    `this is not json`,
			errPart: "compiling",
		},
		{
			name:    "valid JSON but not a schema",
			body:    `["just", "an", "array"]`,
			errPart: "compiling",
		},
		{
			name:    "invalid type value in schema",
			body:    `{"type": "notavalidtype"}`,
			errPart: "compiling",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			err := ValidateValuesAgainstSchema(map[string]any{"key": "value"}, srv.URL+"/schema.json")
			if err == nil {
				t.Fatal("expected error for invalid schema")
			}
			if !strings.Contains(err.Error(), tc.errPart) {
				t.Errorf("error %q should contain %q", err.Error(), tc.errPart)
			}
		})
	}
}

func TestValidateValuesAgainstSchema_ExternalRef(t *testing.T) {
	// Serve a definitions file that the main schema references via $ref.
	//nolint:gosec // Test server only — r.Host is the httptest server address.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/schema.json":
			_, _ = w.Write([]byte(`{
				"$schema": "http://json-schema.org/draft-07/schema#",
				"type": "object",
				"properties": {
					"port": { "$ref": "` + "http://" + r.Host + `/defs.json#/$defs/portNumber` + `" }
				}
			}`))
		case "/defs.json":
			_, _ = w.Write([]byte(`{
				"$defs": {
					"portNumber": {
						"type": "integer",
						"minimum": 1,
						"maximum": 65535
					}
				}
			}`))
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer srv.Close()

	// Valid port.
	err := ValidateValuesAgainstSchema(map[string]any{"port": 8080}, srv.URL+"/schema.json")
	if err != nil {
		t.Fatalf("expected valid, got: %v", err)
	}

	// Invalid port (out of range).
	err = ValidateValuesAgainstSchema(map[string]any{"port": 99999}, srv.URL+"/schema.json")
	if err == nil {
		t.Fatal("expected validation error for out-of-range port")
	}
	if !strings.Contains(err.Error(), "port") {
		t.Errorf("error should mention port, got: %v", err)
	}
}
