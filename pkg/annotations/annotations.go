package annotations

import (
	"strings"

	"github.com/giantswarm/microerror"

	"k8s.io/apimachinery/pkg/util/validation"
)

// Source https://github.com/kubernetes/apimachinery/blob/v0.21.3/pkg/api/validation/objectmeta.go#L36
// Licensed under Apache-2.0
const totalAnnotationSizeLimitB int = 256 * (1 << 10) // 256 kB

// Logic partially lifted from https://github.com/kubernetes/apimachinery/blob/v0.21.3/pkg/api/validation/objectmeta.go#L47-L60
// Licensed under Apache-2.0
func Parse(rawAnnotations []string) (map[string]string, error) {
	annotations := map[string]string{}
	var totalSize int64

	for _, annotationSpec := range rawAnnotations {
		parts := strings.Split(annotationSpec, "=")
		if len(parts) != 2 {
			return nil, microerror.Maskf(invalidAnnotationSpecError, "%s", annotationSpec)
		}
		if errs := validation.IsQualifiedName(strings.ToLower(parts[0])); len(errs) != 0 {
			return nil, microerror.Maskf(invalidAnnotationKeyError, "%q: %s", annotationSpec, strings.Join(errs, ";"))
		}
		annotations[parts[0]] = parts[1]
		totalSize += (int64)(len(parts[0])) + (int64)(len(parts[1]))
	}

	if totalSize > (int64)(totalAnnotationSizeLimitB) {
		return nil, microerror.Maskf(annotationsTooBigError, "Annotations exceed size limit of %d bytes", totalAnnotationSizeLimitB)
	}

	return annotations, nil
}
