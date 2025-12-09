package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/sahilm/fuzzy"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	selectedItemStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170"))
	normalItemStyle    = lipgloss.NewStyle()
	promptStyleSelect  = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	matchStyleSelect   = lipgloss.NewStyle().Foreground(lipgloss.Color("211")).Underline(true)
	selectedMatchStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("211")).Underline(true)
)

// SelectionResult holds the result of the interactive selection
type SelectionResult struct {
	AppName    string
	Version    string
	Catalog    string
	ConfigRepo string
	Branch     string
	Canceled   bool
}

// catalogItem represents an item in the catalog selector
type catalogItem struct {
	name string
}

func (i catalogItem) String() string {
	return i.name
}

// catalogEntryItem represents an item in the catalog entry selector
type catalogEntryItem struct {
	appName   string
	version   string
	catalog   string
	namespace string
	date      string
	title     string // Combined string for display and fuzzy matching
}

func (i catalogEntryItem) String() string {
	return i.title
}

// configRepoItem represents a config repository in the selector
type configRepoItem struct {
	name   string
	url    string
	branch string
}

func (i configRepoItem) String() string {
	if i.branch != "" {
		return fmt.Sprintf("%-40s (current branch: %s)", i.name, i.branch)
	}
	return i.name
}

// configPRItem represents a PR for a config repository
type configPRItem struct {
	number      int
	title       string
	branch      string
	author      string
	displayText string
}

func (i configPRItem) String() string {
	return i.displayText
}

// configVersionItem represents a combined config repo + branch/PR for selection
type configVersionItem struct {
	configName    string
	branch        string
	prNumber      int
	prTitle       string
	author        string
	isDeployed    bool
	displayText   string
}

func (i configVersionItem) String() string {
	return i.displayText
}

type itemMatch struct {
	item      interface{}
	positions []int
}

// selectorModel is the bubbletea model for the interactive selector
type selectorModel struct {
	items    []interface{}
	filtered []itemMatch
	cursor   int
	filter   textinput.Model
	width    int
	height   int
	prompt   string
	selected interface{}
	err      error
}

func newSelectorModel(items []interface{}, prompt string) selectorModel {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 50

	// Initialize filtered list
	filtered := make([]itemMatch, len(items))
	for i, item := range items {
		filtered[i] = itemMatch{item: item, positions: nil}
	}

	return selectorModel{
		items:    items,
		filtered: filtered,
		cursor:   0,
		filter:   ti,
		prompt:   prompt,
	}
}

func (m selectorModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *selectorModel) filterItems() {
	query := m.filter.Value()
	if query == "" {
		m.filtered = make([]itemMatch, len(m.items))
		for i, item := range m.items {
			m.filtered[i] = itemMatch{item: item, positions: nil}
		}
	} else {
		// Use fuzzy matching library
		matches := fuzzy.FindFromNoSort(query, itemSource(m.items))
		m.filtered = make([]itemMatch, len(matches))
		for i, match := range matches {
			m.filtered[i] = itemMatch{
				item:      m.items[match.Index],
				positions: match.MatchedIndexes,
			}
		}
	}
	// Reset cursor if out of bounds
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 && len(m.filtered) > 0 {
		m.cursor = 0
	}
}

// itemSource implements fuzzy.Source interface
type itemSource []interface{}

func (s itemSource) String(i int) string {
	switch item := s[i].(type) {
	case catalogItem:
		return item.String()
	case catalogEntryItem:
		return item.String()
	case configRepoItem:
		return item.String()
	case configPRItem:
		return item.String()
	case configVersionItem:
		return item.String()
	default:
		return ""
	}
}

func (s itemSource) Len() int {
	return len(s)
}

func (m selectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			if len(m.filtered) > 0 && m.cursor >= 0 && m.cursor < len(m.filtered) {
				m.selected = m.filtered[m.cursor].item
			}
			return m, tea.Quit
		case "up", "ctrl+k":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil
		case "down", "ctrl+j":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		default:
			// Update the filter input
			m.filter, cmd = m.filter.Update(msg)
			m.filterItems()
			return m, cmd
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	}

	return m, nil
}

