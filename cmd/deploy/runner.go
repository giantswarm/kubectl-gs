package deploy

import (
	"context"
	"fmt"
	"io"
	"strings"

	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/kubectl-gs/v5/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/v5/pkg/data/domain/app"
)

type runner struct {
	commonConfig *commonconfig.CommonConfig
	flag         *flag
	logger       micrologger.Logger
	fs           afero.Fs
	appService   app.Interface
	stderr       io.Writer
	stdout       io.Writer
}

type resourceSpec struct {
	name    string
	version string
}

func (r *runner) Run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	err := r.flag.Validate()
	if err != nil {
		return err
	}

	err = r.run(ctx, cmd, args)
	if err != nil {
		return err
	}

	return nil
}

func (r *runner) run(ctx context.Context, cmd *cobra.Command, args []string) error {
	action := r.flag.GetAction()

	switch action {
	case "deploy":
		return r.handleDeploy(ctx, args)
	case "undeploy":
		return r.handleUndeploy(ctx, args)
	case "status":
		return r.handleStatus(ctx)
	default:
		return fmt.Errorf("%w: unknown action: %s", ErrInvalidFlag, action)
	}
}

func (r *runner) handleDeploy(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: resource@version argument is required for deploy action", ErrInvalidArgument)
	}

	spec, err := r.parseResourceSpec(args[0], true)
	if err != nil {
		return err
	}

	k8sClient, err := r.commonConfig.GetClient(r.logger)
	if err != nil {
		return err
	}

	ctrlClient := k8sClient.CtrlClient()

	switch r.flag.Type {
	case "app":
		return r.deployApp(ctx, ctrlClient, spec)
	case "config":
		return r.deployConfig(ctx, ctrlClient, spec)
	}

	return fmt.Errorf("%w: unsupported resource type: %s", ErrInvalidFlag, r.flag.Type)
}

func (r *runner) handleUndeploy(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: resource name is required for undeploy action", ErrInvalidArgument)
	}

	spec, err := r.parseResourceSpec(args[0], false)
	if err != nil {
		return err
	}

	k8sClient, err := r.commonConfig.GetClient(r.logger)
	if err != nil {
		return err
	}

	ctrlClient := k8sClient.CtrlClient()

	switch r.flag.Type {
	case "app":
		return r.undeployApp(ctx, ctrlClient, spec)
	case "config":
		return r.undeployConfig(ctx, ctrlClient, spec)
	}

	return fmt.Errorf("%w: unsupported resource type: %s", ErrInvalidFlag, r.flag.Type)
}

func (r *runner) handleStatus(ctx context.Context) error {
	k8sClient, err := r.commonConfig.GetClient(r.logger)
	if err != nil {
		return err
	}

	ctrlClient := k8sClient.CtrlClient()

	fmt.Fprintf(r.stdout, "Cluster Status:\n\n")

	// List all apps
	err = r.listApps(ctx, ctrlClient)
	if err != nil {
		fmt.Fprintf(r.stderr, "Error listing apps: %v\n", err)
	}

	// List kustomizations (Flux resources)
	err = r.listKustomizations(ctx, ctrlClient)
	if err != nil {
		fmt.Fprintf(r.stderr, "Error listing kustomizations: %v\n", err)
	}

	// List config repositories (GitRepository resources)
	err = r.listGitRepositories(ctx, ctrlClient)
	if err != nil {
		fmt.Fprintf(r.stderr, "Error listing git repositories: %v\n", err)
	}

	return nil
}

func (r *runner) getAppService() error {
	if r.appService != nil {
		return nil
	}

	client, err := r.commonConfig.GetClient(r.logger)
	if err != nil {
		return err
	}

	serviceConfig := app.Config{
		Client: client.CtrlClient(),
	}
	r.appService, err = app.New(serviceConfig)
	if err != nil {
		return err
	}

	return nil
}

