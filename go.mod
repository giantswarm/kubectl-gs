module github.com/giantswarm/kubectl-gs

go 1.13

require (
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/fatih/color v1.9.0
	github.com/ghodss/yaml v1.0.0
	github.com/giantswarm/apiextensions v0.4.16
	github.com/giantswarm/microerror v0.2.0
	github.com/giantswarm/micrologger v0.3.1
	github.com/google/go-cmp v0.5.0
	github.com/markbates/pkger v0.17.0
	github.com/mpvl/unique v0.0.0-20150818121801-cbe035fff7de
	github.com/pkg/errors v0.9.1
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966
	github.com/spf13/afero v1.3.1
	github.com/spf13/cobra v0.0.7
	github.com/stretchr/testify v1.5.1 // indirect
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	gopkg.in/yaml.v3 v3.0.0-20200121175148-a6ecf24a6d71
	k8s.io/api v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v0.17.2
)

replace (
	k8s.io/api v0.0.0 => k8s.io/api v0.16.6
	k8s.io/apiextensions-apiserver v0.0.0 => k8s.io/apiextensions-apiserver v0.16.6
	k8s.io/apimachinery v0.0.0 => k8s.io/apimachinery v0.16.6
	k8s.io/apiserver v0.0.0 => k8s.io/apiserver v0.16.6
	k8s.io/cli-runtime v0.0.0 => k8s.io/cli-runtime v0.16.6
	k8s.io/client-go v0.0.0 => k8s.io/client-go v0.16.6
	k8s.io/cloud-provider v0.0.0 => k8s.io/cloud-provider v0.16.6
	k8s.io/cluster-bootstrap v0.0.0 => k8s.io/cluster-bootstrap v0.16.6
	k8s.io/code-generator v0.0.0 => k8s.io/code-generator v0.16.6
	k8s.io/component-base v0.0.0 => k8s.io/component-base v0.16.6
	k8s.io/cri-api v0.0.0 => k8s.io/cri-api v0.16.6
	k8s.io/csi-translation-lib v0.0.0 => k8s.io/csi-translation-lib v0.16.6
	k8s.io/kube-aggregator v0.0.0 => k8s.io/kube-aggregator v0.16.6
	k8s.io/kube-controller-manager v0.0.0 => k8s.io/kube-controller-manager v0.16.6
	k8s.io/kube-proxy v0.0.0 => k8s.io/kube-proxy v0.16.6
	k8s.io/kube-scheduler v0.0.0 => k8s.io/kube-scheduler v0.16.6
	k8s.io/kubectl v0.0.0 => k8s.io/kubectl v0.16.6
	k8s.io/kubelet v0.0.0 => k8s.io/kubelet v0.16.6
	k8s.io/legacy-cloud-providers v0.0.0 => k8s.io/legacy-cloud-providers v0.16.6
	k8s.io/metrics v0.0.0 => k8s.io/metrics v0.16.6
	k8s.io/sample-apiserver v0.0.0 => k8s.io/sample-apiserver v0.16.6

)
