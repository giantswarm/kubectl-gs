package deploychart

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestBuildRegistrySecret(t *testing.T) {
	opts := RegistrySecretOptions{
		Name:        "mycluster01-hello-world-app-registry",
		Namespace:   "org-acme",
		ClusterName: "mycluster01",
		Registry:    "gsoci.azurecr.io",
		Username:    "myuser",
		Password:    "mypassword",
	}

	secret, err := BuildRegistrySecret(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if secret.Name != opts.Name {
		t.Errorf("name = %q, want %q", secret.Name, opts.Name)
	}
	if secret.Namespace != opts.Namespace {
		t.Errorf("namespace = %q, want %q", secret.Namespace, opts.Namespace)
	}
	if secret.Type != corev1.SecretTypeDockerConfigJson {
		t.Errorf("type = %q, want %q", secret.Type, corev1.SecretTypeDockerConfigJson)
	}
	if secret.Labels["giantswarm.io/cluster"] != opts.ClusterName {
		t.Errorf("cluster label = %q, want %q", secret.Labels["giantswarm.io/cluster"], opts.ClusterName)
	}

	// Verify .dockerconfigjson content.
	raw, ok := secret.Data[corev1.DockerConfigJsonKey]
	if !ok {
		t.Fatal("missing .dockerconfigjson key in secret data")
	}

	var dockerConfig struct {
		Auths map[string]struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Auth     string `json:"auth"`
		} `json:"auths"`
	}
	if err := json.Unmarshal(raw, &dockerConfig); err != nil {
		t.Fatalf("parsing .dockerconfigjson: %v", err)
	}

	entry, ok := dockerConfig.Auths[opts.Registry]
	if !ok {
		t.Fatalf("missing auth entry for registry %q", opts.Registry)
	}
	if entry.Username != opts.Username {
		t.Errorf("username = %q, want %q", entry.Username, opts.Username)
	}
	if entry.Password != opts.Password {
		t.Errorf("password = %q, want %q", entry.Password, opts.Password)
	}
	expectedAuth := base64.StdEncoding.EncodeToString([]byte(opts.Username + ":" + opts.Password))
	if entry.Auth != expectedAuth {
		t.Errorf("auth = %q, want %q", entry.Auth, expectedAuth)
	}
}

func TestBuildRegistrySecretMarshal(t *testing.T) {
	opts := RegistrySecretOptions{
		Name:        "mycluster01-hello-world-app-registry",
		Namespace:   "org-acme",
		ClusterName: "mycluster01",
		Registry:    "gsoci.azurecr.io",
		Username:    "myuser",
		Password:    "mypassword",
	}

	secret, err := BuildRegistrySecret(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	yamlBytes, err := MarshalManifest(secret)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	// Verify it contains expected fields (not a golden-file test since
	// the base64-encoded dockerconfigjson varies with JSON key ordering).
	yaml := string(yamlBytes)
	for _, want := range []string{
		"kind: Secret",
		"apiVersion: v1",
		"name: mycluster01-hello-world-app-registry",
		"namespace: org-acme",
		"giantswarm.io/cluster: mycluster01",
		"type: kubernetes.io/dockerconfigjson",
	} {
		if !strings.Contains(yaml, want) {
			t.Errorf("YAML output missing %q:\n%s", want, yaml)
		}
	}
}
