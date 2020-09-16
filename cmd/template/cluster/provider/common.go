package provider

type ClusterCRConfig struct {
	// AWS only.
	ExternalSNAT bool
	PodsCIDR     string

	// Azure only.
	PublicSSHKey string

	// Common.
	FileName          string
	ClusterID         string
	Credential        string
	Domain            string
	MasterAZ          []string
	Description       string
	Owner             string
	Region            string
	ReleaseComponents map[string]string
	ReleaseVersion    string
	Labels            map[string]string
	Namespace         string
}
