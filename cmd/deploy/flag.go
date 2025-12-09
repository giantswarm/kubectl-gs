package deploy

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	flagDeploy         = "deploy"
	flagUndeploy       = "undeploy"
	flagStatus         = "status"
	flagList           = "list"
	flagNamespace      = "namespace"
	flagType           = "type"
	flagCatalog        = "catalog"
	flagInteractive    = "interactive"
	flagUndeployOnExit = "undeploy-on-exit"
	flagSync           = "sync"
	flagInstalledOnly  = "installed-only"

	// Resource types
	resourceTypeApp    = "app"
	resourceTypeConfig = "config"

	// List types
	listTypeApps     = "apps"
	listTypeVersions = "versions"
	listTypeConfigs  = "configs"
	listTypeCatalogs = "catalogs"

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
	List     string

	// Option flags
	Namespace      string
	Type           string
	Catalog        string
	Interactive    bool
	UndeployOnExit bool
	Sync           bool
	InstalledOnly  bool

	// Print flags
	print *genericclioptions.PrintFlags
}

func (f *flag) Init(cmd *cobra.Command) {
	// Action flags
	cmd.Flags().BoolVarP(&f.Deploy, flagDeploy, "d", false, "Deploy a resource onto a cluster")
	cmd.Flags().BoolVarP(&f.Undeploy, flagUndeploy, "u", false, "Undeploy a resource from a cluster")
	cmd.Flags().BoolVar(&f.Status, flagStatus, false, "Show status of all kustomizations, config repositories, and apps of the cluster")
	cmd.Flags().StringVarP(&f.List, flagList, "l", "", "List resources. Valid values: apps, versions, configs, catalogs")

	// Option flags
	cmd.Flags().StringVarP(&f.Namespace, flagNamespace, "n", "", "Namespace where the resource lives (default for app: giantswarm, default for config: flux-giantswarm)")
	cmd.Flags().StringVarP(&f.Type, flagType, "t", resourceTypeApp, "Resource type to handle either 'app' or 'config'")
	cmd.Flags().StringVarP(&f.Catalog, flagCatalog, "c", defaultCatalog, "Catalog to use for the app deployment (only for app type)")
	cmd.Flags().BoolVarP(&f.Interactive, flagInteractive, "i", false, "Interactive mode: select app and version interactively from catalog entries")
	cmd.Flags().BoolVarP(&f.UndeployOnExit, flagUndeployOnExit, "r", false, "Wait for interrupt signal and undeploy on exit")
	cmd.Flags().BoolVar(&f.Sync, flagSync, false, "Force synchronous deployment by triggering flux reconciliation")
	cmd.Flags().BoolVar(&f.InstalledOnly, flagInstalledOnly, false, "When listing apps, show only installed apps (default: show all catalog apps with installation status)")

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
	if f.List != "" {
		actionCount++
	}

	if actionCount == 0 {
		return fmt.Errorf("%w: must specify one action: -d (deploy), -u (undeploy), -s (status), or -l (list)", ErrInvalidFlag)
	}
	if actionCount > 1 {
		return fmt.Errorf("%w: can only specify one action at a time", ErrInvalidFlag)
	}

	// Validate list type if specified
	if f.List != "" {
		validListTypes := []string{listTypeApps, listTypeVersions, listTypeConfigs, listTypeCatalogs}
		if !slices.Contains(validListTypes, f.List) {
			return fmt.Errorf("%w: --%s must be one of: %s", ErrInvalidFlag, flagList, strings.Join(validListTypes, ", "))
		}
	}

	// Validate resource type
	validTypes := []string{resourceTypeApp, resourceTypeConfig}
	if !slices.Contains(validTypes, f.Type) {
		return fmt.Errorf("%w: --%s must be one of: %s", ErrInvalidFlag, flagType, strings.Join(validTypes, ", "))
	}

	// Validate interactive flag
	if f.Interactive {
		if !f.Deploy {
			return fmt.Errorf("%w: --%s can only be used with --%s action", ErrInvalidFlag, flagInteractive, flagDeploy)
		}
		if f.Type != resourceTypeApp {
			return fmt.Errorf("%w: --%s is only supported for app deployments", ErrInvalidFlag, flagInteractive)
		}
	}

	// Validate undeploy-on-exit flag
	if f.UndeployOnExit {
		if !f.Deploy {
			return fmt.Errorf("%w: --%s can only be used with --%s action", ErrInvalidFlag, flagUndeployOnExit, flagDeploy)
		}
	}

	// Validate sync flag
	if f.Sync {
		if !f.Deploy {
			return fmt.Errorf("%w: --%s can only be used with --%s action", ErrInvalidFlag, flagSync, flagDeploy)
		}
	}

	// Set default namespace based on resource type or list type if not specified
	if f.Namespace == "" {
		// If listing configs, use config namespace
		if f.List == listTypeConfigs {
			f.Namespace = defaultConfigNamespace
		} else if f.Type == resourceTypeApp {
			f.Namespace = defaultAppNamespace
		} else if f.Type == resourceTypeConfig {
			f.Namespace = defaultConfigNamespace
		}
	}

	return nil
}

func (f *flag) GetAction() string {
	switch {
	case f.Deploy:
		return "deploy"
	case f.Undeploy:
		return "undeploy"
	case f.Status:
		return "status"
	case f.List != "":
		return "list"
	default:
		return ""
	}
}
