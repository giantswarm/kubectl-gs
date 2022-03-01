package openstack

type ClusterConfig struct {
	ClusterDescription string        `json:"clusterDescription,omitempty"`
	DNSNameservers     []string      `json:"dnsNameservers,omitempty"`
	Organization       string        `json:"organization,omitempty"`
	CloudConfig        string        `json:"cloudConfig,omitempty"`
	CloudName          string        `json:"cloudName,omitempty"`
	ClusterName        string        `json:"clusterName,omitempty"`
	KubernetesVersion  string        `json:"kubernetesVersion,omitempty"`
	NodeCIDR           string        `json:"nodeCIDR,omitempty"`
	NetworkName        string        `json:"networkName,omitempty"`
	SubnetName         string        `json:"subnetName,omitempty"`
	ExternalNetworkID  string        `json:"externalNetworkID,omitempty"`
	OIDC               *OIDC         `json:"oidc,omitempty"`
	Bastion            *Bastion      `json:"bastion,omitempty"`
	NodeClasses        []NodeClass   `json:"nodeClasses,omitempty"`
	ControlPlane       *ControlPlane `json:"controlPlane,omitempty"`
	NodePools          []NodePool    `json:"nodePools,omitempty"`
}

type DefaultAppsConfig struct {
	ClusterName  string `json:"clusterName,omitempty"`
	Organization string `json:"organization,omitempty"`
	OIDC         *OIDC  `json:"oidc,omitempty"`
}

type OIDC struct {
	Enabled bool `json:"enabled"`
}

type MachineConfig struct {
	BootFromVolume bool   `json:"bootFromVolume"`
	DiskSize       int    `json:"diskSize"`
	Flavor         string `json:"flavor"`
	Image          string `json:"image"`
}

type Bastion struct {
	MachineConfig `json:",inline"`
}

type ControlPlane struct {
	MachineConfig `json:",inline"`
	Replicas      int `json:"replicas,omitempty"`
}

type NodeClass struct {
	Name          string `json:"name"`
	MachineConfig `json:",inline"`
}

type NodePool struct {
	Class         string `json:"class"`
	FailureDomain string `json:"failureDomain,omitempty"`
	Name          string `json:"name"`
	Replicas      int    `json:"replicas"`
}
