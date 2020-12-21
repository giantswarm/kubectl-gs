module github.com/giantswarm/kubectl-gs

go 1.14

require (
	github.com/blang/semver v3.5.1+incompatible
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/fatih/color v1.10.0
	github.com/ghodss/yaml v1.0.0
	github.com/giantswarm/apiextensions/v2 v2.6.2 // indirect
	github.com/giantswarm/apiextensions/v3 v3.6.0
	github.com/giantswarm/app/v3 v3.0.0-20201104142815-217bc60dc14d
	github.com/giantswarm/k8sclient/v5 v5.0.0
	github.com/giantswarm/microerror v0.2.1
	github.com/giantswarm/micrologger v0.3.4
	github.com/google/go-cmp v0.5.3
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/markbates/pkger v0.17.1
	github.com/mpvl/unique v0.0.0-20150818121801-cbe035fff7de
	github.com/pkg/errors v0.9.1
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966
	github.com/spf13/afero v1.4.1
	github.com/spf13/cobra v1.1.1
	github.com/xeipuuv/gojsonschema v1.2.0
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c
	k8s.io/api v0.18.9
	k8s.io/apiextensions-apiserver v0.18.9
	k8s.io/apimachinery v0.18.9
	k8s.io/cli-runtime v0.18.9
	k8s.io/client-go v0.18.9
	sigs.k8s.io/cluster-api v0.3.10
	sigs.k8s.io/cluster-api-provider-azure v0.4.9
	sigs.k8s.io/controller-runtime v0.6.3
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/Microsoft/hcsshim v0.8.7 => github.com/Microsoft/hcsshim v0.8.10
	github.com/coreos/etcd => github.com/coreos/etcd v3.3.24+incompatible
	// Use moby v20.10.0-beta1 to fix build issue on darwin.
	github.com/docker/docker => github.com/moby/moby v20.10.0-beta1+incompatible
	github.com/gorilla/websocket v1.4.0 => github.com/gorilla/websocket v1.4.2
	github.com/opencontainers/runc v0.1.1 => github.com/opencontainers/runc v1.0.0-rc7
	k8s.io/kubernetes v1.13.0 => k8s.io/kubernetes v1.16.13
	sigs.k8s.io/cluster-api v0.3.10 => github.com/giantswarm/cluster-api v0.3.10-gs
	sigs.k8s.io/cluster-api-provider-azure v0.4.9 => github.com/giantswarm/cluster-api-provider-azure v0.4.9-gsalpha2
)
