package project

var (
	name    = "kubectl-gs"
	source  = "https://github.com/giantswarm/kubectl-gs"
	version = "5.7.2-dev"
)

func Name() string {
	return name
}

func Source() string {
	return source
}

func Version() string {
	return version
}
