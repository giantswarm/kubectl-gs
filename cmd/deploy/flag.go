package deploy

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	flagDeploy    = "deploy"
	flagUndeploy  = "undeploy"
	flagStatus    = "status"
	flagNamespace = "namespace"
	flagType      = "type"
	flagCatalog   = "catalog"

	// Resource types
	resourceTypeApp    = "app"
	resourceTypeConfig = "config"

	// Default values
	defaultAppNamespace    = "giantswarm"
	defaultConfigNamespace = "flux-giantswarm"
	defaultCatalog         = "control-plane-test-catalog"
)

type flag struct {
	// Action flags
	Deploy   bool
	Undeploy bool
	Status   bool

	// Option flags
	Namespace string
	Type      string
	Catalog   string

	// Print flags
	print *genericclioptions.PrintFlags
}

func (f *flag) Init(cmd *cobra.Command) {
	// Action flags
	cmd.Flags().BoolVarP(&f.Deploy, flagDeploy, "d", false, "Deploy a resource onto a cluster")
	cmd.Flags().BoolVarP(&f.Undeploy, flagUndeploy, "u", false, "Undeploy a resource from a cluster")
	cmd.Flags().BoolVar(&f.Status, flagStatus, false, "Show status of all kustomizations, config repositories, and apps of the cluster")

	// Option flags
	cmd.Flags().StringVarP(&f.Namespace, flagNamespace, "n", "", "Namespace where the resource lives (default for app: giantswarm, default for config: flux-giantswarm)")
	cmd.Flags().StringVarP(&f.Type, flagType, "t", resourceTypeApp, "Resource type to handle either 'app' or 'config'")
	cmd.Flags().StringVarP(&f.Catalog, flagCatalog, "c", defaultCatalog, "Catalog to use for the app deployment (only for app type)")

	// Print flags for output formatting
	f.print = genericclioptions.NewPrintFlags("")
	f.print.AddFlags(cmd)
}

func (f *flag) Validate() error {
	// Validate that exactly one action is specified
	actionCount := 0
	if f.Deploy {
		actionCount++
	}
	if f.Undeploy {
		actionCount++
	}
	if f.Status {
		actionCount++
	}

	if actionCount == 0 {
		return fmt.Errorf("%w: must specify one action: -d (deploy), -u (undeploy), or -s (status)", ErrInvalidFlag)
	}
	if actionCount > 1 {
		return fmt.Errorf("%w: can only specify one action at a time", ErrInvalidFlag)
	}

	// Validate resource type
	validTypes := []string{resourceTypeApp, resourceTypeConfig}
	if !slices.Contains(validTypes, f.Type) {
		return fmt.Errorf("%w: --%s must be one of: %s", ErrInvalidFlag, flagType, strings.Join(validTypes, ", "))
	}

	// Set default namespace based on resource type if not specified
	if f.Namespace == "" {
		if f.Type == resourceTypeApp {
			f.Namespace = defaultAppNamespace
		} else if f.Type == resourceTypeConfig {
			f.Namespace = defaultConfigNamespace
		}
	}

	return nil
}

func (f *flag) GetAction() string {
	if f.Deploy {
		return "deploy"
	}
	if f.Undeploy {
		return "undeploy"
	}
	if f.Status {
		return "status"
	}
	return ""
}
