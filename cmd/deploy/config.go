package deploy

import (
	"context"
	"fmt"
	"strings"

	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	k8smetadataAnnotation "github.com/giantswarm/k8smetadata/pkg/annotation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// isSuspended checks if a resource has Flux reconciliation suspended
func isSuspended(obj metav1.Object) bool {
	annotations := obj.GetAnnotations()
	if annotations != nil {
		if val, ok := annotations[k8smetadataAnnotation.FluxKustomizeReconcile]; ok && val == "disabled" {
			return true
		}
	}
	labels := obj.GetLabels()
	if labels != nil {
		if val, ok := labels[k8smetadataAnnotation.FluxKustomizeReconcile]; ok && val == "disabled" {
			return true
		}
	}
	return false
}

func (r *runner) deployConfig(ctx context.Context, spec *resourceSpec) error {
	var gitRepo *sourcev1.GitRepository
	var resourceName, resourceNamespace string

	// Find and patch the GitRepository with spinner
	err := RunWithSpinner(fmt.Sprintf("Deploying config repository %s@%s", spec.name, spec.version), func() error {
		var findErr error
		gitRepo, findErr = r.findGitRepository(ctx, spec.name, r.flag.Namespace)
		if findErr != nil {
			return fmt.Errorf("failed to find GitRepository for %s: %w", spec.name, findErr)
		}

		// Get the resource name and namespace
		resourceName = gitRepo.Name
		resourceNamespace = gitRepo.Namespace

		// Create a patch to set the reconcile annotation and update the branch
		patch := client.MergeFrom(gitRepo.DeepCopy())

		// Add the Flux reconcile annotation to suspend reconciliation
		annotations := gitRepo.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations[k8smetadataAnnotation.FluxKustomizeReconcile] = "disabled"
		gitRepo.SetAnnotations(annotations)

		// Update the spec.ref.branch to the desired version (branch)
		if gitRepo.Spec.Reference == nil {
			gitRepo.Spec.Reference = &sourcev1.GitRepositoryRef{}
		}
		gitRepo.Spec.Reference.Branch = spec.version

		// Apply the patch
		patchErr := r.ctrlClient.Patch(ctx, gitRepo, patch)
		if patchErr != nil {
			return fmt.Errorf("failed to patch GitRepository: %w", patchErr)
		}

		return nil
	})

	if err != nil {
		return err
	}

	output := DeployOutput("config", resourceName, spec.version, resourceNamespace, !r.flag.UndeployOnExit)
	fmt.Fprint(r.stdout, output)
	return nil
}

func (r *runner) undeployConfig(ctx context.Context, spec *resourceSpec) error {
	var resourceName, resourceNamespace string

	err := RunWithSpinner(fmt.Sprintf("Undeploying config repository %s", spec.name), func() error {
		// Find the GitRepository by matching its URL against the config repo name
		gitRepo, findErr := r.findGitRepository(ctx, spec.name, r.flag.Namespace)
		if findErr != nil {
			return fmt.Errorf("failed to find GitRepository for %s: %w", spec.name, findErr)
		}

		// Get the resource name and namespace
		resourceName = gitRepo.Name
		resourceNamespace = gitRepo.Namespace

		// Create a patch to remove the reconcile annotation and label
		patch := client.MergeFrom(gitRepo.DeepCopy())

		// Remove the Flux reconcile annotation
		annotations := gitRepo.GetAnnotations()
		if annotations != nil {
			delete(annotations, k8smetadataAnnotation.FluxKustomizeReconcile)
			gitRepo.SetAnnotations(annotations)
		}

		// Remove the Flux reconcile label if present
		labels := gitRepo.GetLabels()
		if labels != nil {
			delete(labels, k8smetadataAnnotation.FluxKustomizeReconcile)
			gitRepo.SetLabels(labels)
		}

		// Apply the patch
		patchErr := r.ctrlClient.Patch(ctx, gitRepo, patch)
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

// findGitRepository finds a GitRepository CR by matching the config repo name in its URL
func (r *runner) findGitRepository(ctx context.Context, configRepoName, namespace string) (*sourcev1.GitRepository, error) {
	// List all GitRepository resources in the namespace
	gitRepoList := &sourcev1.GitRepositoryList{}

	listOptions := &client.ListOptions{
		Namespace: namespace,
	}

	err := r.ctrlClient.List(ctx, gitRepoList, listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list GitRepositories: %w", err)
	}

	// Find the GitRepository that matches the config repo name in its URL
	for i := range gitRepoList.Items {
		gitRepo := &gitRepoList.Items[i]
		url := gitRepo.Spec.URL

		// Check if the URL contains the config repo name
		// The URL format is typically: https://github.com/giantswarm/{configRepoName}
		if strings.Contains(url, fmt.Sprintf("giantswarm/%s", configRepoName)) {
			return gitRepo, nil
		}
	}

	return nil, fmt.Errorf("%w: GitRepository for config repo %s not found in namespace %s", ErrResourceNotFound, configRepoName, namespace)
}

// checkGitRepositories checks if any GitRepository resources have Flux reconciliation suspended
func (r *runner) checkGitRepositories(ctx context.Context) ([]resourceInfo, error) {
	gitRepoList := &sourcev1.GitRepositoryList{}

	err := r.ctrlClient.List(ctx, gitRepoList)
	if err != nil {
		return nil, err
	}

	suspendedRepos := []resourceInfo{}

	for i := range gitRepoList.Items {
		gitRepo := &gitRepoList.Items[i]
		if isSuspended(gitRepo) {
			branch := ""
			if gitRepo.Spec.Reference != nil {
				branch = gitRepo.Spec.Reference.Branch
			}

			// Get status from Ready condition
			status := getGitRepoStatusFromConditions(gitRepo)

			suspendedRepos = append(suspendedRepos, resourceInfo{
				name:      gitRepo.Name,
				namespace: gitRepo.Namespace,
				branch:    branch,
				url:       gitRepo.Spec.URL,
				status:    status,
			})
		}
	}

	return suspendedRepos, nil
}

// getGitRepoStatusFromConditions extracts status from GitRepository conditions
func getGitRepoStatusFromConditions(gitRepo *sourcev1.GitRepository) string {
	for _, cond := range gitRepo.Status.Conditions {
		if cond.Type == "Ready" {
			if cond.Status == metav1.ConditionTrue {
				return "Ready"
			}
			if cond.Reason != "" {
				return cond.Reason
			}
			return "Not Ready"
		}
	}
	return "Unknown"
}

func (r *runner) listConfigs(ctx context.Context) error {
	var gitRepoList *sourcev1.GitRepositoryList

	err := RunWithSpinner("Listing config repositories", func() error {
		gitRepoList = &sourcev1.GitRepositoryList{}

		listOptions := &client.ListOptions{
			Namespace: r.flag.Namespace,
		}

		return r.ctrlClient.List(ctx, gitRepoList, listOptions)
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
