package gsrelease

type Release struct {
	Kind     string
	Metadata ReleaseMetadata
	Spec     ReleaseSpec
	Version  string
}

type ReleaseMetadata struct {
	Name        string
	Annotations map[string]string
}

type ReleaseSpec struct {
	Date       string
	Apps       []ReleaseSpecApp
	Components []ReleaseSpecComponent
	State      string
	Version    string
}

type ReleaseSpecApp struct {
	ComponentVersion string
	Name             string
	Version          string
}

type ReleaseSpecComponent struct {
	Name    string
	Version string
}
