package capa

type ClusterConfig struct {
	Connectivity     *Connectivity           `json:"connectivity,omitempty"`
	ControlPlane     *ControlPlane           `json:"controlPlane,omitempty"`
	Metadata         *Metadata               `json:"metadata,omitempty"`
	NodePools        *map[string]MachinePool `json:"nodePools,omitempty"`
	ProviderSpecific *ProviderSpecific       `json:"providerSpecific,omitempty"`
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
	AMI                        string `json:"ami,omitempty"`
	AWSClusterRoleIdentityName string `json:"awsClusterRoleIdentityName,omitempty"`
	FlatcarAWSAccount          string `json:"flatcarAwsAccount,omitempty"`
	Region                     string `json:"region,omitempty"`
}

type Connectivity struct {
	AvailabilityZoneUsageLimit int       `json:"availabilityZoneUsageLimit,omitempty"`
	Bastion                    *Bastion  `json:"bastion,omitempty"`
	Network                    *Network  `json:"network,omitempty"`
	Proxy                      *Proxy    `json:"proxy,omitempty"`
	Subnets                    []Subnet  `json:"subnets,omitempty"`
	Topology                   *Topology `json:"topology,omitempty"`
	VPCMode                    string    `json:"vpcMode,omitempty"`
}

type Network struct {
	VPCCIDR string `json:"vpcCidr,omitempty"`
}

type Topology struct {
	Mode             string `json:"mode,omitempty"`
	PrefixListID     string `json:"prefixListId,omitempty"`
	TransitGatewayID string `json:"transitGatewayId,omitempty"`
}
type Subnet struct {
	CidrBlocks []CIDRBlock `json:"cidrBlocks"`
	IsPublic   bool        `json:"isPublic"`
}

type CIDRBlock struct {
	CIDR             string `json:"cidr"`
	AvailabilityZone string `json:"availabilityZone"`
}

type Bastion struct {
	Enabled      bool   `json:"enabled,omitempty"`
	InstanceType string `json:"instanceType,omitempty"`
	Replicas     int    `json:"replicas,omitempty"`
}

type ControlPlane struct {
	APIMode                string   `json:"apiMode,omitempty"`
	InstanceType           string   `json:"instanceType,omitempty"`
	Replicas               int      `json:"replicas,omitempty"`
	RootVolumeSizeGB       int      `json:"rootVolumeSizeGB,omitempty"`
	EtcdVolumeSizeGB       int      `json:"etcdVolumeSizeGB,omitempty"`
	ContainerdVolumeSizeGB int      `json:"containerdVolumeSizeGB,omitempty"`
	KubeletVolumeSizeGB    int      `json:"kubeletVolumeSizeGB,omitempty"`
	AvailabilityZones      []string `json:"availabilityZones,omitempty"`
}

type MachinePool struct {
	Name              string   `json:"name,omitempty"`
	AvailabilityZones []string `json:"availabilityZones,omitempty"`
	InstanceType      string   `json:"instanceType,omitempty"`
	MinSize           int      `json:"minSize,omitempty"`
	MaxSize           int      `json:"maxSize,omitempty"`
	RootVolumeSizeGB  int      `json:"rootVolumeSizeGB,omitempty"`
	CustomNodeLabels  []string `json:"customNodeLabels,omitempty"`
}

type Proxy struct {
	Enabled    bool   `json:"enabled,omitempty"`
	HttpsProxy string `json:"httpsProxy,omitempty"`
	HttpProxy  string `json:"httpProxy,omitempty"`
	NoProxy    string `json:"noProxy,omitempty"`
}
