package key

const (
	ProviderAWS       = "aws"
	ProviderAzure     = "azure"
	ProviderCAPA      = "capa"
	ProviderGCP       = "gcp"
	ProviderKVM       = "kvm"
	ProviderOpenStack = "openstack"
	ProviderVSphere   = "vsphere"
)

// PureCAPIProviders is the list of all providers which are purely based on or fully migrated to CAPI
func PureCAPIProviders() []string {
	return []string{
		ProviderCAPA,
		ProviderGCP,
		ProviderVSphere,
		ProviderOpenStack,
	}
}
