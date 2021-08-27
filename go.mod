module github.com/giantswarm/kubectl-gs

go 1.16

require (
	github.com/blang/semver v3.5.1+incompatible
	github.com/blang/semver/v4 v4.0.0
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/fatih/color v1.12.0
	github.com/giantswarm/apiextensions/v3 v3.32.0
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
	k8s.io/api v0.18.19
	k8s.io/apiextensions-apiserver v0.18.19
	k8s.io/apimachinery v0.18.19
	k8s.io/cli-runtime v0.18.19
	k8s.io/client-go v0.18.19
	sigs.k8s.io/cluster-api v0.3.15-0.20210309173700-34de71aaaac8
	sigs.k8s.io/cluster-api-provider-aws v0.6.4
	sigs.k8s.io/cluster-api-provider-azure v0.4.11
	sigs.k8s.io/controller-runtime v0.6.4
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/Microsoft/hcsshim v0.8.7 => github.com/Microsoft/hcsshim v0.8.10
	github.com/coreos/etcd => github.com/etcd-io/etcd v3.3.25+incompatible
	github.com/dgrijalva/jwt-go => github.com/dgrijalva/jwt-go/v4 v4.0.0-preview1
	github.com/docker/docker => github.com/moby/moby v20.10.6+incompatible // Use moby v20.10.x to fix build issue on darwin.
	github.com/gogo/protobuf => github.com/gogo/protobuf v1.3.2 // [CVE-2021-3121]
	github.com/gorilla/websocket v1.4.0 => github.com/gorilla/websocket v1.4.2
	github.com/opencontainers/runc v0.1.1 => github.com/opencontainers/runc v1.0.0-rc7
	k8s.io/client-go => k8s.io/client-go v0.18.18
	sigs.k8s.io/cluster-api v0.3.15-0.20210309173700-34de71aaaac8 => github.com/giantswarm/cluster-api v0.3.13-gs
	sigs.k8s.io/cluster-api-provider-aws v0.6.4 => github.com/giantswarm/cluster-api-provider-aws v0.6.5-gs1
	sigs.k8s.io/cluster-api-provider-azure v0.4.11 => github.com/giantswarm/cluster-api-provider-azure v0.4.12-gsalpha3
)
