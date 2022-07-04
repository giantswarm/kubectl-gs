package structure

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
