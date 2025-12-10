package deploy

import (
	"context"
	"fmt"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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

	// Service dependencies
	appService app.Interface
	ctrlClient client.Client

	stderr io.Writer
	stdout io.Writer
}

type resourceSpec struct {
	name    string
	version string
}

type resourceInfo struct {
	name      string
	namespace string
	reason    string
	version   string
	catalog   string
	branch    string
	url       string
	status    string
}

func (r *runner) Run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if err := r.flag.Validate(cmd, r.commonConfig); err != nil {
		return err
	}

	// Initialize dependencies
	k8sClient, err := r.commonConfig.GetClient(r.logger)
	if err != nil {
		return err
	}

	r.ctrlClient = k8sClient.CtrlClient()

	serviceConfig := app.Config{
		Client: r.ctrlClient,
	}
	r.appService, err = app.New(serviceConfig)
	if err != nil {
		return err
	}

	return r.run(ctx, cmd, args)
}

func (r *runner) run(ctx context.Context, cmd *cobra.Command, args []string) error {
	action := r.flag.GetAction()

	switch action {
	case "deploy":
		return r.handleDeploy(ctx, cmd, args)
	case "undeploy":
		return r.handleUndeploy(ctx, args)
	case "status":
		return r.handleStatus(ctx)
	case "list":
		return r.handleList(ctx, args)
	default:
		return fmt.Errorf("%w: unknown action: %s", ErrInvalidFlag, action)
	}
}

func (r *runner) handleDeploy(ctx context.Context, cmd *cobra.Command, args []string) error {
	var spec *resourceSpec
	var err error

	// Handle interactive mode
	if r.flag.Interactive {
		// Interactive mode: select app and version from catalog entries
		spec, err = r.handleInteractiveMode(ctx, cmd, args)
		if err != nil {
			return err
		}
	} else {
		// Non-interactive mode: parse from args
		if len(args) == 0 {
			return fmt.Errorf("%w: resource@version argument is required for deploy action", ErrInvalidArgument)
		}

		spec, err = r.parseResourceSpec(args[0], true)
		if err != nil {
			return err
		}
	}

	// Capture state before deployment if undeploy-on-exit is enabled
	var savedState interface{}
	var stateCaptured bool
	if r.flag.UndeployOnExit {
		var captureErr error
		switch r.flag.Type {
		case "app":
			savedState, captureErr = r.captureAppState(ctx, spec.name, r.flag.Namespace)
		case "config":
			savedState, captureErr = r.captureConfigState(ctx, spec.name, r.flag.Namespace)
		default:
			return fmt.Errorf("%w: unsupported resource type: %s", ErrInvalidFlag, r.flag.Type)
		}
		if captureErr != nil {
			fmt.Fprintf(r.stderr, "Warning: failed to capture state for restore: %v\n", captureErr)
		} else {
			stateCaptured = true
		}
	}

	// Perform the deployment
	var deployErr error
	var deploymentSucceeded bool
	switch r.flag.Type {
	case "app":
		deployErr = r.deployApp(ctx, spec)
	case "config":
		deployErr = r.deployConfig(ctx, spec)
	default:
		return fmt.Errorf("%w: unsupported resource type: %s", ErrInvalidFlag, r.flag.Type)
	}

	if deployErr != nil {
		// If undeploy-on-exit is enabled and state was captured, restore before returning error
		if r.flag.UndeployOnExit && stateCaptured {
			fmt.Fprintf(r.stderr, "\n%s Deployment failed, restoring previous state...\n", warningStyle.Render("⚠"))
			restoreErr := r.restoreState(ctx, r.flag.Type, savedState)
			if restoreErr != nil {
				fmt.Fprintf(r.stderr, "Error: failed to restore state: %v\n", restoreErr)
			}
		}
		return deployErr
	}

	deploymentSucceeded = true

	// If undeploy-on-exit is enabled, wait for interrupt and restore
	if r.flag.UndeployOnExit && deploymentSucceeded {
		return r.waitForInterruptAndRestore(ctx, r.flag.Type, savedState)
	}

	return nil
}

