package feature

const (
	NodePoolConditions = "nodepool-conditions"
)

var nodePoolConditions = Feature{
	ProviderAzure: Capability{
		MinVersion: "13.0.0",
	},
}
