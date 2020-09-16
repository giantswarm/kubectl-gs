package provider

type NodePoolCRsConfig struct {
	// AWS only.
	AWSInstanceType                     string
	OnDemandBaseCapacity                int
	OnDemandPercentageAboveBaseCapacity int
	UseAlikeInstanceTypes               bool

	// Common.
	FileName          string
	NodePoolID        string
	AvailabilityZones []string
	ClusterID         string
	Description       string
	NodesMax          int
	NodesMin          int
	Owner             string
	ReleaseComponents map[string]string
	ReleaseVersion    string
	Namespace         string
}