func (r *runner) handleInteractiveMode(ctx context.Context, cmd *cobra.Command, args []string) (*resourceSpec, error) {
	if r.flag.Type == resourceTypeConfig {
		return r.handleInteractiveConfigMode(ctx, args)
	}

	// Extract app name filter from args if provided
	appNameFilter := ""
	if len(args) > 0 {
		// Parse args[0] to extract app name (before @)
		parts := strings.Split(args[0], "@")
		appNameFilter = parts[0]
	}

	// Check if catalog flag was explicitly set to empty string (to trigger catalog selection)
	catalogFilter := r.flag.Catalog
	catalogChanged := cmd.Flags().Changed("catalog")

	// If catalog was explicitly set to empty string, let user select it
	if catalogChanged && catalogFilter == "" {
		selectedCatalog, err := r.selectCatalog(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to select catalog: %w", err)
		}
		catalogFilter = selectedCatalog
		// Update the flag so deployment uses the selected catalog
		r.flag.Catalog = selectedCatalog
	}

	// Select app catalog entry
	result, err := r.selectCatalogEntry(ctx, appNameFilter, catalogFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to select catalog entry: %w", err)
	}

	if result.Canceled {
		return nil, fmt.Errorf("selection canceled")
	}

	// Update catalog flag to match the selected entry's catalog
	r.flag.Catalog = result.Catalog

	return &resourceSpec{
		name:    result.AppName,
		version: result.Version,
	}, nil
}

func (r *runner) handleInteractiveConfigMode(ctx context.Context, args []string) (*resourceSpec, error) {
	// Extract config name filter from args if provided
	configNameFilter := ""
	if len(args) > 0 {
		// Parse args[0] to extract config name (before @)
		parts := strings.Split(args[0], "@")
		configNameFilter = parts[0]
	}

	// Select config version (combined repo + branch/PR)
	result, err := r.selectConfigVersion(ctx, configNameFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to select config version: %w", err)
	}

	if result.Canceled {
		return nil, fmt.Errorf("selection canceled")
	}

	return &resourceSpec{
		name:    result.ConfigRepo,
		version: result.Branch,
	}, nil
}

func (r *runner) handleUndeploy(ctx context.Context, args []string) error {
	var spec *resourceSpec
	var err error

	// Handle interactive mode
	if r.flag.Interactive {
		if r.flag.Type == resourceTypeConfig {
			spec, err = r.handleInteractiveUndeployConfig(ctx)
		} else {
			spec, err = r.handleInteractiveUndeploy(ctx)
		}
		if err != nil {
			return err
		}
	} else {
		// Non-interactive mode: parse from args
		if len(args) == 0 {
			return fmt.Errorf("%w: resource name is required for undeploy action", ErrInvalidArgument)
		}

		spec, err = r.parseResourceSpec(args[0], false)
		if err != nil {
			return err
		}
	}

	switch r.flag.Type {
	case "app":
		return r.undeployApp(ctx, spec)
	case "config":
		return r.undeployConfig(ctx, spec)
	}

	return fmt.Errorf("%w: unsupported resource type: %s", ErrInvalidFlag, r.flag.Type)
}

func (r *runner) handleInteractiveUndeploy(ctx context.Context) (*resourceSpec, error) {
	// Get list of installed apps in the namespace
	installedApps, err := r.appService.ListApps(ctx, r.flag.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to list installed apps: %w", err)
	}

	if len(installedApps.Items) == 0 {
		return nil, fmt.Errorf("no apps found in namespace %s", r.flag.Namespace)
	}

	// Build a list of app items for selection
	items := make([]interface{}, len(installedApps.Items))
	for i, app := range installedApps.Items {
		// Format: name version catalog
		title := fmt.Sprintf("%-40s %-20s %s",
			app.Spec.Name,
			app.Spec.Version,
			app.Spec.Catalog)

		items[i] = catalogEntryItem{
			appName: app.Spec.Name,
			version: app.Spec.Version,
			catalog: app.Spec.Catalog,
			title:   title,
		}
	}

	// Create and run the selector
	model := newSelectorModel(items, "Select app to undeploy:")

	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("error running app selector: %w", err)
	}

	m := finalModel.(selectorModel)
	if m.err != nil {
		return nil, m.err
	}

	if m.selected == nil {
		return nil, fmt.Errorf("selection canceled")
	}

	selectedItem, ok := m.selected.(catalogEntryItem)
	if !ok {
		return nil, fmt.Errorf("unexpected item type")
	}

	return &resourceSpec{
		name: selectedItem.appName,
	}, nil
}

