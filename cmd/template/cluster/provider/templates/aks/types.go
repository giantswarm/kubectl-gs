package aks

type ClusterConfig struct {
	Global *Global `json:"global,omitempty"`
}

type Global struct {
	ControlPlane     *ControlPlane     `json:"controlPlane,omitempty"`
	Metadata         *Metadata         `json:"metadata,omitempty"`
	ProviderSpecific *ProviderSpecific `json:"providerSpecific,omitempty"`
	Release          *Release          `json:"release,omitempty"`
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
	Location          string             `json:"location,omitempty"`
	SubscriptionID    string             `json:"subscriptionId,omitempty"`
	ASOAuthentication *ASOAuthentication `json:"asoAuthentication,omitempty"`
}

type ASOAuthentication struct {
	SubscriptionID string `json:"subscriptionID,omitempty"`
	TenantID       string `json:"tenantID,omitempty"`
	ClientID       string `json:"clientID,omitempty"`
}

type Release struct {
	Version string `json:"version,omitempty"`
}
