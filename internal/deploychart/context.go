package deploychart

import "strings"

// Known prefixes for management cluster kubectl context names.
var mcContextPrefixes = []string{
	"gs-",
	"teleport.giantswarm.io-",
}

// ClusterNameFromContext extracts the management cluster name from a kubectl
// context name. Known formats are "gs-<mcname>" and "teleport.giantswarm.io-<mcname>".
// If no known prefix matches, the full context name is returned as fallback.
func ClusterNameFromContext(contextName string) string {
	for _, prefix := range mcContextPrefixes {
		if name, ok := strings.CutPrefix(contextName, prefix); ok && name != "" {
			return name
		}
	}
	return contextName
}
