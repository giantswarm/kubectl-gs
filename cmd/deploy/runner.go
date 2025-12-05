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
	"github.com/giantswarm/kubectl-gs/v5/pkg/data/domain/catalog"
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
	version   string
	catalog   string
	branch    string
	url       string
	status    string
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

	var (
		kustomizationsReady      bool
		notReadyKustomizations   []resourceInfo
		suspendedKustomizations  []resourceInfo
		suspendedApps            []resourceInfo
		suspendedGitRepos        []resourceInfo
	)

	// Check kustomizations with spinner
	err = RunWithSpinner("Checking kustomizations", func() error {
		var checkErr error
		kustomizationsReady, notReadyKustomizations, suspendedKustomizations, checkErr = r.checkKustomizations(ctx, ctrlClient)
		return checkErr
	})
	if err != nil {
		fmt.Fprintf(r.stderr, "Error checking kustomizations: %v\n", err)
	}

	// Check apps with spinner
	err = RunWithSpinner("Checking apps", func() error {
		var checkErr error
		suspendedApps, checkErr = r.checkApps(ctx, ctrlClient)
		return checkErr
	})
	if err != nil {
		fmt.Fprintf(r.stderr, "Error checking apps: %v\n", err)
	}

	// Check config repositories with spinner
	err = RunWithSpinner("Checking git repositories", func() error {
		var checkErr error
		suspendedGitRepos, checkErr = r.checkGitRepositories(ctx, ctrlClient)
		return checkErr
	})
	if err != nil {
		fmt.Fprintf(r.stderr, "Error checking git repositories: %v\n", err)
	}

	// Display formatted status
	output := StatusOutput(
		kustomizationsReady,
		notReadyKustomizations,
		suspendedKustomizations,
		suspendedApps,
		suspendedGitRepos,
	)
	fmt.Fprint(r.stdout, output)

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
			var createErr error
			err = RunWithSpinner(fmt.Sprintf("Deploying app %s@%s", spec.name, spec.version), func() error {
				createOptions := app.CreateOptions{
					Name:         spec.name,
					Namespace:    r.flag.Namespace,
					AppName:      spec.name,
					AppNamespace: r.flag.Namespace,
					AppCatalog:   r.flag.Catalog,
					AppVersion:   spec.version,
				}

				_, createErr = r.appService.Create(ctx, createOptions)
				return createErr
			})

			if app.IsNoResources(err) {
				return fmt.Errorf("no app with the name %s and the version %s found in the catalog", spec.name, spec.version)
			} else if err != nil {
				return err
			}

			output := DeployOutput("app", spec.name, spec.version, r.flag.Namespace)
			fmt.Fprint(r.stdout, output)
			return nil
		}
		return err
	}

	// App exists, use the app service to patch it with version validation
	var state []string
	err = RunWithSpinner(fmt.Sprintf("Updating app %s to version %s", spec.name, spec.version), func() error {
		// Set SuspendReconciliation to true to prevent Flux from overriding the changes
		patchOptions := app.PatchOptions{
			Namespace:             r.flag.Namespace,
			Name:                  spec.name,
			Version:               spec.version,
			SuspendReconciliation: true,
		}

		var patchErr error
		state, patchErr = r.appService.Patch(ctx, patchOptions)
		return patchErr
	})

	if app.IsNotFound(err) {
		return fmt.Errorf("app %s not found in namespace %s", spec.name, r.flag.Namespace)
	} else if app.IsNoResources(err) {
		return fmt.Errorf("no app with the name %s and the version %s found in the catalog", spec.name, spec.version)
	} else if err != nil {
		return err
	}

	output := UpdateOutput(spec.name, r.flag.Namespace, state)
	fmt.Fprint(r.stdout, output)
	return nil
}

func (r *runner) deployConfig(ctx context.Context, ctrlClient client.Client, spec *resourceSpec) error {
	var gitRepo *unstructured.Unstructured
	var resourceName, resourceNamespace string

	// Find and patch the GitRepository with spinner
	err := RunWithSpinner(fmt.Sprintf("Deploying config repository %s@%s", spec.name, spec.version), func() error {
		var findErr error
		gitRepo, findErr = r.findGitRepository(ctx, ctrlClient, spec.name, r.flag.Namespace)
		if findErr != nil {
			return fmt.Errorf("failed to find GitRepository for %s: %w", spec.name, findErr)
		}

		// Get the resource name and namespace
		resourceName, _, _ = unstructured.NestedString(gitRepo.Object, "metadata", "name")
		resourceNamespace, _, _ = unstructured.NestedString(gitRepo.Object, "metadata", "namespace")

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
		setErr := unstructured.SetNestedField(gitRepo.Object, spec.version, "spec", "ref", "branch")
		if setErr != nil {
			return fmt.Errorf("failed to set branch: %w", setErr)
		}

		// Apply the patch
		patchErr := ctrlClient.Patch(ctx, gitRepo, patch)
		if patchErr != nil {
			return fmt.Errorf("failed to patch GitRepository: %w", patchErr)
		}

		return nil
	})

	if err != nil {
		return err
	}

	output := DeployOutput("config", resourceName, spec.version, resourceNamespace)
	fmt.Fprint(r.stdout, output)
	return nil
}

