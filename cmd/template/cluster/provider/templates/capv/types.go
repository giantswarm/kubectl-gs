package capv

type ClusterConfig struct {
	Connectivity       *Connectivity               `json:"connectivity,omitempty"`
	ControlPlane       *ControlPlane               `json:"controlPlane,omitempty"`
	NodeClasses        map[string]*MachineTemplate `json:"nodeClasses,omitempty"`
	NodePools          map[string]*NodePool        `json:"nodePools,omitempty"`
	HelmReleases       *HelmReleases               `json:"helmReleases,omitempty"`
	BaseDomain         string                      `json:"baseDomain,omitempty"`
	ClusterDescription string                      `json:"clusterDescription,omitempty"`
	Organization       string                      `json:"organization,omitempty"`
	Cluster            *Cluster                    `json:"cluster,omitempty"`
}

type Cluster struct {
	KubernetesVersion        string `json:"kubernetesVersion,omitempty"`
	EnableEncryptionProvider bool   `json:"enableEncryptionProvider,omitempty"`
}

type DefaultAppsConfig struct {
	ClusterName  string `json:"clusterName,omitempty"`
	Organization string `json:"organization,omitempty"`
}

type Connectivity struct {
	Network *Network `json:"network,omitempty"`
}

type Network struct {
	AllowAllEgress       bool                  `json:"allowAllEgress,omitempty"`
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

type HelmReleases struct {
	Cilium  *HelmRelease `json:"cilium,omitempty"`
	Cpi     *HelmRelease `json:"cpi,omitempty"`
	Coredns *HelmRelease `json:"coredns,omitempty"`
}

type HelmRelease struct {
	Interval string `json:"interval,omitempty"`
}
