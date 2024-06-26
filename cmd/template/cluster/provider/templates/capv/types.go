package capv

type ClusterConfig struct {
	Global *Global `json:"global,omitempty"`
}

type Global struct {
	Connectivity *Connectivity               `json:"connectivity,omitempty"`
	ControlPlane *ControlPlane               `json:"controlPlane,omitempty"`
	Metadata     *Metadata                   `json:"metadata,omitempty"`
	NodeClasses  map[string]*MachineTemplate `json:"nodeClasses,omitempty"`
	NodePools    map[string]*NodePool        `json:"nodePools,omitempty"`
}

type Metadata struct {
	Description  string `json:"description,omitempty"`
	Organization string `json:"organization,omitempty"`
}

type DefaultAppsConfig struct {
	ClusterName  string `json:"clusterName,omitempty"`
	Organization string `json:"organization,omitempty"`
}

type Connectivity struct {
	BaseDomain string   `json:"baseDomain,omitempty"`
	Network    *Network `json:"network,omitempty"`
}

type Network struct {
	ControlPlaneEndpoint *ControlPlaneEndpoint `json:"controlPlaneEndpoint,omitempty"`
	LoadBalancers        *LoadBalancers        `json:"loadBalancers,omitempty"`
}

type ControlPlaneEndpoint struct {
	Host       string `json:"host"`
	Port       int    `json:"port,omitempty"`
	IpPoolName string `json:"ipPoolName,omitempty"`
}

type LoadBalancers struct {
	CidrBlocks []string `json:"cidrBlocks,omitempty"`
	IpPoolName string   `json:"ipPoolName,omitempty"`
}

type ControlPlane struct {
	Image           *Image           `json:"image,omitempty"`
	Replicas        int              `json:"replicas,omitempty"`
	MachineTemplate *MachineTemplate `json:"machineTemplate,omitempty"`
}

type MTNetwork struct {
	Devices []*MTDevice `json:"devices,omitempty"`
}

type MTDevice struct {
	NetworkName string `json:"networkName,omitempty"`
	Dhcp4       bool   `json:"dhcp4,omitempty"`
}

type Image struct {
	Repository string `json:"repository,omitempty"`
}

type MachineTemplate struct {
	Network      *MTNetwork `json:"network,omitempty"`
	CloneMode    string     `json:"cloneMode,omitempty"`
	DiskGiB      int        `json:"diskGiB,omitempty"`
	NumCPUs      int        `json:"numCPUs,omitempty"`
	MemoryMiB    int        `json:"memoryMiB,omitempty"`
	ResourcePool string     `json:"resourcePool,omitempty"`
	Template     string     `json:"template,omitempty"`
}

type NodePool struct {
	Class    string `json:"class,omitempty"`
	Replicas int    `json:"replicas,omitempty"`
}