func (r *runner) undeployApp(ctx context.Context, ctrlClient client.Client, spec *resourceSpec) error {
	// Initialize the app service
	err := r.getAppService()
	if err != nil {
		return err
	}

	var state []string
	err = RunWithSpinner(fmt.Sprintf("Undeploying app %s", spec.name), func() error {
		// Use the app service to patch the app and remove the Flux reconciliation annotation
		// This allows Flux to manage the resource again
		patchOptions := app.PatchOptions{
			Namespace:             r.flag.Namespace,
			Name:                  spec.name,
			Version:               "", // Don't change the version during undeploy
			SuspendReconciliation: false,
		}

		var patchErr error
		state, patchErr = r.appService.Patch(ctx, patchOptions)
		return patchErr
	})

	if app.IsNotFound(err) {
		return fmt.Errorf("app %s not found in namespace %s", spec.name, r.flag.Namespace)
	} else if err != nil {
		return err
	}

	output := UndeployOutput("app", spec.name, r.flag.Namespace, state)
	fmt.Fprint(r.stdout, output)
	return nil
}

func (r *runner) undeployConfig(ctx context.Context, ctrlClient client.Client, spec *resourceSpec) error {
	var resourceName, resourceNamespace string

	err := RunWithSpinner(fmt.Sprintf("Undeploying config repository %s", spec.name), func() error {
		// Find the GitRepository by matching its URL against the config repo name
		gitRepo, findErr := r.findGitRepository(ctx, ctrlClient, spec.name, r.flag.Namespace)
		if findErr != nil {
			return fmt.Errorf("failed to find GitRepository for %s: %w", spec.name, findErr)
		}

		// Get the resource name and namespace
		resourceName, _, _ = unstructured.NestedString(gitRepo.Object, "metadata", "name")
		resourceNamespace, _, _ = unstructured.NestedString(gitRepo.Object, "metadata", "namespace")

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
		patchErr := ctrlClient.Patch(ctx, gitRepo, patch)
		if patchErr != nil {
			return fmt.Errorf("failed to patch GitRepository: %w", patchErr)
		}

		return nil
	})

	if err != nil {
		return err
	}

	output := UndeployOutput("config", resourceName, resourceNamespace, nil)
	fmt.Fprint(r.stdout, output)
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
func (r *runner) checkKustomizations(ctx context.Context, ctrlClient client.Client) (allReady bool, notReady []resourceInfo, suspended []resourceInfo, err error) {
	gvk := schema.GroupVersionKind{
		Group:   "kustomize.toolkit.fluxcd.io",
		Version: "v1",
		Kind:    "Kustomization",
	}

	kustomizationList := &unstructured.UnstructuredList{}
	kustomizationList.SetGroupVersionKind(gvk)

	err = ctrlClient.List(ctx, kustomizationList)
	if err != nil {
		return false, nil, nil, err
	}

	allReady = true
	notReady = []resourceInfo{}
	suspended = []resourceInfo{}

	for _, item := range kustomizationList.Items {
		name, _, _ := unstructured.NestedString(item.Object, "metadata", "name")
		namespace, _, _ := unstructured.NestedString(item.Object, "metadata", "namespace")

		// Check if suspended
		isSuspended, _, _ := unstructured.NestedBool(item.Object, "spec", "suspend")
		if isSuspended {
			suspended = append(suspended, resourceInfo{
				name:      name,
				namespace: namespace,
			})
			// Skip ready check for suspended kustomizations
			continue
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
			notReady = append(notReady, resourceInfo{
				name:      name,
				namespace: namespace,
				reason:    reason,
			})
		}
	}

	return allReady, notReady, suspended, nil
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
			status := app.Status.Release.Status
			if status == "" {
				status = "Unknown"
			}
			suspendedApps = append(suspendedApps, resourceInfo{
				name:      app.Name,
				namespace: app.Namespace,
				version:   app.Spec.Version,
				catalog:   app.Spec.Catalog,
				status:    status,
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
			branch, _, _ := unstructured.NestedString(item.Object, "spec", "ref", "branch")
			url, _, _ := unstructured.NestedString(item.Object, "spec", "url")

			// Get status from Ready condition
			status := "Unknown"
			conditions, found, _ := unstructured.NestedSlice(item.Object, "status", "conditions")
			if found {
				for _, cond := range conditions {
					condMap, ok := cond.(map[string]interface{})
					if !ok {
						continue
					}
					condType, _, _ := unstructured.NestedString(condMap, "type")
					if condType == "Ready" {
						condStatus, _, _ := unstructured.NestedString(condMap, "status")
						if condStatus == "True" {
							status = "Ready"
						} else {
							reason, _, _ := unstructured.NestedString(condMap, "reason")
							if reason != "" {
								status = reason
							} else {
								status = "Not Ready"
							}
						}
						break
					}
				}
			}

			suspendedRepos = append(suspendedRepos, resourceInfo{
				name:      name,
				namespace: namespace,
				branch:    branch,
				url:       url,
				status:    status,
			})
		}
	}

	return suspendedRepos, nil
}

func (r *runner) handleList(ctx context.Context, args []string) error {
	k8sClient, err := r.commonConfig.GetClient(r.logger)
	if err != nil {
		return err
	}

	ctrlClient := k8sClient.CtrlClient()

	switch r.flag.List {
	case "apps":
		return r.listApps(ctx, ctrlClient)
	case "versions":
		if len(args) == 0 {
			return fmt.Errorf("%w: app name is required for listing versions", ErrInvalidArgument)
		}
		return r.listVersions(ctx, ctrlClient, args[0])
	case "configs":
		return r.listConfigs(ctx, ctrlClient)
	case "catalogs":
		return r.listCatalogs(ctx, ctrlClient)
	default:
		return fmt.Errorf("%w: unknown list type: %s", ErrInvalidFlag, r.flag.List)
	}
}

func (r *runner) listApps(ctx context.Context, ctrlClient client.Client) error {
	// Initialize the app service
	err := r.getAppService()
	if err != nil {
		return err
	}

	var apps *applicationv1alpha1.AppList
	err = RunWithSpinner("Listing apps", func() error {
		var listErr error
		apps, listErr = r.appService.ListApps(ctx, r.flag.Namespace)
		return listErr
	})

	if app.IsNoResources(err) {
		fmt.Fprintf(r.stdout, "No apps found in namespace %s\n", r.flag.Namespace)
		return nil
	} else if err != nil {
		return err
	}

	output := ListAppsOutput(apps, r.flag.Namespace)
	fmt.Fprint(r.stdout, output)
	return nil
}

func (r *runner) listVersions(ctx context.Context, ctrlClient client.Client, appName string) error {
	// Initialize the app service
	err := r.getAppService()
	if err != nil {
		return err
	}

	var entries *applicationv1alpha1.AppCatalogEntryList
	var deployedVersion string
	var deployedCatalog string
	err = RunWithSpinner(fmt.Sprintf("Listing versions for %s", appName), func() error {
		// Get catalog data service
		catalogDataService, serviceErr := r.getCatalogService(ctrlClient)
		if serviceErr != nil {
			return serviceErr
		}

		// List all entries for this app name
		selector := fmt.Sprintf("app.kubernetes.io/name=%s", appName)
		var listErr error
		entries, listErr = catalogDataService.GetEntries(ctx, selector)
		if listErr != nil {
			return listErr
		}

		// Try to get the currently deployed app version and catalog
		existingApp := &applicationv1alpha1.App{}
		getErr := ctrlClient.Get(ctx, client.ObjectKey{
			Name:      appName,
			Namespace: r.flag.Namespace,
		}, existingApp)
		if getErr == nil {
			deployedVersion = existingApp.Spec.Version
			deployedCatalog = existingApp.Spec.Catalog
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to list versions for app %s: %w", appName, err)
	}

	output := ListVersionsOutput(appName, entries, deployedVersion, deployedCatalog)
	fmt.Fprint(r.stdout, output)
	return nil
}

func (r *runner) listConfigs(ctx context.Context, ctrlClient client.Client) error {
	var gitRepoList *unstructured.UnstructuredList

	err := RunWithSpinner("Listing config repositories", func() error {
		gvk := schema.GroupVersionKind{
			Group:   "source.toolkit.fluxcd.io",
			Version: "v1",
			Kind:    "GitRepository",
		}

		gitRepoList = &unstructured.UnstructuredList{}
		gitRepoList.SetGroupVersionKind(gvk)

		listOptions := &client.ListOptions{
			Namespace: r.flag.Namespace,
		}

		return ctrlClient.List(ctx, gitRepoList, listOptions)
	})

	if err != nil {
		return err
	}

	if len(gitRepoList.Items) == 0 {
		fmt.Fprintf(r.stdout, "No config repositories found in namespace %s\n", r.flag.Namespace)
		return nil
	}

	output := ListConfigsOutput(gitRepoList, r.flag.Namespace)
	fmt.Fprint(r.stdout, output)
	return nil
}

func (r *runner) listCatalogs(ctx context.Context, ctrlClient client.Client) error {
	var catalogs *applicationv1alpha1.CatalogList

	err := RunWithSpinner("Listing catalogs", func() error {
		catalogs = &applicationv1alpha1.CatalogList{}
		// List catalogs in both default and giantswarm namespaces
		return ctrlClient.List(ctx, catalogs, &client.ListOptions{})
	})

	if err != nil {
		return err
	}

	if len(catalogs.Items) == 0 {
		fmt.Fprintf(r.stdout, "No catalogs found\n")
		return nil
	}

	output := ListCatalogsOutput(catalogs)
	fmt.Fprint(r.stdout, output)
	return nil
}

func (r *runner) getCatalogService(ctrlClient client.Client) (catalog.Interface, error) {
	// Create a new catalog data service instance
	catalogConfig := catalog.Config{
		Client: ctrlClient,
	}
	catalogDataService, err := catalog.New(catalogConfig)
	if err != nil {
		return nil, err
	}

	return catalogDataService, nil
}
