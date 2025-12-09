package deploy

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(colorInfo)

	// ansiRegex matches ANSI escape codes
	ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)
)

// visibleWidth returns the display width of a string, excluding ANSI codes
func visibleWidth(s string) int {
	return len(ansiRegex.ReplaceAllString(s, ""))
}

// tableBuilder helps build formatted tables
type tableBuilder struct {
	headers []string
	rows    [][]string
	indent  string
}

func newTable(headers ...string) *tableBuilder {
	return &tableBuilder{
		headers: headers,
		rows:    [][]string{},
		indent:  "  ",
	}
}

func (t *tableBuilder) addRow(values ...string) {
	t.rows = append(t.rows, values)
}

func (t *tableBuilder) render() string {
	if len(t.rows) == 0 {
		return ""
	}

	// Calculate max widths for each column
	colWidths := make([]int, len(t.headers))
	for i, header := range t.headers {
		colWidths[i] = visibleWidth(header)
	}
	for _, row := range t.rows {
		for i, cell := range row {
			cellWidth := visibleWidth(cell)
			if i < len(colWidths) && cellWidth > colWidths[i] {
				colWidths[i] = cellWidth
			}
		}
	}

	var b strings.Builder

	// Render header
	headerParts := make([]string, len(t.headers))
	for i, header := range t.headers {
		headerVisWidth := visibleWidth(header)
		paddingNeeded := colWidths[i] - headerVisWidth
		if paddingNeeded > 0 {
			headerParts[i] = header + strings.Repeat(" ", paddingNeeded)
		} else {
			headerParts[i] = header
		}
	}
	b.WriteString(t.indent + headerStyle.Render(strings.Join(headerParts, "  ")) + "\n")

	// Render separator
	totalWidth := 0
	for i, width := range colWidths {
		totalWidth += width
		if i < len(colWidths)-1 {
			totalWidth += 2 // for "  " spacing
		}
	}
	b.WriteString(t.indent + mutedStyle.Render(strings.Repeat("─", totalWidth)) + "\n")

	// Render rows
	for _, row := range t.rows {
		rowParts := make([]string, len(row))
		for i, cell := range row {
			if i < len(colWidths) {
				// Calculate padding needed based on visible width
				cellVisWidth := visibleWidth(cell)
				paddingNeeded := colWidths[i] - cellVisWidth
				if paddingNeeded > 0 {
					rowParts[i] = cell + strings.Repeat(" ", paddingNeeded)
				} else {
					rowParts[i] = cell
				}
			}
		}
		b.WriteString(t.indent + strings.Join(rowParts, "  ") + "\n")
	}

	return b.String()
}

// DeployOutput renders a formatted deploy success message
func DeployOutput(kind, name, version, namespace string) string {
	var b strings.Builder

	// Header
	b.WriteString(successStyle.Render("✓ Deployment Successful") + "\n\n")

	// Details box
	details := fmt.Sprintf(
		"%s: %s/%s\n%s: %s\n%s: %s",
		infoStyle.Render("Resource"),
		kind,
		name,
		infoStyle.Render("Version"),
		version,
		infoStyle.Render("Namespace"),
		namespace,
	)
	b.WriteString(boxStyle.Render(details) + "\n")

	return b.String()
}

// ReminderOutput renders the undeploy reminder
func ReminderOutput(resourceType, name string) string {
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
	return reminderStyle.Render(reminder) + "\n"
}

