package feature

// Capability contains a provider's requirements
// for a feature to be supported.
type Capability struct {
	MinVersion string
}

// Feature is a collection of different requirements
// for supporting a functionality on different providers.
type Feature map[string]Capability
