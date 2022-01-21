package app

import (
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/pkg/annotations"
	"github.com/giantswarm/kubectl-gs/pkg/labels"
)

const (
	flagAppName                    = "app-name"
	flagCatalog                    = "catalog"
	flagCluster                    = "cluster"
	flagDefaultingEnabled          = "defaulting-enabled"
	flagInCluster                  = "in-cluster"
	flagName                       = "name"
	flagNamespace                  = "namespace"
	flagNamespaceConfigAnnotations = "namespace-annotations"
	flagNamespaceConfigLabels      = "namespace-labels"
	flagOrganization               = "organization"
	flagUserConfigMap              = "user-configmap"
	flagUserSecret                 = "user-secret"
	flagVersion                    = "version"
)

type flag struct {
	AppName                        string
	Catalog                        string
	Cluster                        string
	DefaultingEnabled              bool
	InCluster                      bool
	Name                           string
	Namespace                      string
	Organization                   string
	Version                        string
	flagNamespaceConfigAnnotations []string
	flagNamespaceConfigLabels      []string
	flagUserConfigMap              string
	flagUserSecret                 string
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.AppName, flagAppName, "", "Optionally set a different name for the App CR.")
	cmd.Flags().StringVar(&f.Catalog, flagCatalog, "", "Catalog name where app is stored.")
	cmd.Flags().StringVar(&f.Name, flagName, "", "Name of the app in the Catalog.")
	cmd.Flags().StringVar(&f.Namespace, flagNamespace, "", "Namespace where the app will be deployed.")
	cmd.Flags().StringVar(&f.Cluster, flagCluster, "", "Name of the cluster the app will be deployed to.")
	cmd.Flags().StringVar(&f.Organization, flagOrganization, "", "Workload cluster organization.")
	cmd.Flags().BoolVar(&f.DefaultingEnabled, flagDefaultingEnabled, true, "Don't template fields that will be defaulted.")
	cmd.Flags().BoolVar(&f.InCluster, flagInCluster, false, fmt.Sprintf("Deploy the app in the current management cluster rather than in a workload cluster. If this is set, --%s will be ignored.", flagCluster))
	cmd.Flags().StringVar(&f.flagUserConfigMap, flagUserConfigMap, "", "Path to the user values configmap YAML file.")
	cmd.Flags().StringVar(&f.flagUserSecret, flagUserSecret, "", "Path to the user secrets YAML file.")
	cmd.Flags().StringVar(&f.Version, flagVersion, "", "App version to be installed.")
	cmd.Flags().StringSliceVar(&f.flagNamespaceConfigAnnotations, flagNamespaceConfigAnnotations, nil, "Namespace configuration annotations in form key=value.")
	cmd.Flags().StringSliceVar(&f.flagNamespaceConfigLabels, flagNamespaceConfigLabels, nil, "Namespace configuration labels in form key=value.")
}

func (f *flag) Validate() error {
	if f.Catalog == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagCatalog)
	}
	if f.Name == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagName)
	}
	if f.Namespace == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagNamespace)
	}
	if !f.InCluster && f.Cluster == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagCluster)
	}
	// Once we fully migrate to org namespace, checking for
	// the `organization` flag should be mandatory.
	//
	/*if !f.InCluster && f.Organization == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagOrganization)
	}*/
	if f.Version == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagVersion)
	}

	_, err := labels.Parse(f.flagNamespaceConfigLabels)
	if err != nil {
		return microerror.Maskf(invalidFlagError, "--%s must contain valid label definitions (%s)", flagNamespaceConfigLabels, err)
	}

	_, err = annotations.Parse(f.flagNamespaceConfigAnnotations)
	if err != nil {
		return microerror.Maskf(invalidFlagError, "--%s must contain valid annotation definitions (%s)", flagNamespaceConfigAnnotations, err)
	}

	return nil
}