func (r *runner) deployApp(ctx context.Context, ctrlClient client.Client, spec *resourceSpec) error {
	// Initialize the app service
	err := r.getAppService()
	if err != nil {
		return err
	}

	// Try to get existing app to determine if we need to create or update
	existingApp := &applicationv1alpha1.App{}
	err = ctrlClient.Get(ctx, client.ObjectKey{
		Name:      spec.name,
		Namespace: r.flag.Namespace,
	}, existingApp)
	if err != nil {
		// App doesn't exist, create it
		if client.IgnoreNotFound(err) == nil {
			createOptions := app.CreateOptions{
				Name:         spec.name,
				Namespace:    r.flag.Namespace,
				AppName:      spec.name,
				AppNamespace: r.flag.Namespace,
				AppCatalog:   r.flag.Catalog,
				AppVersion:   spec.version,
			}

			_, err = r.appService.Create(ctx, createOptions)
			if app.IsNoResources(err) {
				return fmt.Errorf("no app with the name %s and the version %s found in the catalog", spec.name, spec.version)
			} else if err != nil {
				return err
			}
			fmt.Fprintf(r.stdout, "App %s@%s deployed successfully to namespace %s\n", spec.name, spec.version, r.flag.Namespace)
			return nil
		}
		return err
	}

	// App exists, use the app service to patch it with version validation
	patchOptions := app.PatchOptions{
		Namespace:             r.flag.Namespace,
		Name:                  spec.name,
		Version:               spec.version,
		SuspendReconciliation: false,
	}

	state, err := r.appService.Patch(ctx, patchOptions)
	if app.IsNotFound(err) {
		return fmt.Errorf("app %s not found in namespace %s", spec.name, r.flag.Namespace)
	} else if app.IsNoResources(err) {
		return fmt.Errorf("no app with the name %s and the version %s found in the catalog", spec.name, spec.version)
	} else if err != nil {
		return err
	}

	fmt.Fprintf(r.stdout, "App %s in namespace %s updated with %s\n", spec.name, r.flag.Namespace, strings.Join(state, " "))
	return nil
}

func (r *runner) deployConfig(ctx context.Context, ctrlClient client.Client, spec *resourceSpec) error {
	// Config repositories are typically GitRepository CRs managed by Flux
	// This is a placeholder implementation
	fmt.Fprintf(r.stdout, "Deploying config repository %s@%s to namespace %s\n", spec.name, spec.version, r.flag.Namespace)
	fmt.Fprintf(r.stderr, "Config repository deployment is not yet fully implemented\n")
	return nil
}

func (r *runner) undeployApp(ctx context.Context, ctrlClient client.Client, spec *resourceSpec) error {
	// TODO: Implement undeploy
	return fmt.Errorf("undeploy not yet implemented")
}

func (r *runner) undeployConfig(ctx context.Context, ctrlClient client.Client, spec *resourceSpec) error {
	// Config repository undeployment placeholder
	fmt.Fprintf(r.stdout, "Undeploying config repository %s from namespace %s\n", spec.name, r.flag.Namespace)
	fmt.Fprintf(r.stderr, "Config repository undeployment is not yet fully implemented\n")
	return nil
}

func (r *runner) listApps(ctx context.Context, ctrlClient client.Client) error {
	apps := &applicationv1alpha1.AppList{}
	err := ctrlClient.List(ctx, apps, &client.ListOptions{
		Namespace: r.flag.Namespace,
	})
	if err != nil {
		return err
	}

	if len(apps.Items) == 0 {
		fmt.Fprintf(r.stdout, "No apps found in namespace %s\n", r.flag.Namespace)
		return nil
	}

	fmt.Fprintf(r.stdout, "Apps in namespace %s:\n", r.flag.Namespace)
	for _, app := range apps.Items {
		status := "Unknown"
		if app.Status.Release.Status != "" {
			status = app.Status.Release.Status
		}
		fmt.Fprintf(r.stdout, "  - %s (version: %s, status: %s)\n", app.Name, app.Spec.Version, status)
	}
	fmt.Fprintf(r.stdout, "\n")

	return nil
}

func (r *runner) listKustomizations(ctx context.Context, ctrlClient client.Client) error {
	// Kustomizations are Flux resources
	// This would require importing flux types, which may not be available
	// For now, we'll skip this or use unstructured types
	fmt.Fprintf(r.stdout, "Kustomizations: (listing not yet implemented)\n\n")
	return nil
}

func (r *runner) listGitRepositories(ctx context.Context, ctrlClient client.Client) error {
	// GitRepository resources from Flux
	// Similar to kustomizations, this requires flux types
	fmt.Fprintf(r.stdout, "Config Repositories: (listing not yet implemented)\n\n")
	return nil
}

func (r *runner) parseResourceSpec(arg string, requireVersion bool) (*resourceSpec, error) {
	parts := strings.Split(arg, "@")

	if len(parts) == 1 {
		if requireVersion {
			return nil, fmt.Errorf("%w: version is required, format: resource@version", ErrInvalidArgument)
		}
		return &resourceSpec{
			name: parts[0],
		}, nil
	}

	if len(parts) == 2 {
		if parts[0] == "" {
			return nil, fmt.Errorf("%w: resource name cannot be empty", ErrInvalidArgument)
		}
		if parts[1] == "" && requireVersion {
			return nil, fmt.Errorf("%w: version cannot be empty", ErrInvalidArgument)
		}
		return &resourceSpec{
			name:    parts[0],
			version: parts[1],
		}, nil
	}

	return nil, fmt.Errorf("%w: invalid resource format, expected: resource@version", ErrInvalidArgument)
}
