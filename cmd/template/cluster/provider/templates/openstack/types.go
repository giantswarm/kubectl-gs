package openstack

type ClusterConfig struct {
	ClusterDescription string               `json:"clusterDescription,omitempty"`
	DNSNameservers     []string             `json:"dnsNameservers,omitempty"`
	Organization       string               `json:"organization,omitempty"`
	CloudConfig        string               `json:"cloudConfig,omitempty"`
	CloudName          string               `json:"cloudName,omitempty"`
	NodeCIDR           string               `json:"nodeCIDR,omitempty"`
	ExternalNetworkID  string               `json:"externalNetworkID,omitempty"`
	OIDC               *OIDC                `json:"oidc,omitempty"`
	Bastion            *Bastion             `json:"bastion,omitempty"`
	RootVolume         *RootVolume          `json:"rootVolume,omitempty"`
	NodeClasses        map[string]NodeClass `json:"nodeClasses,omitempty"`
	ControlPlane       *ControlPlane        `json:"controlPlane,omitempty"`
	NodePools          []NodePool           `json:"nodePools,omitempty"`
}

type DefaultAppsConfig struct {
	ClusterName  string `json:"clusterName,omitempty"`
	Organization string `json:"organization,omitempty"`
	OIDC         *OIDC  `json:"oidc,omitempty"`
}

type OIDC struct {
	Enabled bool `json:"enabled"`
}

type MachineRootVolume struct {
	DiskSize   int    `json:"diskSize"`
	SourceUUID string `json:"sourceUUID"`
}

type Bastion struct {
	Flavor     string            `json:"flavor"`
	Image      string            `json:"image"`
	RootVolume MachineRootVolume `json:"rootVolume"`
}

type RootVolume struct {
	Enabled    bool   `json:"enabled"`
	SourceUUID string `json:"sourceUUID"`
}

type NodeClass struct {
	MachineFlavor string `json:"machineFlavor"`
	DiskSize      int    `json:"diskSize"`
}

type ControlPlane struct {
	MachineFlavor string `json:"machineFlavor"`
	DiskSize      int    `json:"diskSize"`
	Replicas      int    `json:"replicas"`
}

type NodePool struct {
	Name     string `json:"name"`
	Class    string `json:"class"`
	Replicas int    `json:"replicas"`
}
