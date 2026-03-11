package deploychart

import (
	"sigs.k8s.io/yaml"
)

// MarshalManifest marshals a Kubernetes resource to YAML, removing the empty
// status block that typed structs generate (empty struct is not omitted by
// omitempty since it's not a pointer).
func MarshalManifest(obj any) ([]byte, error) {
	raw, err := yaml.Marshal(obj)
	if err != nil {
		return nil, err
	}

	// Round-trip through a generic map to remove the empty status block.
	var m map[string]any
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return nil, err
	}

	delete(m, "status")

	return yaml.Marshal(m)
}
