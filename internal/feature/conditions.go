package feature

const (
	Conditions = "conditions"
)

var conditions = Feature{
	ProviderAzure: Capability{
		MinVersion: "13.0.0",
	},
}
