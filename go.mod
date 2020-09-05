module github.com/giantswarm/kubectl-gs

go 1.14

require (
	github.com/Azure/azure-sdk-for-go v46.4.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.7 // indirect
	github.com/blang/semver v3.5.1+incompatible
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/fatih/color v1.9.0
	github.com/ghodss/yaml v1.0.0
	github.com/giantswarm/apiextensions/v2 v2.6.2
	github.com/giantswarm/app-operator/v2 v2.3.2
	github.com/giantswarm/k8sclient/v4 v4.0.0
	github.com/giantswarm/microerror v0.2.1
	github.com/giantswarm/micrologger v0.3.3
	github.com/google/go-cmp v0.5.2
	github.com/google/uuid v1.1.2 // indirect
	github.com/markbates/pkger v0.17.1
	github.com/mpvl/unique v0.0.0-20150818121801-cbe035fff7de
	github.com/onsi/ginkgo v1.14.1 // indirect
	github.com/onsi/gomega v1.10.2 // indirect
	github.com/pkg/errors v0.9.1
	github.com/qri-io/jsonschema v0.2.0
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966
	github.com/spf13/afero v1.4.1
	github.com/spf13/cobra v1.1.1
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sys v0.0.0-20200826173525-f9321e4c35a6 // indirect
	gopkg.in/yaml.v2 v2.3.0
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c
	k8s.io/api v0.18.9
	k8s.io/apiextensions-apiserver v0.18.9
	k8s.io/apimachinery v0.18.9
	k8s.io/cli-runtime v0.18.9
	k8s.io/client-go v0.18.9
	k8s.io/utils v0.0.0-20200912215256-4140de9c8800 // indirect
	sigs.k8s.io/cluster-api v0.3.10
	sigs.k8s.io/cluster-api-provider-azure v0.4.8
	sigs.k8s.io/controller-runtime v0.6.3
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/coreos/etcd => github.com/coreos/etcd v3.3.24+incompatible
	github.com/gorilla/websocket v1.4.0 => github.com/gorilla/websocket v1.4.2
	sigs.k8s.io/cluster-api v0.3.10 => github.com/giantswarm/cluster-api v0.3.10-gs
	sigs.k8s.io/cluster-api-provider-azure v0.4.9 => github.com/giantswarm/cluster-api-provider-azure v0.4.9-gsalpha2
)
