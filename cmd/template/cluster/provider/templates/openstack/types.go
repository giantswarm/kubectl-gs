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
	ExternalNetworkID  string        `json:"externalNetworkID,omitempty"`
	Bastion            *Bastion      `json:"bastion,omitempty"`
	NodeClasses        []NodeClass   `json:"nodeClasses,omitempty"`
	ControlPlane       *ControlPlane `json:"controlPlane,omitempty"`
	NodePools          []NodePool    `json:"nodePools,omitempty"`
	OIDC               *OIDC         `json:"oidc,omitempty"`
}

type DefaultAppsConfig struct {
	ClusterName  string `json:"clusterName,omitempty"`
	Organization string `json:"organization,omitempty"`
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

type OIDC struct {
	IssuerURL     string `json:"issuerUrl"`
	CAFile        string `json:"caFile"`
	ClientID      string `json:"clientId"`
	UsernameClaim string `json:"usernameClaim"`
	GroupsClaim   string `json:"groupsClaim"`
}

type NodePool struct {
	Class         string `json:"class"`
	FailureDomain string `json:"failureDomain,omitempty"`
	Name          string `json:"name"`
	Replicas      int    `json:"replicas"`
}
