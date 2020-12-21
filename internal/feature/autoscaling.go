package feature

const (
	Autoscaling = "autoscaling"
)

var autoscaling = Feature{
	ProviderAWS: Capability{
		MinVersion: "10.0.0",
	},
	ProviderAzure: Capability{
		MinVersion: "13.1.0",
	},
}
