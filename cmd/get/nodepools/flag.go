package nodepools

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	flagAllNamespaces       = "all-namespaces"
	flagClusterIDDeprecated = "cluster-id"
	flagClusterName         = "cluster-name"
)

type flag struct {
	AllNamespaces       bool
	ClusterIDDeprecated string
	ClusterName         string

	config genericclioptions.RESTClientGetter
	print  *genericclioptions.PrintFlags
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&f.AllNamespaces, flagAllNamespaces, "A", false, "If present, list the requested object(s) across all namespaces. Namespace in current context is ignored even if specified with --namespace.")
	cmd.Flags().StringVarP(&f.ClusterIDDeprecated, flagClusterIDDeprecated, "", "", "Set this to a cluster name to only show this cluster's node pools")
	cmd.Flags().StringVarP(&f.ClusterName, flagClusterName, "c", "", "Only show node pools of the cluster with this name")

	// TODO: remove by ~ December 2021
	_ = cmd.Flags().MarkDeprecated(flagClusterIDDeprecated, "use --cluster-name instead")

	f.config = genericclioptions.NewConfigFlags(true)
	f.print = genericclioptions.NewPrintFlags("")

	// Merging current command flags and config flags,
	// to be able to override kubectl-specific ones.
	f.config.(*genericclioptions.ConfigFlags).AddFlags(cmd.Flags())
	f.print.AddFlags(cmd)
}

func (f *flag) Validate() error {
	// Apply --clsuter-id value to --cluster-name
	// TODO: remove by ~ December 2021
	if f.ClusterIDDeprecated != "" && f.ClusterName == "" {
		f.ClusterName = f.ClusterIDDeprecated
	}

	return nil
}
