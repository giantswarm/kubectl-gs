package structure

type AppConfig struct {
	App                 string
	Base                string
	Catalog             string
	Name                string
	Namespace           string
	Organization        string
	UserValuesConfigMap string
	UserValuesSecret    string
	WorkloadCluster     string
	Version             string
}

type McConfig struct {
	Name            string
	RefreshInterval string
	RefreshTimeout  string
	RepositoryName  string
	ServiceAccount  string
}

type OrgConfig struct {
	Name string
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
