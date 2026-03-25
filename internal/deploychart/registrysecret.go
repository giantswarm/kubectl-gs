package deploychart

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RegistrySecretOptions contains the parameters for building a docker-registry Secret.
type RegistrySecretOptions struct {
	Name        string
	Namespace   string
	ClusterName string
	Registry    string
	Username    string
	Password    string
}

// BuildRegistrySecret creates a kubernetes.io/dockerconfigjson Secret
// that Flux uses to pull from private OCI registries.
func BuildRegistrySecret(opts RegistrySecretOptions) (*corev1.Secret, error) {
	dockerConfig, err := buildDockerConfigJSON(opts.Registry, opts.Username, opts.Password)
	if err != nil {
		return nil, fmt.Errorf("building docker config: %w", err)
	}

	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      opts.Name,
			Namespace: opts.Namespace,
			Labels: map[string]string{
				"giantswarm.io/cluster": opts.ClusterName,
			},
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: dockerConfig,
		},
	}, nil
}

// buildDockerConfigJSON produces the JSON content for a .dockerconfigjson key.
func buildDockerConfigJSON(registry, username, password string) ([]byte, error) {
	auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))

	config := map[string]any{
		"auths": map[string]any{
			registry: map[string]any{
				"username": username,
				"password": password,
				"auth":     auth,
			},
		},
	}

	return json.Marshal(config)
}
