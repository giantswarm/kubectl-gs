module github.com/giantswarm/kubectl-gs

go 1.14

require (
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/fatih/color v1.9.0
	github.com/ghodss/yaml v1.0.0
	github.com/giantswarm/apiextensions v0.4.20
	github.com/giantswarm/k8sclient/v3 v3.1.2
	github.com/giantswarm/microerror v0.2.1
	github.com/giantswarm/micrologger v0.3.1
	github.com/google/go-cmp v0.5.1
	github.com/markbates/pkger v0.17.0
	github.com/mpvl/unique v0.0.0-20150818121801-cbe035fff7de
	github.com/pkg/errors v0.9.1
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966
	github.com/spf13/afero v1.3.2
	github.com/spf13/cobra v0.0.7
	github.com/stretchr/testify v1.5.1 // indirect
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	gopkg.in/yaml.v3 v3.0.0-20200121175148-a6ecf24a6d71
	k8s.io/api v0.17.8
	k8s.io/apiextensions-apiserver v0.17.8
	k8s.io/apimachinery v0.17.8
	k8s.io/cli-runtime v0.17.2
	k8s.io/client-go v0.17.8
	sigs.k8s.io/cluster-api v0.3.7
	sigs.k8s.io/controller-runtime v0.5.8
)
