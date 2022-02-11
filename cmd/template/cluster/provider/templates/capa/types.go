package capa

type ClusterConfig struct {
	ClusterDescription string         `json:"clusterDescription,omitempty"`
	Organization       string         `json:"organization,omitempty"`
	AWS                *AWS           `json:"aws,omitempty"`
	Network            *Network       `json:"network,omitempty"`
	Bastion            *Bastion       `json:"bastion,omitempty"`
	ControlPlane       *ControlPlane  `json:"controlPlane"`
	MachinePools       *[]MachinePool `json:"machinePools"`
	SSHSSOPublicKey    string         `json:"sshSSOPublicKey"`
	FlatcarAWSAccount  string         `json:"flatcarAWSAccount"`
}

type DefaultAppsConfig struct {
	ClusterName  string `json:"clusterName,omitempty"`
	Organization string `json:"organization,omitempty"`
}

type AWS struct {
	Region  string `json:"region,omitempty"`
	RoleARN string `json:"roleARN,omitempty"`
}

type Network struct {
	AvailabilityZoneUsageLimit int    `json:"availabilityZoneUsageLimit,omitempty"`
	VPCCIDR                    string `json:"vpcCIDR,omitempty"`
}

type Bastion struct {
	InstanceType string `json:"instanceType,omitempty"`
	Replicas     int    `json:"replicas,omitempty"`
}

type ControlPlane struct {
	InstanceType           string   `json:"instanceType,omitempty"`
	Replicas               int      `json:"replicas,omitempty"`
	RootVolumeSizeGB       int      `json:"rootVolumeSizeGB,omitempty"`
	EtcdVolumeSizeGB       int      `json:"etcdVolumeSizeGB,omitempty"`
	ContainerdVolumeSizeGB int      `json:"containerdVolumeSizeGB,omitempty"`
	KubeletVolumeSizeGB    int      `json:"kubeletVolumeSizeGB,omitempty"`
	AvailabilityZones      []string `json:"availabilityZones,omitempty"`
}

type MachinePool struct {
	Name              string   `json:"name"`
	AvailabilityZones []string `json:"availabilityZones,omitempty"`
	InstanceType      string   `json:"instanceType,omitempty"`
	MinSize           int      `json:"minSize"`
	MaxSize           int      `json:"maxSize"`
	RootVolumeSizeGB  int      `json:"rootVolumeSizeGB"`
	CustomNodeLabels  []string `json:"customNodeLabels"`
}
