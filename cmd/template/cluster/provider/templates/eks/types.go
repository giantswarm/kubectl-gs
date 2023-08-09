package eks

type DefaultAppsConfig struct {
	ClusterName  string `json:"clusterName,omitempty"`
	Organization string `json:"organization,omitempty"`
}

type ClusterConfig struct {
	Metadata *Metadata `json:"metadata,omitempty"`
}

type Metadata struct {
	Name         string `json:"name,omitempty"`
	Description  string `json:"description,omitempty"`
	Organization string `json:"organization,omitempty"`
}
