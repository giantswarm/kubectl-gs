package feature

const (
	ClientCert = "client-cert"
)

var clientCert = Feature{
	ProviderAzure: Capability{
		MinVersion: "13.0.0",
	},
	ProviderAWS: Capability{
		MinVersion: "16.0.1",
	},
	ProviderOpenStack: Capability{
		MinVersion: "20.0.0-alpha1",
	},
}
