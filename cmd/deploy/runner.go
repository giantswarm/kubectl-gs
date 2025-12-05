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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

type resourceInfo struct {
	name      string
	namespace string
	reason    string
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

	// Check kustomizations
	kustomizationsReady, kustomizationsSuspended, notReadyKustomizations, err := r.checkKustomizations(ctx, ctrlClient)
	if err != nil {
		fmt.Fprintf(r.stderr, "Error checking kustomizations: %v\n", err)
	}

	// Check apps
	suspendedApps, err := r.checkApps(ctx, ctrlClient)
	if err != nil {
		fmt.Fprintf(r.stderr, "Error checking apps: %v\n", err)
	}

	// Check config repositories (GitRepositories)
	suspendedGitRepos, err := r.checkGitRepositories(ctx, ctrlClient)
	if err != nil {
		fmt.Fprintf(r.stderr, "Error checking git repositories: %v\n", err)
	}

	// Display status
	if kustomizationsReady && !kustomizationsSuspended && len(suspendedApps) == 0 && len(suspendedGitRepos) == 0 {
		fmt.Fprintf(r.stderr, "Kustomisations, config repositories, and apps are healthy\n")
	} else {
		fmt.Fprintf(r.stderr, "Some elements are not healthy\n")

		if !kustomizationsReady {
			fmt.Fprintf(r.stderr, "\nNot ready kustomizations:\n")
			for _, kustomization := range notReadyKustomizations {
				fmt.Fprintf(r.stderr, "  - %s/%s (reason: %s)\n",
					kustomization.namespace, kustomization.name, kustomization.reason)
			}
		}

		if kustomizationsSuspended {
			fmt.Fprintf(r.stderr, "\nWarning: Some kustomizations are suspended\n")
		}

		if len(suspendedApps) > 0 {
			fmt.Fprintf(r.stderr, "\nSuspended apps:\n")
			for _, app := range suspendedApps {
				fmt.Fprintf(r.stderr, "  - %s/%s\n", app.namespace, app.name)
			}
		}

		if len(suspendedGitRepos) > 0 {
			fmt.Fprintf(r.stderr, "\nSuspended git repositories:\n")
			for _, repo := range suspendedGitRepos {
				fmt.Fprintf(r.stderr, "  - %s/%s\n", repo.namespace, repo.name)
			}
		}
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
			fmt.Fprintf(r.stderr, "\n/!\\ Reminder: don't forget to undeploy your changes after testing, with:\n")
			fmt.Fprintf(r.stderr, "kubectl gs deploy -u %s\n", spec.name)
			return nil
		}
		return err
	}

	// App exists, use the app service to patch it with version validation
	// Set SuspendReconciliation to true to prevent Flux from overriding the changes
	patchOptions := app.PatchOptions{
		Namespace:             r.flag.Namespace,
		Name:                  spec.name,
		Version:               spec.version,
		SuspendReconciliation: true,
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
	fmt.Fprintf(r.stderr, "\n/!\\ Reminder: don't forget to undeploy your changes after testing, with:\n")
	fmt.Fprintf(r.stderr, "kubectl gs deploy -u %s\n", spec.name)
	return nil
}

func (r *runner) deployConfig(ctx context.Context, ctrlClient client.Client, spec *resourceSpec) error {
	// Find the GitRepository by matching its URL against the config repo name
	gitRepo, err := r.findGitRepository(ctx, ctrlClient, spec.name, r.flag.Namespace)
	if err != nil {
		return fmt.Errorf("failed to find GitRepository for %s: %w", spec.name, err)
	}

	// Get the resource name and namespace
	resourceName, _, _ := unstructured.NestedString(gitRepo.Object, "metadata", "name")
	resourceNamespace, _, _ := unstructured.NestedString(gitRepo.Object, "metadata", "namespace")

	// Create a patch to set the reconcile annotation and update the branch
	patch := client.MergeFrom(gitRepo.DeepCopy())

	// Add the Flux reconcile annotation to suspend reconciliation
	annotations := gitRepo.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["kustomize.toolkit.fluxcd.io/reconcile"] = "disabled"
	gitRepo.SetAnnotations(annotations)

	// Update the spec.ref.branch to the desired version (branch)
	err = unstructured.SetNestedField(gitRepo.Object, spec.version, "spec", "ref", "branch")
	if err != nil {
		return fmt.Errorf("failed to set branch: %w", err)
	}

	// Apply the patch
	err = ctrlClient.Patch(ctx, gitRepo, patch)
	if err != nil {
		return fmt.Errorf("failed to patch GitRepository: %w", err)
	}

	fmt.Fprintf(r.stdout, "Config repository %s in namespace %s deployed with branch %s\n", resourceName, resourceNamespace, spec.version)
	fmt.Fprintf(r.stderr, "\n/!\\ Reminder: don't forget to undeploy your changes after testing, with:\n")
	fmt.Fprintf(r.stderr, "kubectl gs deploy -t config -u %s\n", spec.name)
	return nil
}

