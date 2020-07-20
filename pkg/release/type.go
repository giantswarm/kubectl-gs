package release

type ReleaseObject struct {
	Kind     string
	Metadata ReleaseObjectMetadata
	Spec     ReleaseObjectSpec
	Version  string
}

type ReleaseObjectMetadata struct {
	Name        string
	Annotations map[string]string
}

type ReleaseObjectSpec struct {
	Date       string
	Apps       []ReleaseObjectSpecApp
	Components []ReleaseObjectSpecComponent
	State      string
	Version    string
}

type ReleaseObjectSpecApp struct {
	ComponentVersion string
	Name             string
	Version          string
}

type ReleaseObjectSpecComponent struct {
	Name    string
	Version string
}
