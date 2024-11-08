package capvcd

import "github.com/giantswarm/kubectl-gs/v5/cmd/template/cluster/common"

type ClusterConfig struct {
	Global *Global `json:"global,omitempty"`
}

type Global struct {
	Connectivity *Connectivity        `json:"connectivity,omitempty"`
	ControlPlane *ControlPlane        `json:"controlPlane,omitempty"`
	Metadata     *Metadata            `json:"metadata,omitempty"`
	NodePools    map[string]*NodePool `json:"nodePools,omitempty"`
	Release      *Release             `json:"release,omitempty"`
}

type Connectivity struct {
	BaseDomain string   `json:"baseDomain,omitempty"`
	Network    *Network `json:"network,omitempty"`
	Proxy      *Proxy   `json:"proxy,omitempty"`
}

type Network struct {
	LoadBalancers     *LoadBalancers `json:"loadBalancers,omitempty"`
	ExtraOvdcNetworks []string       `json:"extraOvdcNetworks,omitempty"`
	StaticRoutes      []StaticRoute  `json:"staticRoutes,omitempty"`
}

type LoadBalancers struct {
	VipSubnet string `json:"vipSubnet,omitempty"`
}

type StaticRoute struct {
	Destination string `json:"destination,omitempty"`
	Via         string `json:"via,omitempty"`
}

type Proxy struct {
	Enabled    bool   `json:"enabled,omitempty"`
	HttpProxy  string `json:"httpProxy,omitempty"`
	HttpsProxy string `json:"httpsProxy,omitempty"`
	NoProxy    string `json:"noProxy,omitempty"`
}

type ControlPlane struct {
	Replicas        int              `json:"replicas,omitempty"`
	Oidc            *common.OIDC     `json:"oidc,omitempty"`
	MachineTemplate *MachineTemplate `json:"machineTemplate,omitempty"`
}

type MachineTemplate struct {
	Catalog      string `json:"catalog,omitempty"`
	DiskSizeGB   int    `json:"diskSizeGB,omitempty"`
	SizingPolicy string `json:"sizingPolicy,omitempty"`
}

type Metadata struct {
	Description     string `json:"description,omitempty"`
	Name            string `json:"name,omitempty"`
	Organization    string `json:"organization,omitempty"`
	PreventDeletion bool   `json:"preventDeletion,omitempty"`
}

type NodePool struct {
	Catalog      string `json:"catalog,omitempty"`
	Replicas     int    `json:"replicas,omitempty"`
	DiskSizeGB   int    `json:"diskSizeGB,omitempty"`
	SizingPolicy string `json:"sizingPolicy,omitempty"`
}

type ProviderSpecific struct {
	Org         string `json:"org,omitempty"`
	Ovdc        string `json:"ovdc,omitempty"`
	OvdcNetwork string `json:"ovdcNetwork,omitempty"`
	Site        string `json:"site,omitempty"`
}

type Release struct {
	Version string `json:"version,omitempty"`
}
