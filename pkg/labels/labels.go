package labels

import (
	"strings"

	k8smetadata "github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/giantswarm/kubectl-gs/v4/internal/label"
)

// Logic partially lifted from https://github.com/kubernetes/kubectl/blob/445ad13/pkg/cmd/label/label.go#L400-L425
// Licensed under Apache-2.0
func Parse(rawLabels []string) (map[string]string, error) {
	labels := map[string]string{}
	for _, labelSpec := range rawLabels {
		parts := strings.Split(labelSpec, "=")
		if len(parts) != 2 {
			return nil, microerror.Maskf(invalidLabelSpecError, labelSpec, "")
		}
		if strings.Contains(parts[0], label.ForbiddenLabelKeyPart) && parts[0] != k8smetadata.ServicePriority {
			return nil, microerror.Maskf(invalidLabelKeyError, "%q: containing forbidden substring '%s'", labelSpec, label.ForbiddenLabelKeyPart)
		}
		if errs := validation.IsQualifiedName(parts[0]); len(errs) != 0 {
			return nil, microerror.Maskf(invalidLabelKeyError, "%q: %s", labelSpec, strings.Join(errs, ";"))
		}
		if errs := validation.IsValidLabelValue(parts[1]); len(errs) != 0 {
			return nil, microerror.Maskf(invalidLabelValueError, "%q: %s", labelSpec, strings.Join(errs, ";"))
		}
		labels[parts[0]] = parts[1]
	}
	return labels, nil
}
