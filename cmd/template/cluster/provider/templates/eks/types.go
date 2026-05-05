package eks

type Global struct {
	Metadata *Metadata `json:"metadata,omitempty"`
	Release  *Release  `json:"release,omitempty"`
}

type ClusterConfig struct {
	Global *Global `json:"global,omitempty"`
}

type Metadata struct {
	Name            string            `json:"name,omitempty"`
	Description     string            `json:"description,omitempty"`
	Labels          map[string]string `json:"labels,omitempty"`
	Organization    string            `json:"organization,omitempty"`
	PreventDeletion bool              `json:"preventDeletion,omitempty"`
}

type Release struct {
	Version string `json:"version,omitempty"`
}
