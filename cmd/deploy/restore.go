package deploy

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	k8smetadataAnnotation "github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/kubectl-gs/v5/pkg/data/domain/app"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// savedAppState holds the previous state of an app before deployment
type savedAppState struct {
	name      string
	namespace string
	version   string
	catalog   string
}

// savedConfigState holds the previous state of a config repository before deployment
type savedConfigState struct {
	resourceName      string
	resourceNamespace string
	branch            string
}

// captureAppState captures the current state of an app before deployment
func (r *runner) captureAppState(ctx context.Context, name, namespace string) (*savedAppState, error) {
	app, err := r.appService.GetApp(ctx, namespace, name)
	if err != nil {
		// App doesn't exist, return nil state
		return nil, nil
	}

	return &savedAppState{
		name:      name,
		namespace: namespace,
		version:   app.Spec.Version,
		catalog:   app.Spec.Catalog,
	}, nil
}

// captureConfigState captures the current state of a config repository before deployment
func (r *runner) captureConfigState(ctx context.Context, configName, namespace string) (*savedConfigState, error) {
	result, err := r.findGitRepository(ctx, configName, namespace)
	if err != nil {
		return nil, err
	}

	if result == nil {
		// Config doesn't exist
		return nil, nil
	}

	var branch string
	if result.Spec.Reference != nil {
		branch = result.Spec.Reference.Branch
	}

	return &savedConfigState{
		resourceName:      result.Name,
		resourceNamespace: result.Namespace,
		branch:            branch,
	}, nil
}

// waitForInterruptAndRestore waits for an interrupt signal and restores the previous state
func (r *runner) waitForInterruptAndRestore(ctx context.Context, resourceType string, savedState interface{}) error {
	// Set up signal handler
	signalCtx, _ := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	fmt.Fprintf(r.stdout, "\n%s Waiting for interrupt signal (Ctrl+C) to restore previous state...\n",
		infoStyle.Render("ℹ"))
	fmt.Fprintf(r.stdout, "%s Press Ctrl+C to restore and exit\n\n",
		mutedStyle.Render("⌨"))

	// Wait for either signal or context cancellation
	<-signalCtx.Done()
	fmt.Fprintf(r.stdout, "\n%s Stopping and restoring previous state...\n", warningStyle.Render("⚠"))

	// Restore based on resource type
	switch resourceType {
	case "app":
		return r.restoreAppState(ctx, savedState.(*savedAppState))
	case "config":
		return r.restoreConfigState(ctx, savedState.(*savedConfigState))
	default:
		return fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

// restoreAppState restores an app to its previous state
func (r *runner) restoreAppState(ctx context.Context, state *savedAppState) error {
	if state == nil {
		// App didn't exist before, so delete it
		fmt.Fprintf(r.stderr, "App did not exist before deployment, deletion not yet implemented\n")
		return nil
	}

	// Restore the app to its previous version and suspension state
	// The app service Patch method handles setting/unsetting the flux reconciliation annotation
	// based on the SuspendReconciliation parameter
	err := RunWithSpinner(fmt.Sprintf("Restoring app %s to version %s", state.name, state.version), func() error {
		patchOptions := app.PatchOptions{
			Namespace:             state.namespace,
			Name:                  state.name,
			Version:               state.version,
			SuspendReconciliation: false, // Always unsuspend during restore
		}
		_, patchErr := r.appService.Patch(ctx, patchOptions)
		return patchErr
	})
	if err != nil {
		return fmt.Errorf("failed to restore app state: %w", err)
	}

	output := fmt.Sprintf("%s App %s restored to version %s\n",
		successStyle.Render("✓"),
		state.name,
		state.version)
	fmt.Fprint(r.stdout, output)

	return nil
}

// restoreConfigState restores a config repository to its previous state
func (r *runner) restoreConfigState(ctx context.Context, state *savedConfigState) error {
	if state == nil {
		// Config didn't exist before
		fmt.Fprintf(r.stderr, "Config did not exist before deployment, deletion not yet implemented\n")
		return nil
	}

	var gitRepo *sourcev1.GitRepository

	// Restore the config to its previous branch
	err := RunWithSpinner(fmt.Sprintf("Restoring config %s to branch %s", state.resourceName, state.branch), func() error {
		// Get the GitRepository directly by name and namespace
		gitRepo = &sourcev1.GitRepository{}
		key := client.ObjectKey{
			Name:      state.resourceName,
			Namespace: state.resourceNamespace,
		}

		getErr := r.ctrlClient.Get(ctx, key, gitRepo)
		if getErr != nil {
			return fmt.Errorf("failed to get GitRepository: %w", getErr)
		}

		// Create a patch to update the branch
		patch := client.MergeFrom(gitRepo.DeepCopy())

		// Update the branch
		if gitRepo.Spec.Reference == nil {
			gitRepo.Spec.Reference = &sourcev1.GitRepositoryRef{}
		}
		gitRepo.Spec.Reference.Branch = state.branch

		// Restore to non-suspended state - ensure annotation and label are removed
		annotations := gitRepo.GetAnnotations()
		labels := gitRepo.GetLabels()
		delete(annotations, k8smetadataAnnotation.FluxKustomizeReconcile)
		delete(labels, k8smetadataAnnotation.FluxKustomizeReconcile)
		gitRepo.SetAnnotations(annotations)
		gitRepo.SetLabels(labels)

		// Apply the patch
		return r.ctrlClient.Patch(ctx, gitRepo, patch)
	})
	if err != nil {
		return fmt.Errorf("failed to restore config state: %w", err)
	}

	output := fmt.Sprintf("%s Config %s restored to branch %s\n",
		successStyle.Render("✓"),
		state.resourceName,
		state.branch)
	fmt.Fprint(r.stdout, output)

	return nil
}
