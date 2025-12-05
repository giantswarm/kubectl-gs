package deploy

import (
	"fmt"
	"sort"
	"strings"

	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/charmbracelet/lipgloss"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var (
	// Color palette
	colorSuccess = lipgloss.Color("#5f875f")
	colorWarning = lipgloss.Color("#af8700")
	colorError   = lipgloss.Color("#af5f5f")
	colorInfo    = lipgloss.Color("#5f87af")
	colorMuted   = lipgloss.Color("#808080")

	// Styles
	successStyle = lipgloss.NewStyle().
			Foreground(colorSuccess).
			Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(colorWarning).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(colorInfo).
			Bold(true)

	mutedStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Underline(true).
			MarginBottom(1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorInfo).
			Padding(1, 2).
			MarginTop(1).
			MarginBottom(1)

	listItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	reminderStyle = lipgloss.NewStyle().
			Foreground(colorWarning).
			Bold(true).
			MarginTop(1)
)

// DeployOutput renders a formatted deploy success message
func DeployOutput(resourceType, name, version, namespace string) string {
	var b strings.Builder

	// Header
	b.WriteString(successStyle.Render("✓ Deployment Successful") + "\n\n")

	// Details box
	details := fmt.Sprintf(
		"%s: %s\n%s: %s\n%s: %s",
		infoStyle.Render("Resource"),
		name,
		infoStyle.Render("Version"),
		version,
		infoStyle.Render("Namespace"),
		namespace,
	)
	b.WriteString(boxStyle.Render(details) + "\n")

	// Reminder
	var undeployCmd string
	if resourceType == "config" {
		undeployCmd = fmt.Sprintf("kubectl gs deploy -t config -u %s", name)
	} else {
		undeployCmd = fmt.Sprintf("kubectl gs deploy -u %s", name)
	}

	reminder := fmt.Sprintf(
		"%s Don't forget to undeploy after testing:\n  %s",
		warningStyle.Render("⚠"),
		mutedStyle.Render(undeployCmd),
	)
	b.WriteString(reminderStyle.Render(reminder) + "\n")

	return b.String()
}

// UpdateOutput renders a formatted update success message
func UpdateOutput(name, namespace string, changes []string) string {
	var b strings.Builder

	// Header
	b.WriteString(successStyle.Render("✓ Update Successful") + "\n\n")

	// Details
	details := fmt.Sprintf(
		"%s: %s\n%s: %s",
		infoStyle.Render("Resource"),
		name,
		infoStyle.Render("Namespace"),
		namespace,
	)

	if len(changes) > 0 {
		details += "\n" + infoStyle.Render("Changes") + ":"
		for _, change := range changes {
			details += fmt.Sprintf("\n  • %s", change)
		}
	}

	b.WriteString(boxStyle.Render(details) + "\n")

	// Reminder
	reminder := fmt.Sprintf(
		"%s Don't forget to undeploy after testing:\n  %s",
		warningStyle.Render("⚠"),
		mutedStyle.Render(fmt.Sprintf("kubectl gs deploy -u %s", name)),
	)
	b.WriteString(reminderStyle.Render(reminder) + "\n")

	return b.String()
}

// UndeployOutput renders a formatted undeploy success message
func UndeployOutput(resourceType, name, namespace string, changes []string) string {
	var b strings.Builder

	// Header
	b.WriteString(successStyle.Render("✓ Undeployment Successful") + "\n\n")

	// Details
	details := fmt.Sprintf(
		"%s: %s\n%s: %s",
		infoStyle.Render("Resource"),
		name,
		infoStyle.Render("Namespace"),
		namespace,
	)

	if len(changes) > 0 {
		details += "\n" + infoStyle.Render("Changes") + ":"
		for _, change := range changes {
			details += fmt.Sprintf("\n  • %s", change)
		}
	}

	b.WriteString(boxStyle.Render(details) + "\n")

	return b.String()
}

