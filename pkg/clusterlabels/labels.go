package clusterlabels

import (
	"fmt"
	"k8s.io/apimachinery/pkg/util/validation"
	"strings"

	"github.com/giantswarm/kubectl-gs/internal/label"
)

// Logic partially lifted from https://github.com/kubernetes/kubectl/blob/445ad13/pkg/cmd/label/label.go#L400-L425
// Licensed under Apache-2.0
func Parse(rawLabels []string) (map[string]string, error) {
	labels := map[string]string{}
	for _, labelSpec := range rawLabels {
		if strings.Contains(labelSpec, "=") {
			parts := strings.Split(labelSpec, "=")
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid label spec: %v", labelSpec)
			}
			if strings.Contains(parts[0], label.ForbiddenLabelKeyPart) {
				return nil, fmt.Errorf("invalid label key: %q: containing forbidden substring '%s'", labelSpec, label.ForbiddenLabelKeyPart)
			}
			if errs := validation.IsQualifiedName(parts[0]); len(errs) != 0 {
				return nil, fmt.Errorf("invalid label key: %q: %s", labelSpec, strings.Join(errs, ";"))
			}
			if errs := validation.IsValidLabelValue(parts[1]); len(errs) != 0 {
				return nil, fmt.Errorf("invalid label value: %q: %s", labelSpec, strings.Join(errs, ";"))
			}
			labels[parts[0]] = parts[1]
		} else {
			return nil, fmt.Errorf("unknown label spec: %v", labelSpec)
		}
	}
	return labels, nil
}
