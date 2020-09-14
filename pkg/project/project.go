package project

var (
	description = "Kubectl plugin to render CRs for Giant Swarm tenant clusters."
	gitSHA      = "n/a"
	name        = "kubectl-gs"
	source      = "https://github.com/giantswarm/kubectl-gs"
	version     = "0.6.1"
)

func Description() string {
	return description
}

func GitSHA() string {
	return gitSHA
}

func Name() string {
	return name
}

func Source() string {
	return source
}

func Version() string {
	return version
}
