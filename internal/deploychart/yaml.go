package deploychart

import (
	"sigs.k8s.io/yaml"
)

// MarshalCRD marshals a Kubernetes resource to YAML, removing empty fields
// that typed structs generate (status, creationTimestamp) and normalizing
// duration strings (e.g. "10m0s" → "10m").
func MarshalCRD(obj any) ([]byte, error) {
	raw, err := yaml.Marshal(obj)
	if err != nil {
		return nil, err
	}

	// Unmarshal into a generic map to strip unwanted fields.
	var m map[string]any
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return nil, err
	}

	delete(m, "status")

	if metadata, ok := m["metadata"].(map[string]any); ok {
		delete(metadata, "creationTimestamp")
	}

	cleanDurations(m)

	return yaml.Marshal(m)
}

// cleanDurations walks a map tree and normalizes Go duration strings
// by removing trailing zero components (e.g. "10m0s" → "10m").
func cleanDurations(m map[string]any) {
	for k, v := range m {
		switch val := v.(type) {
		case string:
			if k == "interval" || k == "timeout" {
				m[k] = normalizeDuration(val)
			}
		case map[string]any:
			cleanDurations(val)
		}
	}
}

func normalizeDuration(s string) string {
	// Remove trailing zero-value components: "10m0s" → "10m", "1h0m0s" → "1h".
	// Only strip "0s", "0m", "0h" when preceded by another unit (not a digit-zero).
	for {
		if len(s) >= 3 && s[len(s)-2:] == "0s" && isUnit(s[len(s)-3]) {
			s = s[:len(s)-2]
		} else if len(s) >= 3 && s[len(s)-2:] == "0m" && isUnit(s[len(s)-3]) {
			s = s[:len(s)-2]
		} else if len(s) >= 3 && s[len(s)-2:] == "0h" && isUnit(s[len(s)-3]) {
			s = s[:len(s)-2]
		} else {
			break
		}
	}
	return s
}

func isUnit(b byte) bool {
	return b == 's' || b == 'm' || b == 'h'
}
