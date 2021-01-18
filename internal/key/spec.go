package key

type LabelsGetter interface {
	GetLabels() map[string]string
}

type AnnotationsGetter interface {
	GetAnnotations() map[string]string
}
