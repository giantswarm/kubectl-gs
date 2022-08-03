package structure

// TBD: as you can see some of the fields are the same accorss
// different structs, so instead of keeping N types with duplicates
// it could be considered to add one big struct, and simply neglecting
// fields that are not neccessary for the structure being added.

type AppConfig struct {
	App                 string
	Base                string
	Catalog             string
	ManagementCluster   string
	Name                string
	Namespace           string
	Organization        string
	UserValuesConfigMap string
	UserValuesSecret    string
	WorkloadCluster     string
	Version             string
}

type AutomaticUpdateConfig struct {
	App               string
	ManagementCluster string
	Organization      string
	Repository        string
	WorkloadCluster   string
	VersionRepository string
}

type McConfig struct {
	Name            string
	RefreshInterval string
	RefreshTimeout  string
	RepositoryName  string
	ServiceAccount  string
}

type OrgConfig struct {
	ManagementCluster string
	Name              string
}

type WcConfig struct {
	Base                  string
	ClusterRelease        string
	ClusterUserConfig     string
	DefaultAppsRelease    string
	DefaultAppsUserConfig string
	ManagementCluster     string
	Name                  string
	Organization          string
	RepositoryName        string
}
