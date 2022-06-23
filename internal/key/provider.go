package key

const (
	ProviderAWS       = "aws"
	ProviderAzure     = "azure"
	ProviderGCP       = "gcp"
	ProviderKVM       = "kvm"
	ProviderOpenStack = "openstack"
	ProviderVSphere   = "vsphere"
)

// PureCAPIProviders is the list of all providers which are purely based on or fully migrated to CAPI
func PureCAPIProviders() []string {
	return []string{
		ProviderGCP,
		ProviderVSphere,
		ProviderOpenStack,
	}
}