// UpdateOutput renders a formatted update success message
func UpdateOutput(kind, name, namespace string, changes []string) string {
	var b strings.Builder

	// Header
	b.WriteString(successStyle.Render("✓ Update Successful") + "\n")

	// Details
	details := fmt.Sprintf(
		"%s: %s/%s\n%s: %s",
		infoStyle.Render("Resource"),
		kind,
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

// UndeployOutput renders a formatted undeploy success message
func UndeployOutput(kind, name, namespace string, changes []string) string {
	var b strings.Builder

	// Header
	b.WriteString(successStyle.Render("✓ Undeployment Successful") + "\n\n")

	// Details
	details := fmt.Sprintf(
		"%s: %s/%s\n%s: %s",
		infoStyle.Render("Resource"),
		kind,
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
		table := newTable("NAME", "NAMESPACE")
		for _, kust := range suspendedKustomizations {
			table.addRow(kust.name, kust.namespace)
		}
		b.WriteString(table.render() + "\n")
	}

	// Suspended apps
	if len(suspendedApps) > 0 {
		b.WriteString(warningStyle.Render("⚠ Suspended Apps:") + "\n\n")
		table := newTable("NAME", "NAMESPACE", "VERSION", "CATALOG", "STATUS")
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
			table.addRow(app.name, app.namespace, version, catalog, colorizeStatus(status))
		}
		b.WriteString(table.render() + "\n")
	}

	// Suspended git repositories
	if len(suspendedGitRepos) > 0 {
		b.WriteString(warningStyle.Render("⚠ Suspended Git Repositories:") + "\n\n")
		table := newTable("NAME", "NAMESPACE", "BRANCH", "URL", "STATUS")
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
			table.addRow(repo.name, repo.namespace, branch, url, colorizeStatus(status))
		}
		b.WriteString(table.render() + "\n")
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
func ListAppsOutput(apps []appInfo, namespace string, catalog string, installedOnly bool) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("📦 Applications") + "\n")
	if installedOnly {
		b.WriteString(mutedStyle.Render(fmt.Sprintf("Catalog: %s | Namespace: %s (installed only)", catalog, namespace)) + "\n\n")
	} else {
		b.WriteString(mutedStyle.Render(fmt.Sprintf("Catalog: %s | Namespace: %s", catalog, namespace)) + "\n\n")
	}

	if len(apps) == 0 {
		b.WriteString(warningStyle.Render("No apps found") + "\n")
		return b.String()
	}

	// Sort apps by name
	sortedApps := make([]appInfo, len(apps))
	copy(sortedApps, apps)
	sort.Slice(sortedApps, func(i, j int) bool {
		return sortedApps[i].name < sortedApps[j].name
	})

	// Build table
	var table *tableBuilder
	if installedOnly {
		table = newTable("NAME", "VERSION", "CATALOG", "STATUS")
	} else {
		table = newTable("NAME", "INSTALLED", "VERSION", "CATALOG", "STATUS")
	}

	installedCount := 0
	for _, app := range sortedApps {
		version := app.version
		if version == "" {
			version = "-"
		}
		catalog := app.catalog
		if catalog == "" {
			catalog = "-"
		}
		status := colorizeStatus(app.status)

		if installedOnly {
			// When showing installed only, all apps are installed
			table.addRow(app.name, version, catalog, status)
			installedCount++
		} else {
			// Show installation status
			installedStatus := "No"
			if app.installed {
				installedStatus = successStyle.Render("Yes")
				installedCount++
			} else {
				installedStatus = mutedStyle.Render("No")
			}
			table.addRow(app.name, installedStatus, version, catalog, status)
		}
	}

	b.WriteString(table.render())
	if installedOnly {
		b.WriteString("\n" + mutedStyle.Render(fmt.Sprintf("Total: %d apps", len(apps))) + "\n")
	} else {
		b.WriteString("\n" + mutedStyle.Render(fmt.Sprintf("Total: %d apps (%d installed)", len(apps), installedCount)) + "\n")
	}
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
func ListConfigsOutput(gitRepoList *sourcev1.GitRepositoryList, namespace string) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("⚙️  Config Repositories") + "\n")
	b.WriteString(mutedStyle.Render(fmt.Sprintf("Namespace: %s", namespace)) + "\n\n")

	if len(gitRepoList.Items) == 0 {
		b.WriteString(warningStyle.Render("No config repositories found") + "\n")
		return b.String()
	}

	// Sort by name
	sortedRepos := make([]sourcev1.GitRepository, len(gitRepoList.Items))
	copy(sortedRepos, gitRepoList.Items)
	sort.Slice(sortedRepos, func(i, j int) bool {
		return sortedRepos[i].Name < sortedRepos[j].Name
	})

	// Build table
	table := newTable("NAME", "BRANCH", "URL", "STATUS")
	for i := range sortedRepos {
		repo := &sortedRepos[i]
		branch := ""
		if repo.Spec.Reference != nil {
			branch = repo.Spec.Reference.Branch
		}
		if branch == "" {
			branch = "-"
		}
		status := colorizeStatus(getGitRepoStatus(repo))
		table.addRow(repo.Name, branch, repo.Spec.URL, status)
	}

	b.WriteString(table.render())
	b.WriteString("\n" + mutedStyle.Render(fmt.Sprintf("Total: %d repositories", len(gitRepoList.Items))) + "\n")
	return b.String()
}

// ListCatalogsOutput renders a formatted list of catalogs
func ListCatalogsOutput(catalogList *applicationv1alpha1.CatalogList) string {
	var b strings.Builder

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

	// Build table
	table := newTable("NAME", "NAMESPACE", "URL")
	for _, cat := range sortedCatalogs {
		url := getCatalogURL(&cat)
		table.addRow(cat.Name, cat.Namespace, url)
	}

	b.WriteString(table.render())
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
func getGitRepoStatus(repo *sourcev1.GitRepository) string {
	for _, cond := range repo.Status.Conditions {
		if cond.Type == "Ready" {
			if cond.Status == metav1.ConditionTrue {
				return "Ready"
			}
			// Not ready - show the reason
			if cond.Reason != "" {
				return cond.Reason
			}
			if cond.Message != "" {
				return cond.Message
			}
			return "Not Ready"
		}
	}
	return "Unknown"
}

// colorizeStatus applies color to status text based on the status value
func colorizeStatus(status string) string {
	// Empty or dash status (muted)
	if status == "-" || status == "" {
		return mutedStyle.Render("-")
	}

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