// StatusOutput renders a formatted status display
func StatusOutput(
	kustomizationsReady bool,
	kustomizationsSuspended bool,
	notReadyKustomizations []resourceInfo,
	suspendedApps []resourceInfo,
	suspendedGitRepos []resourceInfo,
) string {
	var b strings.Builder

	// Header
	b.WriteString(titleStyle.Render("📊 Deployment Status") + "\n\n")

	// Overall health check
	allHealthy := kustomizationsReady && !kustomizationsSuspended &&
		len(suspendedApps) == 0 && len(suspendedGitRepos) == 0

	if allHealthy {
		healthMsg := successStyle.Render("✓ All Systems Healthy")
		b.WriteString(boxStyle.Render(healthMsg) + "\n")
		return b.String()
	}

	// Show issues
	b.WriteString(warningStyle.Render("⚠ Issues Detected") + "\n\n")

	// Not ready kustomizations
	if !kustomizationsReady && len(notReadyKustomizations) > 0 {
		b.WriteString(errorStyle.Render("✗ Not Ready Kustomizations:") + "\n")
		for _, k := range notReadyKustomizations {
			item := fmt.Sprintf("• %s/%s", k.namespace, k.name)
			if k.reason != "" {
				item += mutedStyle.Render(fmt.Sprintf(" (reason: %s)", k.reason))
			}
			b.WriteString(listItemStyle.Render(item) + "\n")
		}
		b.WriteString("\n")
	}

	// Suspended kustomizations warning
	if kustomizationsSuspended {
		b.WriteString(warningStyle.Render("⚠ Some Kustomizations are Suspended") + "\n\n")
	}

	// Suspended apps
	if len(suspendedApps) > 0 {
		b.WriteString(warningStyle.Render("⚠ Suspended Apps:") + "\n")
		for _, app := range suspendedApps {
			item := fmt.Sprintf("• %s/%s", app.namespace, app.name)
			b.WriteString(listItemStyle.Render(item) + "\n")
		}
		b.WriteString("\n")
	}

	// Suspended git repositories
	if len(suspendedGitRepos) > 0 {
		b.WriteString(warningStyle.Render("⚠ Suspended Git Repositories:") + "\n")
		for _, repo := range suspendedGitRepos {
			item := fmt.Sprintf("• %s/%s", repo.namespace, repo.name)
			b.WriteString(listItemStyle.Render(item) + "\n")
		}
		b.WriteString("\n")
	}

	return b.String()
}

// ErrorOutput renders a formatted error message
func ErrorOutput(err error) string {
	var b strings.Builder

	b.WriteString(errorStyle.Render("✗ Error") + "\n\n")
	b.WriteString(boxStyle.
		BorderForeground(colorError).
		Render(err.Error()) + "\n")

	return b.String()
}

// InfoOutput renders a formatted info message
func InfoOutput(message string) string {
	return infoStyle.Render("ℹ ") + message + "\n"
}

// ListAppsOutput renders a formatted list of applications
func ListAppsOutput(apps *applicationv1alpha1.AppList, namespace string) string {
	var b strings.Builder

	// Header
	b.WriteString(titleStyle.Render("📦 Applications") + "\n")
	b.WriteString(mutedStyle.Render(fmt.Sprintf("Namespace: %s", namespace)) + "\n\n")

	if len(apps.Items) == 0 {
		b.WriteString(warningStyle.Render("No apps found") + "\n")
		return b.String()
	}

	// Sort apps by name
	sortedApps := make([]applicationv1alpha1.App, len(apps.Items))
	copy(sortedApps, apps.Items)
	sort.Slice(sortedApps, func(i, j int) bool {
		return sortedApps[i].Name < sortedApps[j].Name
	})

	// Display apps
	for _, app := range sortedApps {
		version := app.Spec.Version
		if version == "" {
			version = mutedStyle.Render("(no version)")
		}
		catalog := app.Spec.Catalog
		if catalog == "" {
			catalog = mutedStyle.Render("(no catalog)")
		}

		appInfo := fmt.Sprintf(
			"%s %s\n  %s %s\n  %s %s",
			infoStyle.Render("•"),
			successStyle.Render(app.Name),
			mutedStyle.Render("Version:"),
			version,
			mutedStyle.Render("Catalog:"),
			catalog,
		)
		b.WriteString(appInfo + "\n\n")
	}

	b.WriteString(mutedStyle.Render(fmt.Sprintf("Total: %d apps", len(apps.Items))) + "\n")
	return b.String()
}

