package cluster

import (
	"io"
	"os"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/giantswarm/kubectl-gs/v6/cmd/template/cluster/flags"
	"github.com/giantswarm/kubectl-gs/v6/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/v6/pkg/middleware"
	"github.com/giantswarm/kubectl-gs/v6/pkg/middleware/renewtoken"
)

const (
	name             = "cluster"
	shortDescription = "Template Giant Swarm clusters."
	longDescription  = `Template Giant Swarm clusters.

The output is a values.yaml plus the wrapping resources for the
cluster app. It is a starting template — fields not exposed as flags
(extra node pools, custom taints, machine pool architecture, etc.)
can be set by editing the generated values.yaml before applying it.

See https://docs.giantswarm.io/getting-started/create-workload-cluster/`

	example = `
    Template an AWS cluster (writes to stdout):

    kubectl-gs template cluster --provider capa --name my-cluster --organization acme --release 35.0.0 --region eu-west-1

    Template an AWS cluster and write to a file for further editing:

    kubectl-gs template cluster --provider capa --name my-cluster --organization acme --release 35.0.0 --region eu-west-1 --output my-cluster.yaml

    Template a proxy-private AWS cluster:

    kubectl-gs template cluster --provider capa --name my-cluster --organization acme --release 35.0.0 --region eu-west-1 --cluster-type proxy-private --https-proxy https://proxy.example.com:3128 --vpc-cidr 10.0.0.0/16

    Add an arm64 node pool to an AWS cluster:

    After running the command above, edit the generated values.yaml and
    add a second pool under global.nodePools. The architecture, instance
    type family, and taint must match. See the docs for supported releases:
    https://docs.giantswarm.io/getting-started/create-workload-cluster/

      global:
        nodePools:
          worker-amd:
            instanceType: m5.xlarge
            minSize: 1
            maxSize: 3
          worker-arm:
            instanceType: m7g.xlarge
            architecture: arm64
            minSize: 1
            maxSize: 3
            customNodeTaints:
            - key: kubernetes.io/arch
              value: arm64
              effect: NoSchedule
`
)

type Config struct {
	Logger      micrologger.Logger
	ConfigFlags *genericclioptions.RESTClientGetter

	Stderr io.Writer
	Stdout io.Writer
}

func New(config Config) (*cobra.Command, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.Stderr == nil {
		config.Stderr = os.Stderr
	}
	if config.Stdout == nil {
		config.Stdout = os.Stdout
	}
	if config.ConfigFlags == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.ConfigFlags must not be empty", config)
	}

	f := &flags.Flag{}

	r := &runner{
		commonConfig: &commonconfig.CommonConfig{
			ConfigFlags: config.ConfigFlags,
		},
		flag:   f,
		logger: config.Logger,
		stderr: config.Stderr,
		stdout: config.Stdout,
	}

	c := &cobra.Command{
		Use:     name,
		Short:   shortDescription,
		Long:    longDescription,
		Example: example,
		Args:    cobra.NoArgs,
		RunE:    r.Run,
		PreRunE: middleware.Compose(
			renewtoken.Middleware(*config.ConfigFlags),
		),
	}

	f.Init(c)

	return c, nil
}
