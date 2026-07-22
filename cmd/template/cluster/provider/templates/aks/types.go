package aks

type ClusterConfig struct {
	Global *Global `json:"global,omitempty"`
}

type Global struct {
	ControlPlane      *ControlPlane     `json:"controlPlane,omitempty"`
	ManagementCluster string            `json:"managementCluster,omitempty"`
	Metadata          *Metadata         `json:"metadata,omitempty"`
	ProviderSpecific  *ProviderSpecific `json:"providerSpecific,omitempty"`
	Release           *Release          `json:"release,omitempty"`
}

type Metadata struct {
	Name            string            `json:"name,omitempty"`
	Description     string            `json:"description,omitempty"`
	Labels          map[string]string `json:"labels,omitempty"`
	Organization    string            `json:"organization,omitempty"`
	PreventDeletion bool              `json:"preventDeletion,omitempty"`
}

type ControlPlane struct {
	SKU *SKU `json:"sku,omitempty"`
}

type SKU struct {
	Tier string `json:"tier,omitempty"`
}

type ProviderSpecific struct {
	Location             string                `json:"location,omitempty"`
	SubscriptionID       string                `json:"subscriptionId,omitempty"`
	AzureClusterIdentity *AzureClusterIdentity `json:"azureClusterIdentity,omitempty"`
}

type AzureClusterIdentity struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

type Release struct {
	Version string `json:"version,omitempty"`
}