func (r *runner) handleInteractiveUndeployConfig(ctx context.Context) (*resourceSpec, error) {
	// Get list of config repositories in the namespace
	gitRepoList, err := r.listAllConfigRepos(ctx, r.flag.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to list config repositories: %w", err)
	}

	if len(gitRepoList.Items) == 0 {
		return nil, fmt.Errorf("no config repositories found in namespace %s", r.flag.Namespace)
	}

	// Build a list of config items for selection, filtered to those with suspended reconciliation
	items := make([]interface{}, 0)
	for _, gitRepo := range gitRepoList.Items {
		// Only show configs with suspended reconciliation (deployed by us)
		if !isSuspended(&gitRepo) {
			continue
		}

		configName := extractConfigNameFromURL(gitRepo.Spec.URL)
		if configName == "" {
			continue
		}

		currentBranch := ""
		if gitRepo.Spec.Reference != nil {
			currentBranch = gitRepo.Spec.Reference.Branch
		}

		items = append(items, configRepoItem{
			name:   configName,
			url:    gitRepo.Spec.URL,
			branch: currentBranch,
		})
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("no deployed config repositories found in namespace %s", r.flag.Namespace)
	}

	// Create and run the selector
	model := newSelectorModel(items, "Select config to undeploy:")

	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("error running config selector: %w", err)
	}

	m := finalModel.(selectorModel)
	if m.err != nil {
		return nil, m.err
	}

	if m.selected == nil {
		return nil, fmt.Errorf("selection canceled")
	}

	selectedItem, ok := m.selected.(configRepoItem)
	if !ok {
		return nil, fmt.Errorf("unexpected item type")
	}

	return &resourceSpec{
		name: selectedItem.name,
	}, nil
}

func (r *runner) handleStatus(ctx context.Context) error {
	var kustomizationsReady bool
	var notReadyKustomizations, suspendedKustomizations, suspendedApps, suspendedGitRepos []resourceInfo

	// Check kustomizations with spinner
	if err := RunWithSpinner("Checking kustomizations", func() error {
		var err error
		kustomizationsReady, notReadyKustomizations, suspendedKustomizations, err = r.checkKustomizations(ctx)
		return err
	}); err != nil {
		fmt.Fprintf(r.stderr, "Error checking kustomizations: %v\n", err)
	}

	// Check apps with spinner
	if err := RunWithSpinner("Checking apps", func() error {
		var err error
		suspendedApps, err = r.checkApps(ctx)
		return err
	}); err != nil {
		fmt.Fprintf(r.stderr, "Error checking apps: %v\n", err)
	}

	// Check config repositories with spinner
	if err := RunWithSpinner("Checking git repositories", func() error {
		var err error
		suspendedGitRepos, err = r.checkGitRepositories(ctx)
		return err
	}); err != nil {
		fmt.Fprintf(r.stderr, "Error checking git repositories: %v\n", err)
	}

	// Display formatted status
	output := StatusOutput(kustomizationsReady, notReadyKustomizations, suspendedKustomizations, suspendedApps, suspendedGitRepos)
	fmt.Fprint(r.stdout, output)

	return nil
}

func (r *runner) handleList(ctx context.Context, args []string) error {
	switch r.flag.List {
	case "apps":
		return r.listApps(ctx)
	case "versions":
		if len(args) == 0 {
			return fmt.Errorf("%w: resource name is required for listing versions", ErrInvalidArgument)
		}
		if r.flag.Type == resourceTypeConfig {
			return r.listConfigVersions(ctx, args[0])
		}
		return r.listVersions(ctx, args[0])
	case "configs":
		return r.listConfigs(ctx)
	case "catalogs":
		return r.listCatalogs(ctx)
	default:
		return fmt.Errorf("%w: unknown list type: %s", ErrInvalidFlag, r.flag.List)
	}
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