// highlightMatches highlights the matching characters at given positions
func highlightMatches(text string, positions []int, isSelected bool) string {
	if len(positions) == 0 {
		if isSelected {
			return selectedItemStyle.Render(text)
		}
		return normalItemStyle.Render(text)
	}

	// Create a set of matched positions for quick lookup
	matchedPos := make(map[int]bool)
	for _, pos := range positions {
		matchedPos[pos] = true
	}

	var result strings.Builder
	textRunes := []rune(text)

	for i, char := range textRunes {
		if matchedPos[i] {
			if isSelected {
				result.WriteString(selectedMatchStyle.Render(string(char)))
			} else {
				result.WriteString(matchStyleSelect.Render(string(char)))
			}
		} else {
			if isSelected {
				result.WriteString(selectedItemStyle.Render(string(char)))
			} else {
				result.WriteString(normalItemStyle.Render(string(char)))
			}
		}
	}

	return result.String()
}

func (m selectorModel) View() string {
	var s strings.Builder

	// Calculate how many items we can show
	availableLines := m.height - 2 // Reserve 2 lines for prompt

	// Determine which items to show (centered around cursor)
	start := 0
	end := len(m.filtered)

	if end > availableLines {
		// Center the view around the cursor
		start = m.cursor - availableLines/2
		if start < 0 {
			start = 0
		}
		end = start + availableLines
		if end > len(m.filtered) {
			end = len(m.filtered)
			start = end - availableLines
			if start < 0 {
				start = 0
			}
		}
	}

	// Add empty lines at the top to push items to bottom
	itemCount := end - start
	for i := itemCount; i < availableLines; i++ {
		s.WriteString("\n")
	}

	// Render items in reverse order (bottom-up, with first item closest to prompt)
	for i := end - 1; i >= start; i-- {
		var displayText string
		switch item := m.filtered[i].item.(type) {
		case catalogItem:
			displayText = item.String()
		case catalogEntryItem:
			displayText = item.String()
		case configRepoItem:
			displayText = item.String()
		case configPRItem:
			displayText = item.String()
		case configVersionItem:
			displayText = item.String()
		}
		s.WriteString(highlightMatches(displayText, m.filtered[i].positions, i == m.cursor))
		s.WriteString("\n")
	}

	// Render prompt at the bottom
	s.WriteString(promptStyleSelect.Render(m.prompt))
	s.WriteString(m.filter.View())

	return s.String()
}

// selectCatalog presents a catalog selector to the user and returns the selected catalog name
func (r *runner) selectCatalog(ctx context.Context) (string, error) {
	// Get all catalogs
	catalogs, err := r.listAllCatalogs(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list catalogs: %w", err)
	}

	if len(catalogs.Items) == 0 {
		return "", fmt.Errorf("no catalogs found")
	}

	// Convert to catalog items
	items := make([]interface{}, len(catalogs.Items))
	for i, catalog := range catalogs.Items {
		items[i] = catalogItem{name: catalog.Name}
	}

	// Create and run the selector
	model := newSelectorModel(items, "Select catalog > ")
	p := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("error running catalog selector: %w", err)
	}

	// Check if selection was made
	m := finalModel.(selectorModel)
	if m.selected == nil {
		return "", fmt.Errorf("no catalog selected")
	}

	catalogItem, ok := m.selected.(catalogItem)
	if !ok {
		return "", fmt.Errorf("invalid selection")
	}

	return catalogItem.name, nil
}

