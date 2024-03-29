package capg

type ClusterConfig struct {
	ClusterName         string               `json:"clusterName,omitempty"`
	ClusterDescription  string               `json:"clusterDescription,omitempty"`
	Organization        string               `json:"organization,omitempty"`
	GCP                 *GCP                 `json:"gcp,omitempty"`
	Network             *Network             `json:"network,omitempty"`
	BastionInstanceType string               `json:"bastion,omitempty"`
	ControlPlane        *ControlPlane        `json:"controlPlane,omitempty"`
	MachineDeployments  *[]MachineDeployment `json:"machineDeployments,omitempty"`
	SSHSSOPublicKey     string               `json:"sshSSOPublicKey,omitempty"`
}

type DefaultAppsConfig struct {
	ClusterName  string `json:"clusterName,omitempty"`
	Organization string `json:"organization,omitempty"`
}

type GCP struct {
	Region         string   `json:"region,omitempty"`
	Project        string   `json:"project,omitempty"`
	FailureDomains []string `json:"failureDomains,omitempty"`
}

type Network struct {
	AutoCreateSubnetNetworks int    `json:"autoCreateSubnetworks,omitempty"`
	PodCIDR                  string `json:"podCidr,omitempty"`
}

type ControlPlane struct {
	InstanceType     string         `json:"instanceType,omitempty"`
	Replicas         int            `json:"replicas,omitempty"`
	RootVolume       Volume         `json:"rootVolume,omitempty"`
	EtcdVolume       Volume         `json:"etcdVolume,omitempty"`
	ContainerdVolume Volume         `json:"containerdVolume,omitempty"`
	KubeletVolume    Volume         `json:"kubeletVolume,omitempty"`
	ServiceAccount   ServiceAccount `json:"serviceAccount,omitempty"`
}

// Defines the size and disk type for a volume
type Volume struct {
	SizeGB   int    `json:"sizeGB,omitempty"`
	DiskType string `json:"diskType,omitempty"`
}

type ServiceAccount struct {
	Email  string   `json:"email,omitempty"`
	Scopes []string `json:"scopes,omitempty"`
}

type MachineDeployment struct {
	Name             string         `json:"name,omitempty"`
	FailureDomain    string         `json:"failureDomain,omitempty"`
	InstanceType     string         `json:"instanceType,omitempty"`
	Replicas         int            `json:"replicas,omitempty"`
	RootVolume       Volume         `json:"rootVolume,omitempty"`
	ContainerdVolume Volume         `json:"containerdVolume,omitempty"`
	KubeletVolume    Volume         `json:"kubeletVolume,omitempty"`
	CustomNodeLabels []string       `json:"customNodeLabels,omitempty"`
	ServiceAccount   ServiceAccount `json:"serviceAccount,omitempty"`
}
