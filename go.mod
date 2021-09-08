module github.com/giantswarm/kubectl-gs

go 1.16

require (
	github.com/blang/semver/v4 v4.0.0
	github.com/coreos/go-oidc/v3 v3.0.0
	github.com/fatih/color v1.12.0
	github.com/giantswarm/apiextensions/v3 v3.32.1-0.20210908214419-4b97f7f45cff
	github.com/giantswarm/app/v5 v5.2.3
	github.com/giantswarm/k8sclient/v5 v5.12.0
	github.com/giantswarm/microerror v0.3.0
	github.com/giantswarm/micrologger v0.5.0
	github.com/google/go-cmp v0.5.6
	github.com/pkg/errors v0.9.1
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966
	github.com/spf13/afero v1.6.0
	github.com/spf13/cobra v1.2.1
	github.com/xeipuuv/gojsonschema v1.2.0
	golang.org/x/oauth2 v0.0.0-20210819190943-2bc19b11175f
	k8s.io/api v0.21.3
	k8s.io/apiextensions-apiserver v0.21.3
	k8s.io/apimachinery v0.21.3
	k8s.io/cli-runtime v0.21.2
	k8s.io/client-go v0.21.3
	sigs.k8s.io/cluster-api v0.4.2
	sigs.k8s.io/cluster-api-provider-aws v0.6.4
	sigs.k8s.io/cluster-api-provider-azure v0.4.11
	sigs.k8s.io/cluster-api-provider-vsphere v0.8.1
	sigs.k8s.io/controller-runtime v0.9.6
	sigs.k8s.io/yaml v1.2.0
)

replace (
	sigs.k8s.io/cluster-api v0.0.0-00010101000000-000000000000 => sigs.k8s.io/cluster-api v0.4.2
	sigs.k8s.io/cluster-api-provider-aws v0.6.4 => github.com/giantswarm/cluster-api-provider-aws v0.6.5-gs1
	sigs.k8s.io/cluster-api-provider-azure v0.4.11 => github.com/giantswarm/cluster-api-provider-azure v0.4.12-gsalpha3
)