// selectCatalogEntry presents a catalog entry selector to the user
func (r *runner) selectCatalogEntry(ctx context.Context, appNameFilter string, catalogFilter string) (*SelectionResult, error) {
	// Get catalog data service
	catalogDataService, err := r.getCatalogService()
	if err != nil {
		return nil, fmt.Errorf("failed to create catalog service: %w", err)
	}

	// Build selector for listing catalog entries
	var selectors []string
	if appNameFilter != "" {
		selectors = append(selectors, fmt.Sprintf("app.kubernetes.io/name=%s", appNameFilter))
	}
	if catalogFilter != "" {
		selectors = append(selectors, fmt.Sprintf("application.giantswarm.io/catalog=%s", catalogFilter))
	}

	selector := strings.Join(selectors, ",")

	// List catalog entries
	entries, err := catalogDataService.GetEntries(ctx, selector)
	if err != nil {
		return nil, fmt.Errorf("failed to list catalog entries: %w", err)
	}

	if len(entries.Items) == 0 {
		return nil, fmt.Errorf("no catalog entries found")
	}

	// Sort entries by date (newest first in array, so after reverse rendering newest appears at bottom)
	sortedEntries := make([]applicationv1alpha1.AppCatalogEntry, len(entries.Items))
	copy(sortedEntries, entries.Items)
	slices.SortFunc(sortedEntries, func(a, b applicationv1alpha1.AppCatalogEntry) int {
		// Sort by date descending (newest first)
		if a.Spec.DateUpdated.Time.After(b.Spec.DateUpdated.Time) {
			return -1
		}
		if a.Spec.DateUpdated.Time.Before(b.Spec.DateUpdated.Time) {
			return 1
		}
		return 0
	})

	// Convert to catalog entry items
	items := make([]interface{}, len(sortedEntries))
	for i, entry := range sortedEntries {
		// Format: name version date [catalog]
		title := fmt.Sprintf("%-40s %-20s %s",
			entry.Spec.AppName,
			entry.Spec.Version,
			entry.Spec.DateUpdated.Format("2006-01-02 15:04:05"))

		items[i] = catalogEntryItem{
			appName:   entry.Spec.AppName,
			version:   entry.Spec.Version,
			catalog:   entry.Spec.Catalog.Name,
			namespace: entry.Namespace,
			date:      entry.Spec.DateUpdated.Format("2006-01-02 15:04:05"),
			title:     title,
		}
	}

	// Create and run thse selector
	model := newSelectorModel(items, "Select app and version ")
	p := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("error running catalog entry selector: %w", err)
	}

	// Check if selection was made
	m := finalModel.(selectorModel)
	if m.selected == nil {
		return &SelectionResult{Canceled: true}, nil
	}

	entryItem, ok := m.selected.(catalogEntryItem)
	if !ok {
		return nil, fmt.Errorf("invalid selection")
	}

	return &SelectionResult{
		AppName:  entryItem.appName,
		Version:  entryItem.version,
		Catalog:  entryItem.catalog,
		Canceled: false,
	}, nil
}

// listAllCatalogs gets all catalogs in the cluster
func (r *runner) listAllCatalogs(ctx context.Context) (*applicationv1alpha1.CatalogList, error) {
	catalogs := &applicationv1alpha1.CatalogList{}
	err := r.ctrlClient.List(ctx, catalogs)
	return catalogs, err
}

// selectConfigRepo presents a config repository selector to the user
func (r *runner) selectConfigRepo(ctx context.Context, configNameFilter string) (string, error) {
	// Get all config repositories
	gitRepoList, err := r.listAllConfigRepos(ctx, r.flag.Namespace)
	if err != nil {
		return "", fmt.Errorf("failed to list config repositories: %w", err)
	}

	if len(gitRepoList.Items) == 0 {
		return "", fmt.Errorf("no config repositories found in namespace %s", r.flag.Namespace)
	}

	// Convert to config repo items
	items := make([]interface{}, 0)
	for _, gitRepo := range gitRepoList.Items {
		// Extract config repo name from URL
		configName := extractConfigNameFromURL(gitRepo.Spec.URL)
		if configName == "" {
			continue
		}

		// Filter by name if provided
		if configNameFilter != "" && !strings.Contains(configName, configNameFilter) {
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
		if configNameFilter != "" {
			return "", fmt.Errorf("no config repositories found matching '%s'", configNameFilter)
		}
		return "", fmt.Errorf("no config repositories found")
	}

	// Create and run the selector
	model := newSelectorModel(items, "Select config repository > ")
	p := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("error running config repository selector: %w", err)
	}

	// Check if selection was made
	m := finalModel.(selectorModel)
	if m.selected == nil {
		return "", fmt.Errorf("no config repository selected")
	}

	repoItem, ok := m.selected.(configRepoItem)
	if !ok {
		return "", fmt.Errorf("invalid selection")
	}

	return repoItem.name, nil
}

