package capz

type ClusterConfig struct {
	Connectivity     *Connectivity     `json:"connectivity,omitempty"`
	ControlPlane     *ControlPlane     `json:"controlPlane,omitempty"`
	Metadata         *Metadata         `json:"metadata,omitempty"`
	ProviderSpecific *ProviderSpecific `json:"providerSpecific,omitempty"`
}

type Metadata struct {
	Name         string `json:"name,omitempty"`
	Description  string `json:"description,omitempty"`
	Organization string `json:"organization,omitempty"`
}

type DefaultAppsConfig struct {
	ClusterName  string `json:"clusterName,omitempty"`
	Organization string `json:"organization,omitempty"`
}

type ProviderSpecific struct {
	Location       string `json:"location,omitempty"`
	SubscriptionID string `json:"subscriptionId,omitempty"`
}

type Connectivity struct {
	Bastion *Bastion `json:"bastion,omitempty"`
}

type Bastion struct {
	Enabled      bool   `json:"enabled,omitempty"`
	InstanceType string `json:"instanceType,omitempty"`
}

type ControlPlane struct {
	InstanceType string `json:"instanceType,omitempty"`
	Replicas     int    `json:"replicas,omitempty"`
}