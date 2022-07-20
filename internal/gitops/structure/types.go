package structure

type McConfig struct {
	Name            string
	RefreshInterval string
	RefreshTimeout  string
	RepositoryName  string
	ServiceAccount  string
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

type OrgConfig struct {
	Name string
}
