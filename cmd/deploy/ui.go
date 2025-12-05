package deploy

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"
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
	notReadyKustomizations []resourceInfo,
	suspendedKustomizations []resourceInfo,
	suspendedApps []resourceInfo,
	suspendedGitRepos []resourceInfo,
) string {
	var b strings.Builder

	// Header
	b.WriteString(titleStyle.Render("📊 Deployment Status") + "\n\n")

	// Overall health check
	allHealthy := kustomizationsReady && len(suspendedKustomizations) == 0 &&
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

	// Suspended kustomizations
	if len(suspendedKustomizations) > 0 {
		b.WriteString(warningStyle.Render("⚠ Suspended Kustomizations:") + "\n\n")
		
		// Calculate max widths for columns
		maxNameLen := 4      // "NAME"
		maxNamespaceLen := 9 // "NAMESPACE"

		for _, kust := range suspendedKustomizations {
			if len(kust.name) > maxNameLen {
				maxNameLen = len(kust.name)
			}
			if len(kust.namespace) > maxNamespaceLen {
				maxNamespaceLen = len(kust.namespace)
			}
		}

		// Table header
		headerStyle := lipgloss.NewStyle().Bold(true).Foreground(colorInfo)
		header := fmt.Sprintf("  %-*s  %-*s",
			maxNameLen, "NAME",
			maxNamespaceLen, "NAMESPACE",
		)
		b.WriteString(headerStyle.Render(header) + "\n")

		// Separator line
		b.WriteString("  " + mutedStyle.Render(strings.Repeat("─", maxNameLen+maxNamespaceLen+2)) + "\n")

		// Display kustomizations in table format
		for _, kust := range suspendedKustomizations {
			row := fmt.Sprintf("  %-*s  %-*s",
				maxNameLen, kust.name,
				maxNamespaceLen, kust.namespace,
			)
			b.WriteString(row + "\n")
		}
		b.WriteString("\n")
	}

	// Suspended apps
	if len(suspendedApps) > 0 {
		b.WriteString(warningStyle.Render("⚠ Suspended Apps:") + "\n\n")

		// Calculate max widths for columns
		maxNameLen := 4      // "NAME"
		maxNamespaceLen := 9 // "NAMESPACE"
		maxVersionLen := 7   // "VERSION"
		maxCatalogLen := 7   // "CATALOG"
		maxStatusLen := 6    // "STATUS"

		for _, app := range suspendedApps {
			if len(app.name) > maxNameLen {
				maxNameLen = len(app.name)
			}
			if len(app.namespace) > maxNamespaceLen {
				maxNamespaceLen = len(app.namespace)
			}
			if len(app.version) > maxVersionLen {
				maxVersionLen = len(app.version)
			}
			if len(app.catalog) > maxCatalogLen {
				maxCatalogLen = len(app.catalog)
			}
			if len(app.status) > maxStatusLen {
				maxStatusLen = len(app.status)
			}
		}

		// Table header
		headerStyle := lipgloss.NewStyle().Bold(true).Foreground(colorInfo)
		header := fmt.Sprintf("  %-*s  %-*s  %-*s  %-*s  %-*s",
			maxNameLen, "NAME",
			maxNamespaceLen, "NAMESPACE",
			maxVersionLen, "VERSION",
			maxCatalogLen, "CATALOG",
			maxStatusLen, "STATUS",
		)
		b.WriteString(headerStyle.Render(header) + "\n")

		// Separator line
		b.WriteString("  " + mutedStyle.Render(strings.Repeat("─", maxNameLen+maxNamespaceLen+maxVersionLen+maxCatalogLen+maxStatusLen+8)) + "\n")

		// Display apps in table format
		for _, app := range suspendedApps {
			version := app.version
			if version == "" {
				version = "-"
			}
			catalog := app.catalog
			if catalog == "" {
				catalog = "-"
			}
			status := app.status
			if status == "" {
				status = "Unknown"
			}
			statusColored := colorizeStatus(status)

			row := fmt.Sprintf("  %-*s  %-*s  %-*s  %-*s  ",
				maxNameLen, app.name,
				maxNamespaceLen, app.namespace,
				maxVersionLen, version,
				maxCatalogLen, catalog,
			)
			b.WriteString(row + statusColored + "\n")
		}
		b.WriteString("\n")
	}

	// Suspended git repositories
	if len(suspendedGitRepos) > 0 {
		b.WriteString(warningStyle.Render("⚠ Suspended Git Repositories:") + "\n\n")

		// Calculate max widths for columns
		maxNameLen := 4      // "NAME"
		maxNamespaceLen := 9 // "NAMESPACE"
		maxBranchLen := 6    // "BRANCH"
		maxURLLen := 3       // "URL"
		maxStatusLen := 6    // "STATUS"

		for _, repo := range suspendedGitRepos {
			if len(repo.name) > maxNameLen {
				maxNameLen = len(repo.name)
			}
			if len(repo.namespace) > maxNamespaceLen {
				maxNamespaceLen = len(repo.namespace)
			}
			if len(repo.branch) > maxBranchLen {
				maxBranchLen = len(repo.branch)
			}
			if len(repo.url) > maxURLLen {
				maxURLLen = len(repo.url)
			}
			if len(repo.status) > maxStatusLen {
				maxStatusLen = len(repo.status)
			}
		}

		// Table header
		headerStyle := lipgloss.NewStyle().Bold(true).Foreground(colorInfo)
		header := fmt.Sprintf("  %-*s  %-*s  %-*s  %-*s  %-*s",
			maxNameLen, "NAME",
			maxNamespaceLen, "NAMESPACE",
			maxBranchLen, "BRANCH",
			maxURLLen, "URL",
			maxStatusLen, "STATUS",
		)
		b.WriteString(headerStyle.Render(header) + "\n")

		// Separator line
		b.WriteString("  " + mutedStyle.Render(strings.Repeat("─", maxNameLen+maxNamespaceLen+maxBranchLen+maxURLLen+maxStatusLen+8)) + "\n")

		// Display repos in table format
		for _, repo := range suspendedGitRepos {
			branch := repo.branch
			if branch == "" {
				branch = "-"
			}
			url := repo.url
			if url == "" {
				url = "-"
			}
			status := repo.status
			if status == "" {
				status = "Unknown"
			}
			statusColored := colorizeStatus(status)

			row := fmt.Sprintf("  %-*s  %-*s  %-*s  %-*s  ",
				maxNameLen, repo.name,
				maxNamespaceLen, repo.namespace,
				maxBranchLen, branch,
				maxURLLen, url,
			)
			b.WriteString(row + statusColored + "\n")
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

	// Calculate max widths for columns
	maxNameLen := 4    // "NAME"
	maxVersionLen := 7 // "VERSION"
	maxCatalogLen := 7 // "CATALOG"
	maxStatusLen := 6  // "STATUS"

	for _, app := range sortedApps {
		if len(app.Name) > maxNameLen {
			maxNameLen = len(app.Name)
		}
		if len(app.Spec.Version) > maxVersionLen {
			maxVersionLen = len(app.Spec.Version)
		}
		if len(app.Spec.Catalog) > maxCatalogLen {
			maxCatalogLen = len(app.Spec.Catalog)
		}
		status := getAppStatus(&app)
		if len(status) > maxStatusLen {
			maxStatusLen = len(status)
		}
	}

	// Table header
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(colorInfo)
	header := fmt.Sprintf("%-*s  %-*s  %-*s  %-*s",
		maxNameLen, "NAME",
		maxVersionLen, "VERSION",
		maxCatalogLen, "CATALOG",
		maxStatusLen, "STATUS",
	)
	b.WriteString(headerStyle.Render(header) + "\n")

	// Separator line
	b.WriteString(mutedStyle.Render(strings.Repeat("─", maxNameLen+maxVersionLen+maxCatalogLen+maxStatusLen+6)) + "\n")

	// Display apps in table format
	for _, app := range sortedApps {
		version := app.Spec.Version
		if version == "" {
			version = "-"
		}
		catalog := app.Spec.Catalog
		if catalog == "" {
			catalog = "-"
		}
		status := getAppStatus(&app)
		statusColored := colorizeStatus(status)

		row := fmt.Sprintf("%-*s  %-*s  %-*s  %s",
			maxNameLen, app.Name,
			maxVersionLen, version,
			maxCatalogLen, catalog,
			statusColored,
		)
		b.WriteString(row + "\n")
	}

	b.WriteString("\n" + mutedStyle.Render(fmt.Sprintf("Total: %d apps", len(apps.Items))) + "\n")
	return b.String()
}

// ListVersionsOutput renders a formatted list of application versions
func ListVersionsOutput(appName string, entries *applicationv1alpha1.AppCatalogEntryList, deployedVersion string, deployedCatalog string) string {
	var b strings.Builder

	// Header
	b.WriteString(titleStyle.Render(fmt.Sprintf("📋 Versions for %s", appName)) + "\n\n")

	if len(entries.Items) == 0 {
		b.WriteString(warningStyle.Render("No versions found") + "\n")
		return b.String()
	}

	// Sort versions first by catalog (alphabetically), then by version (most recent first)
	sortedEntries := make([]applicationv1alpha1.AppCatalogEntry, len(entries.Items))
	copy(sortedEntries, entries.Items)
	sort.Slice(sortedEntries, func(i, j int) bool {
		// First sort by catalog name (ascending)
		if sortedEntries[i].Spec.Catalog.Name != sortedEntries[j].Spec.Catalog.Name {
			return sortedEntries[i].Spec.Catalog.Name < sortedEntries[j].Spec.Catalog.Name
		}
		// Then sort by version (descending)
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

		// Mark the deployed version (match both version and catalog)
		if deployedVersion != "" && entry.Spec.Version == deployedVersion &&
			deployedCatalog != "" && entry.Spec.Catalog.Name == deployedCatalog {
			versionInfo += " " + warningStyle.Render("(deployed)")
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

	// Calculate max widths for columns
	maxNameLen := 4   // "NAME"
	maxBranchLen := 6 // "BRANCH"
	maxURLLen := 3    // "URL"
	maxStatusLen := 6 // "STATUS"

	for _, repo := range sortedRepos {
		name, _, _ := unstructured.NestedString(repo.Object, "metadata", "name")
		url, _, _ := unstructured.NestedString(repo.Object, "spec", "url")
		branch, _, _ := unstructured.NestedString(repo.Object, "spec", "ref", "branch")

		if len(name) > maxNameLen {
			maxNameLen = len(name)
		}
		if len(branch) > maxBranchLen {
			maxBranchLen = len(branch)
		}
		if len(url) > maxURLLen {
			maxURLLen = len(url)
		}
		status := getGitRepoStatus(&repo)
		if len(status) > maxStatusLen {
			maxStatusLen = len(status)
		}
	}

	// Table header
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(colorInfo)
	header := fmt.Sprintf("%-*s  %-*s  %-*s  %-*s",
		maxNameLen, "NAME",
		maxBranchLen, "BRANCH",
		maxURLLen, "URL",
		maxStatusLen, "STATUS",
	)
	b.WriteString(headerStyle.Render(header) + "\n")

	// Separator line
	b.WriteString(mutedStyle.Render(strings.Repeat("─", maxNameLen+maxBranchLen+maxURLLen+maxStatusLen+6)) + "\n")

	// Display repos in table format
	for _, repo := range sortedRepos {
		name, _, _ := unstructured.NestedString(repo.Object, "metadata", "name")
		url, _, _ := unstructured.NestedString(repo.Object, "spec", "url")
		branch, _, _ := unstructured.NestedString(repo.Object, "spec", "ref", "branch")

		if branch == "" {
			branch = "-"
		}

		status := getGitRepoStatus(&repo)
		statusColored := colorizeStatus(status)

		row := fmt.Sprintf("%-*s  %-*s  %-*s  %s",
			maxNameLen, name,
			maxBranchLen, branch,
			maxURLLen, url,
			statusColored,
		)
		b.WriteString(row + "\n")
	}

	b.WriteString("\n" + mutedStyle.Render(fmt.Sprintf("Total: %d repositories", len(gitRepoList.Items))) + "\n")
	return b.String()
}

// ListCatalogsOutput renders a formatted list of catalogs
func ListCatalogsOutput(catalogList *applicationv1alpha1.CatalogList) string {
	var b strings.Builder

	// Header
	b.WriteString(titleStyle.Render("📚 Catalogs") + "\n\n")

	if len(catalogList.Items) == 0 {
		b.WriteString(warningStyle.Render("No catalogs found") + "\n")
		return b.String()
	}

	// Sort by name
	sortedCatalogs := make([]applicationv1alpha1.Catalog, len(catalogList.Items))
	copy(sortedCatalogs, catalogList.Items)
	sort.Slice(sortedCatalogs, func(i, j int) bool {
		return sortedCatalogs[i].Name < sortedCatalogs[j].Name
	})

	// Calculate max widths for columns
	maxNameLen := 4      // "NAME"
	maxNamespaceLen := 9 // "NAMESPACE"
	maxURLLen := 3       // "URL"

	for _, cat := range sortedCatalogs {
		if len(cat.Name) > maxNameLen {
			maxNameLen = len(cat.Name)
		}
		if len(cat.Namespace) > maxNamespaceLen {
			maxNamespaceLen = len(cat.Namespace)
		}

		url := getCatalogURL(&cat)
		if len(url) > maxURLLen {
			maxURLLen = len(url)
		}
	}

	// Table header
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(colorInfo)
	header := fmt.Sprintf("%-*s  %-*s  %-*s",
		maxNameLen, "NAME",
		maxNamespaceLen, "NAMESPACE",
		maxURLLen, "URL",
	)
	b.WriteString(headerStyle.Render(header) + "\n")

	// Separator line
	b.WriteString(mutedStyle.Render(strings.Repeat("─", maxNameLen+maxNamespaceLen+maxURLLen+4)) + "\n")

	// Display catalogs in table format
	for _, cat := range sortedCatalogs {
		url := getCatalogURL(&cat)

		row := fmt.Sprintf("%-*s  %-*s  %-*s",
			maxNameLen, cat.Name,
			maxNamespaceLen, cat.Namespace,
			maxURLLen, url,
		)
		b.WriteString(row + "\n")
	}

	b.WriteString("\n" + mutedStyle.Render(fmt.Sprintf("Total: %d catalogs", len(catalogList.Items))) + "\n")
	return b.String()
}

// getCatalogURL extracts the URL from a Catalog CR
func getCatalogURL(cat *applicationv1alpha1.Catalog) string {
	// Try the new way first (repositories)
	if len(cat.Spec.Repositories) > 0 {
		return cat.Spec.Repositories[0].URL
	}
	// Fall back to the legacy way
	if cat.Spec.Storage.URL != "" {
		return cat.Spec.Storage.URL
	}
	return "-"
}

// getAppStatus extracts the status from an App CR
func getAppStatus(app *applicationv1alpha1.App) string {
	// Check if app has release status
	status := app.Status.Release.Status
	if status == "" {
		return "Unknown"
	}

	return status
}

// getGitRepoStatus extracts the status from a GitRepository CR
func getGitRepoStatus(repo *unstructured.Unstructured) string {
	// Check for Ready condition in status
	conditions, found, _ := unstructured.NestedSlice(repo.Object, "status", "conditions")
	if !found {
		return "Unknown"
	}

	for _, cond := range conditions {
		condMap, ok := cond.(map[string]interface{})
		if !ok {
			continue
		}

		condType, _, _ := unstructured.NestedString(condMap, "type")
		if condType == "Ready" {
			status, _, _ := unstructured.NestedString(condMap, "status")
			if status == "True" {
				return "Ready"
			}
			// Not ready - show the reason
			reason, _, _ := unstructured.NestedString(condMap, "reason")
			if reason != "" {
				return reason
			}
			message, _, _ := unstructured.NestedString(condMap, "message")
			if message != "" {
				return message
			}
			return "Not Ready"
		}
	}

	return "Unknown"
}

// colorizeStatus applies color to status text based on the status value
func colorizeStatus(status string) string {
	// Success statuses
	if status == "deployed" || status == "Ready" {
		return successStyle.Render(status)
	}

	// Warning statuses
	statusLower := strings.ToLower(status)
	if status == "Unknown" || strings.Contains(statusLower, "pending") || strings.Contains(statusLower, "progressing") {
		return warningStyle.Render(status)
	}

	// Everything else is treated as an error
	return errorStyle.Render(status)
}
