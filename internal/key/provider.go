package key

const (
	ProviderAWS           = "aws"
	ProviderAzure         = "azure"
	ProviderCAPA          = "capa"
	ProviderCAPZ          = "capz"
	ProviderGCP           = "gcp"
	ProviderKVM           = "kvm"
	ProviderOpenStack     = "openstack"
	ProviderVSphere       = "vsphere"
	ProviderCloudDirector = "cloud-director"
)

// PureCAPIProviders is the list of all providers which are purely based on or fully migrated to CAPI
func PureCAPIProviders() []string {
	return []string{
		ProviderCAPA,
		ProviderCAPZ,
		ProviderGCP,
		ProviderVSphere,
		ProviderOpenStack,
		ProviderCloudDirector,
	}
}