// selectConfigPR presents a PR selector for a config repository
func (r *runner) selectConfigPR(ctx context.Context, configRepoName string, currentBranch string) (*SelectionResult, error) {
	// Find the GitRepository to get the GitHub URL
	gitRepo, err := r.findGitRepository(ctx, configRepoName, r.flag.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to find config repository: %w", err)
	}

	// Extract GitHub repo from URL
	repoURL := gitRepo.Spec.URL
	parts := strings.Split(repoURL, "github.com")
	if len(parts) != 2 {
		return nil, fmt.Errorf("failed to extract GitHub repository from URL: %s", repoURL)
	}

	path := parts[1]
	if strings.HasPrefix(path, ":") {
		if idx := strings.Index(path, "/"); idx != -1 {
			path = path[idx:]
		}
	}
	githubRepo := strings.TrimPrefix(path, "/")
	githubRepo = strings.TrimPrefix(githubRepo, ":")
	githubRepo = strings.TrimSuffix(githubRepo, ".git")

	// Fetch open PRs using gh CLI
	var prs []PRInfo
	err = RunWithSpinner(fmt.Sprintf("Fetching open PRs for %s", githubRepo), func() error {
		cmd := exec.CommandContext(ctx, "gh", "pr", "list",
			"--repo", githubRepo,
			"--state", "open",
			"--json", "number,title,headRefName,author,createdAt",
		)

		output, cmdErr := cmd.Output()
		if cmdErr != nil {
			return fmt.Errorf("failed to list PRs: %w", cmdErr)
		}

		jsonErr := json.Unmarshal(output, &prs)
		if jsonErr != nil {
			return fmt.Errorf("failed to parse PR list: %w", jsonErr)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if len(prs) == 0 {
		return nil, fmt.Errorf("no open PRs found for %s", configRepoName)
	}

	// Convert to PR items
	items := make([]interface{}, len(prs))
	for i, pr := range prs {
		displayText := fmt.Sprintf("PR #%-4d %-60s [%s by @%s]",
			pr.Number,
			pr.Title,
			pr.HeadRefName,
			pr.Author.Login)

		// Mark if this is the currently deployed branch
		if currentBranch != "" && pr.HeadRefName == currentBranch {
			displayText += " (deployed)"
		}

		items[i] = configPRItem{
			number:      pr.Number,
			title:       pr.Title,
			branch:      pr.HeadRefName,
			author:      pr.Author.Login,
			displayText: displayText,
		}
	}

	// Create and run the selector
	model := newSelectorModel(items, "Select PR/branch > ")
	p := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("error running PR selector: %w", err)
	}

	// Check if selection was made
	m := finalModel.(selectorModel)
	if m.selected == nil {
		return &SelectionResult{Canceled: true}, nil
	}

	prItem, ok := m.selected.(configPRItem)
	if !ok {
		return nil, fmt.Errorf("invalid selection")
	}

	return &SelectionResult{
		ConfigRepo: configRepoName,
		Branch:     prItem.branch,
		Canceled:   false,
	}, nil
}

// listAllConfigRepos lists all GitRepository resources in the namespace
func (r *runner) listAllConfigRepos(ctx context.Context, namespace string) (*sourcev1.GitRepositoryList, error) {
	gitRepoList := &sourcev1.GitRepositoryList{}
	listOptions := &client.ListOptions{
		Namespace: namespace,
	}
	err := r.ctrlClient.List(ctx, gitRepoList, listOptions)
	return gitRepoList, err
}

// extractConfigNameFromURL extracts the config repository name from its GitHub URL
func extractConfigNameFromURL(url string) string {
	if !strings.Contains(url, "giantswarm/") {
		return ""
	}
	parts := strings.Split(url, "giantswarm/")
	if len(parts) < 2 {
		return ""
	}
	name := parts[len(parts)-1]
	name = strings.TrimSuffix(name, ".git")
	return name
}

// selectConfigVersion presents a combined selector of all config repos and their PRs
func (r *runner) selectConfigVersion(ctx context.Context, configNameFilter string) (*SelectionResult, error) {
	// Get all config repositories
	gitRepoList, err := r.listAllConfigRepos(ctx, r.flag.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to list config repositories: %w", err)
	}

	if len(gitRepoList.Items) == 0 {
		return nil, fmt.Errorf("no config repositories found in namespace %s", r.flag.Namespace)
	}

	// Collect all versions (PRs) from all matching repos
	var items []interface{}

	err = RunWithSpinner("Fetching PRs from config repositories", func() error {
		for _, gitRepo := range gitRepoList.Items {
			configName := extractConfigNameFromURL(gitRepo.Spec.URL)
			if configName == "" {
				continue
			}

			// Filter by name if provided
			if configNameFilter != "" && !strings.Contains(configName, configNameFilter) {
				continue
			}

			currentBranch := ""
			if gitRepo.Spec.Reference != nil {
				currentBranch = gitRepo.Spec.Reference.Branch
			}

			// Extract GitHub repo from URL
			repoURL := gitRepo.Spec.URL
			parts := strings.Split(repoURL, "github.com")
			if len(parts) != 2 {
				continue
			}

			path := parts[1]
			if strings.HasPrefix(path, ":") {
				if idx := strings.Index(path, "/"); idx != -1 {
					path = path[idx:]
				}
			}
			githubRepo := strings.TrimPrefix(path, "/")
			githubRepo = strings.TrimPrefix(githubRepo, ":")
			githubRepo = strings.TrimSuffix(githubRepo, ".git")

			// Fetch PRs for this repo
			var prs []PRInfo
			cmd := exec.CommandContext(ctx, "gh", "pr", "list",
				"--repo", githubRepo,
				"--state", "open",
				"--json", "number,title,headRefName,author,createdAt",
			)

			output, cmdErr := cmd.Output()
			if cmdErr != nil {
				// Skip repos where we can't fetch PRs
				continue
			}

			jsonErr := json.Unmarshal(output, &prs)
			if jsonErr != nil {
				continue
			}

			// Add each PR as an item
			for _, pr := range prs {
				isDeployed := currentBranch != "" && pr.HeadRefName == currentBranch

				// Format: config-name branch [PR #XXX: title by @author]
				displayText := fmt.Sprintf("%-30s %-40s",
					configName,
					pr.HeadRefName)

				if isDeployed {
					displayText += " (deployed)"
				}

				items = append(items, configVersionItem{
					configName: configName,
					branch:     pr.HeadRefName,
					prNumber:   pr.Number,
					prTitle:    pr.Title,
					author:     pr.Author.Login,
					isDeployed: isDeployed,
					displayText: displayText,
				})
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		if configNameFilter != "" {
			return nil, fmt.Errorf("no open PRs found for config repositories matching '%s'", configNameFilter)
		}
		return nil, fmt.Errorf("no open PRs found for any config repository")
	}

	// Create and run the selector
	model := newSelectorModel(items, "Select config and branch > ")
	p := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("error running config version selector: %w", err)
	}

	// Check if selection was made
	m := finalModel.(selectorModel)
	if m.selected == nil {
		return &SelectionResult{Canceled: true}, nil
	}

	versionItem, ok := m.selected.(configVersionItem)
	if !ok {
		return nil, fmt.Errorf("invalid selection")
	}

	return &SelectionResult{
		ConfigRepo: versionItem.configName,
		Branch:     versionItem.branch,
		Canceled:   false,
	}, nil
}
