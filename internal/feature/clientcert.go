package feature

const (
	ClientCert = "client-cert"
)

var clientCert = Feature{
	ProviderAzure: Capability{
		MinVersion: "13.0.0",
	},
	ProviderAWS: Capability{
		MinVersion: "13.0.0",
	},
}
