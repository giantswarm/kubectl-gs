package deploy

import (
	"context"
	"fmt"
	"io"
	"strings"

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
	appService   app.Interface
	ctrlClient   client.Client
	
	stderr       io.Writer
	stdout       io.Writer
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

	if err := r.flag.Validate(); err != nil {
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
		return r.handleDeploy(ctx, args)
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

func (r *runner) handleDeploy(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: resource@version argument is required for deploy action", ErrInvalidArgument)
	}

	spec, err := r.parseResourceSpec(args[0], true)
	if err != nil {
		return err
	}

	switch r.flag.Type {
	case "app":
		return r.deployApp(ctx, spec)
	case "config":
		return r.deployConfig(ctx, spec)
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

	switch r.flag.Type {
	case "app":
		return r.undeployApp(ctx, spec)
	case "config":
		return r.undeployConfig(ctx, spec)
	}

	return fmt.Errorf("%w: unsupported resource type: %s", ErrInvalidFlag, r.flag.Type)
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
			return fmt.Errorf("%w: app name is required for listing versions", ErrInvalidArgument)
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
