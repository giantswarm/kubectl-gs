package key

import (
	"strconv"

	"github.com/giantswarm/apiextensions/v3/pkg/annotation"
	"github.com/giantswarm/apiextensions/v3/pkg/label"
)

func ReleaseVersion(getter LabelsGetter) string {
	return getter.GetLabels()[label.ReleaseVersion]
}

func MachinePoolName(getter AnnotationsGetter) string {
	annotations := getter.GetAnnotations()
	if annotations == nil {
		return ""
	}

	return annotations[annotation.MachinePoolName]
}

func MachinePoolScaling(getter AnnotationsGetter) (int, int) {
	annotations := getter.GetAnnotations()
	if annotations != nil {
		minReplicas, err := strconv.Atoi(annotations[annotation.NodePoolMinSize])
		if err != nil {
			return -1, -1
		}
		maxReplicas, err := strconv.Atoi(annotations[annotation.NodePoolMaxSize])
		if err != nil {
			return -1, -1
		}

		return minReplicas, maxReplicas
	}

	return -1, -1
}