// ListVersionsOutput renders a formatted list of application versions
func ListVersionsOutput(appName string, entries *applicationv1alpha1.AppCatalogEntryList) string {
	var b strings.Builder

	// Header
	b.WriteString(titleStyle.Render(fmt.Sprintf("📋 Versions for %s", appName)) + "\n\n")

	if len(entries.Items) == 0 {
		b.WriteString(warningStyle.Render("No versions found") + "\n")
		return b.String()
	}

	// Sort versions (most recent first, assuming semver)
	sortedEntries := make([]applicationv1alpha1.AppCatalogEntry, len(entries.Items))
	copy(sortedEntries, entries.Items)
	sort.Slice(sortedEntries, func(i, j int) bool {
		return sortedEntries[i].Spec.Version > sortedEntries[j].Spec.Version
	})

	// Display versions
	for i, entry := range sortedEntries {
		versionInfo := fmt.Sprintf(
			"%s %s",
			infoStyle.Render("•"),
			entry.Spec.Version,
		)

		// Mark the latest version
		if i == 0 {
			versionInfo += " " + successStyle.Render("(latest)")
		}

		// Add catalog info if available
		if entry.Spec.Catalog.Name != "" {
			versionInfo += mutedStyle.Render(fmt.Sprintf(" [%s]", entry.Spec.Catalog.Name))
		}

		b.WriteString(listItemStyle.Render(versionInfo) + "\n")
	}

	b.WriteString("\n" + mutedStyle.Render(fmt.Sprintf("Total: %d versions", len(entries.Items))) + "\n")
	return b.String()
}

// ListConfigsOutput renders a formatted list of config repositories
func ListConfigsOutput(gitRepoList *unstructured.UnstructuredList, namespace string) string {
	var b strings.Builder

	// Header
	b.WriteString(titleStyle.Render("⚙️  Config Repositories") + "\n")
	b.WriteString(mutedStyle.Render(fmt.Sprintf("Namespace: %s", namespace)) + "\n\n")

	if len(gitRepoList.Items) == 0 {
		b.WriteString(warningStyle.Render("No config repositories found") + "\n")
		return b.String()
	}

	// Sort by name
	sortedRepos := make([]unstructured.Unstructured, len(gitRepoList.Items))
	copy(sortedRepos, gitRepoList.Items)
	sort.Slice(sortedRepos, func(i, j int) bool {
		nameI, _, _ := unstructured.NestedString(sortedRepos[i].Object, "metadata", "name")
		nameJ, _, _ := unstructured.NestedString(sortedRepos[j].Object, "metadata", "name")
		return nameI < nameJ
	})

	// Display repos
	for _, repo := range sortedRepos {
		name, _, _ := unstructured.NestedString(repo.Object, "metadata", "name")
		url, _, _ := unstructured.NestedString(repo.Object, "spec", "url")
		branch, _, _ := unstructured.NestedString(repo.Object, "spec", "ref", "branch")

		if branch == "" {
			branch = mutedStyle.Render("(no branch)")
		}

		repoInfo := fmt.Sprintf(
			"%s %s\n  %s %s\n  %s %s",
			infoStyle.Render("•"),
			successStyle.Render(name),
			mutedStyle.Render("URL:"),
			url,
			mutedStyle.Render("Branch:"),
			branch,
		)
		b.WriteString(repoInfo + "\n\n")
	}

	b.WriteString(mutedStyle.Render(fmt.Sprintf("Total: %d repositories", len(gitRepoList.Items))) + "\n")
	return b.String()
}
