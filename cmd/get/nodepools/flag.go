package nodepools

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	flagAllNamespaces = "all-namespaces"
	flagClusterName   = "cluster-name"
)

type flag struct {
	AllNamespaces bool
	ClusterName   string

	print *genericclioptions.PrintFlags
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&f.AllNamespaces, flagAllNamespaces, "A", false, "If present, list the requested object(s) across all namespaces. Namespace in current context is ignored even if specified with --namespace.")
	cmd.Flags().StringVarP(&f.ClusterName, flagClusterName, "c", "", "Only show node pools of the cluster with this name")

	f.print = genericclioptions.NewPrintFlags("")

	// Merging current command flags and config flags,
	// to be able to override kubectl-specific ones.
	f.print.AddFlags(cmd)
}

func (f *flag) Validate() error {
	return nil
}