func (r *runner) undeployApp(ctx context.Context, ctrlClient client.Client, spec *resourceSpec) error {
	// Initialize the app service
	err := r.getAppService()
	if err != nil {
		return err
	}

	// Use the app service to patch the app and remove the Flux reconciliation annotation
	// This allows Flux to manage the resource again
	patchOptions := app.PatchOptions{
		Namespace:             r.flag.Namespace,
		Name:                  spec.name,
		Version:               "", // Don't change the version during undeploy
		SuspendReconciliation: false,
	}

	state, err := r.appService.Patch(ctx, patchOptions)
	if app.IsNotFound(err) {
		return fmt.Errorf("app %s not found in namespace %s", spec.name, r.flag.Namespace)
	} else if err != nil {
		return err
	}

	fmt.Fprintf(r.stdout, "App %s in namespace %s undeployed successfully\n", spec.name, r.flag.Namespace)
	if len(state) > 0 {
		fmt.Fprintf(r.stdout, "Changes: %s\n", strings.Join(state, " "))
	}
	return nil
}

func (r *runner) undeployConfig(ctx context.Context, ctrlClient client.Client, spec *resourceSpec) error {
	// Find the GitRepository by matching its URL against the config repo name
	gitRepo, err := r.findGitRepository(ctx, ctrlClient, spec.name, r.flag.Namespace)
	if err != nil {
		return fmt.Errorf("failed to find GitRepository for %s: %w", spec.name, err)
	}

	// Get the resource name and namespace
	resourceName, _, _ := unstructured.NestedString(gitRepo.Object, "metadata", "name")
	resourceNamespace, _, _ := unstructured.NestedString(gitRepo.Object, "metadata", "namespace")

	// Create a patch to remove the reconcile annotation and label
	patch := client.MergeFrom(gitRepo.DeepCopy())

	// Remove the Flux reconcile annotation
	annotations := gitRepo.GetAnnotations()
	if annotations != nil {
		delete(annotations, "kustomize.toolkit.fluxcd.io/reconcile")
		gitRepo.SetAnnotations(annotations)
	}

	// Remove the Flux reconcile label if present
	labels := gitRepo.GetLabels()
	if labels != nil {
		delete(labels, "kustomize.toolkit.fluxcd.io/reconcile")
		gitRepo.SetLabels(labels)
	}

	// Apply the patch
	err = ctrlClient.Patch(ctx, gitRepo, patch)
	if err != nil {
		return fmt.Errorf("failed to patch GitRepository: %w", err)
	}

	fmt.Fprintf(r.stdout, "Config repository %s in namespace %s undeployed successfully\n", resourceName, resourceNamespace)
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

// findGitRepository finds a GitRepository CR by matching the config repo name in its URL
func (r *runner) findGitRepository(ctx context.Context, ctrlClient client.Client, configRepoName, namespace string) (*unstructured.Unstructured, error) {
	// Define the GitRepository GVK
	gvk := schema.GroupVersionKind{
		Group:   "source.toolkit.fluxcd.io",
		Version: "v1",
		Kind:    "GitRepository",
	}

	// List all GitRepository resources in the namespace
	gitRepoList := &unstructured.UnstructuredList{}
	gitRepoList.SetGroupVersionKind(gvk)

	listOptions := &client.ListOptions{
		Namespace: namespace,
	}

	err := ctrlClient.List(ctx, gitRepoList, listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list GitRepositories: %w", err)
	}

	// Find the GitRepository that matches the config repo name in its URL
	for _, item := range gitRepoList.Items {
		url, found, err := unstructured.NestedString(item.Object, "spec", "url")
		if err != nil || !found {
			continue
		}

		// Check if the URL contains the config repo name
		// The URL format is typically: https://github.com/giantswarm/{configRepoName}
		if strings.Contains(url, fmt.Sprintf("giantswarm/%s", configRepoName)) {
			return &item, nil
		}
	}

	return nil, fmt.Errorf("%w: GitRepository for config repo %s not found in namespace %s", ErrResourceNotFound, configRepoName, namespace)
}

// checkKustomizations checks the health of all Kustomization resources
func (r *runner) checkKustomizations(ctx context.Context, ctrlClient client.Client) (allReady bool, anySuspended bool, notReady []resourceInfo, err error) {
	gvk := schema.GroupVersionKind{
		Group:   "kustomize.toolkit.fluxcd.io",
		Version: "v1",
		Kind:    "Kustomization",
	}

	kustomizationList := &unstructured.UnstructuredList{}
	kustomizationList.SetGroupVersionKind(gvk)

	err = ctrlClient.List(ctx, kustomizationList)
	if err != nil {
		return false, false, nil, err
	}

	allReady = true
	anySuspended = false
	notReady = []resourceInfo{}

	for _, item := range kustomizationList.Items {
		// Check if suspended
		suspended, _, _ := unstructured.NestedBool(item.Object, "spec", "suspend")
		if suspended {
			anySuspended = true
		}

		// Check ready condition
		conditions, found, _ := unstructured.NestedSlice(item.Object, "status", "conditions")
		if !found {
			allReady = false
			continue
		}

		ready := false
		reason := ""
		for _, cond := range conditions {
			condMap, ok := cond.(map[string]interface{})
			if !ok {
				continue
			}

			condType, _, _ := unstructured.NestedString(condMap, "type")
			if condType == "Ready" {
				status, _, _ := unstructured.NestedString(condMap, "status")
				ready = (status == "True")
				if !ready {
					reason, _, _ = unstructured.NestedString(condMap, "reason")
				}
				break
			}
		}

		if !ready {
			allReady = false
			name, _, _ := unstructured.NestedString(item.Object, "metadata", "name")
			namespace, _, _ := unstructured.NestedString(item.Object, "metadata", "namespace")
			notReady = append(notReady, resourceInfo{
				name:      name,
				namespace: namespace,
				reason:    reason,
			})
		}
	}

	return allReady, anySuspended, notReady, nil
}

// checkApps checks if any apps have Flux reconciliation suspended
func (r *runner) checkApps(ctx context.Context, ctrlClient client.Client) ([]resourceInfo, error) {
	apps := &applicationv1alpha1.AppList{}
	err := ctrlClient.List(ctx, apps, &client.ListOptions{})
	if err != nil {
		return nil, err
	}

	suspendedApps := []resourceInfo{}

	for _, app := range apps.Items {
		// Check if the app has the Flux reconcile annotation or label set to disabled
		annotations := app.GetAnnotations()
		labels := app.GetLabels()

		isSuspended := false
		if annotations != nil {
			if val, ok := annotations["kustomize.toolkit.fluxcd.io/reconcile"]; ok && val == "disabled" {
				isSuspended = true
			}
		}
		if labels != nil {
			if val, ok := labels["kustomize.toolkit.fluxcd.io/reconcile"]; ok && val == "disabled" {
				isSuspended = true
			}
		}

		if isSuspended {
			suspendedApps = append(suspendedApps, resourceInfo{
				name:      app.Name,
				namespace: app.Namespace,
			})
		}
	}

	return suspendedApps, nil
}

// checkGitRepositories checks if any GitRepository resources have Flux reconciliation suspended
func (r *runner) checkGitRepositories(ctx context.Context, ctrlClient client.Client) ([]resourceInfo, error) {
	gvk := schema.GroupVersionKind{
		Group:   "source.toolkit.fluxcd.io",
		Version: "v1",
		Kind:    "GitRepository",
	}

	gitRepoList := &unstructured.UnstructuredList{}
	gitRepoList.SetGroupVersionKind(gvk)

	err := ctrlClient.List(ctx, gitRepoList)
	if err != nil {
		return nil, err
	}

	suspendedRepos := []resourceInfo{}

	for _, item := range gitRepoList.Items {
		annotations := item.GetAnnotations()
		labels := item.GetLabels()

		isSuspended := false
		if annotations != nil {
			if val, ok := annotations["kustomize.toolkit.fluxcd.io/reconcile"]; ok && val == "disabled" {
				isSuspended = true
			}
		}
		if labels != nil {
			if val, ok := labels["kustomize.toolkit.fluxcd.io/reconcile"]; ok && val == "disabled" {
				isSuspended = true
			}
		}

		if isSuspended {
			name, _, _ := unstructured.NestedString(item.Object, "metadata", "name")
			namespace, _, _ := unstructured.NestedString(item.Object, "metadata", "namespace")
			suspendedRepos = append(suspendedRepos, resourceInfo{
				name:      name,
				namespace: namespace,
			})
		}
	}

	return suspendedRepos, nil
}
