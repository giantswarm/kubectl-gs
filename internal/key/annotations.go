package key

const (
	// TODO: Take this from apiextensions (>=2.2.0).
	AnnotationNodePoolMinSize = "cluster.k8s.io/cluster-api-autoscaler-node-group-min-size"
	AnnotationNodePoolMaxSize = "cluster.k8s.io/cluster-api-autoscaler-node-group-max-size"
)
