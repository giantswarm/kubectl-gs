package deploy

import (
	"context"
	"fmt"

	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/kubectl-gs/v5/pkg/data/domain/app"
)

func (r *runner) deployApp(ctx context.Context, spec *resourceSpec) error {
	// Try to get existing app to determine if we need to create or update
	_, err := r.appService.GetApp(ctx, r.flag.Namespace, spec.name)
	if err != nil {
		// App doesn't exist, create it
		if app.IsNotFound(err) {
			err = RunWithSpinner(fmt.Sprintf("Deploying app %s@%s", spec.name, spec.version), func() error {
				createOptions := app.CreateOptions{
					Name:         spec.name,
					Namespace:    r.flag.Namespace,
					AppName:      spec.name,
					AppNamespace: r.flag.Namespace,
					AppCatalog:   r.flag.Catalog,
					AppVersion:   spec.version,
				}

				_, createErr := r.appService.Create(ctx, createOptions)
				return createErr
			})

			if app.IsNoResources(err) {
				return fmt.Errorf("no app with the name %s and the version %s found in the catalog", spec.name, spec.version)
			} else if err != nil {
				return err
			}

			output := DeployOutput("app", spec.name, spec.version, r.flag.Namespace)
			fmt.Fprint(r.stdout, output)

			// Trigger flux reconciliation if --sync flag is set
			if err := r.reconcileFluxApp(ctx, spec.name, r.flag.Namespace); err != nil {
				return err
			}

			// Show reminder last if not using --undeploy-on-exit
			if !r.flag.UndeployOnExit {
				fmt.Fprint(r.stdout, ReminderOutput("app", spec.name))
			}

			return nil
		}
		return err
	}

	// App exists, use the app service to patch it with version validation
	var state []string
	err = RunWithSpinner(fmt.Sprintf("Updating app %s to version %s", spec.name, spec.version), func() error {
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

	// Trigger flux reconciliation if --sync flag is set
	if err := r.reconcileFluxApp(ctx, spec.name, r.flag.Namespace); err != nil {
		return err
	}

	// Show reminder last if not using --undeploy-on-exit
	if !r.flag.UndeployOnExit {
		fmt.Fprint(r.stdout, ReminderOutput("app", spec.name))
	}

	return nil
}

func (r *runner) undeployApp(ctx context.Context, spec *resourceSpec) error {
	var state []string
	err := RunWithSpinner(fmt.Sprintf("Undeploying app %s", spec.name), func() error {
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

// checkApps checks if any apps have Flux reconciliation suspended
func (r *runner) checkApps(ctx context.Context) ([]resourceInfo, error) {
	// List all apps across all namespaces by passing empty namespace
	apps, err := r.appService.ListApps(ctx, "")
	if err != nil {
		return nil, err
	}

	suspendedApps := []resourceInfo{}

	for _, app := range apps.Items {
		if isSuspended(&app) {
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

// appInfo represents an app with its installation status
type appInfo struct {
	name            string
	version         string
	catalog         string
	status          string
	installed       bool
	installedInNs   string
	availableInCatalog bool
}

func (r *runner) listApps(ctx context.Context) error {
	var installedApps *applicationv1alpha1.AppList
	var catalogEntries *applicationv1alpha1.AppCatalogEntryList

	// Fetch both catalog entries and installed apps
	err := RunWithSpinner("Listing apps", func() error {
		// Get catalog entries
		catalogDataService, serviceErr := r.getCatalogService()
		if serviceErr != nil {
			return serviceErr
		}

		selector := fmt.Sprintf("application.giantswarm.io/catalog=%s", r.flag.Catalog)
		var listErr error
		catalogEntries, listErr = catalogDataService.GetEntries(ctx, selector)
		if listErr != nil {
			return listErr
		}

		// Get installed apps in the namespace
		installedApps, listErr = r.appService.ListApps(ctx, r.flag.Namespace)
		if listErr != nil && !app.IsNoResources(listErr) {
			return listErr
		}

		return nil
	})

	if err != nil {
		return err
	}

	if len(catalogEntries.Items) == 0 {
		fmt.Fprintf(r.stdout, "No apps found in catalog %s\n", r.flag.Catalog)
		return nil
	}

	// Group catalog entries by app name to get the latest version
	latestVersions := make(map[string]applicationv1alpha1.AppCatalogEntry)
	for _, entry := range catalogEntries.Items {
		appName := entry.Spec.AppName
		if existing, exists := latestVersions[appName]; !exists || entry.Spec.Version > existing.Spec.Version {
			latestVersions[appName] = entry
		}
	}

	// Build a map of installed apps for quick lookup
	installedMap := make(map[string]*applicationv1alpha1.App)
	if installedApps != nil {
		for i := range installedApps.Items {
			installedApp := &installedApps.Items[i]
			installedMap[installedApp.Spec.Name] = installedApp
		}
	}

	// Build app info list from catalog entries and mark installed ones
	appInfoList := []appInfo{}
	for _, entry := range latestVersions {
		appName := entry.Spec.AppName
		info := appInfo{
			name:      appName,
			version:   entry.Spec.Version,
			catalog:   entry.Spec.Catalog.Name,
			installed: false,
		}

		// Check if this app is installed from the same catalog
		if installedApp, exists := installedMap[appName]; exists && installedApp.Spec.Catalog == r.flag.Catalog {
			info.installed = true
			info.installedInNs = installedApp.Namespace
			info.status = getAppStatus(installedApp)
		} else {
			info.status = "-"
		}

		// Filter if --installed-only is set
		if r.flag.InstalledOnly && !info.installed {
			continue
		}

		appInfoList = append(appInfoList, info)
	}

	if len(appInfoList) == 0 {
		if r.flag.InstalledOnly {
			fmt.Fprintf(r.stdout, "No installed apps found in namespace %s from catalog %s\n", r.flag.Namespace, r.flag.Catalog)
		} else {
			fmt.Fprintf(r.stdout, "No apps found in catalog %s\n", r.flag.Catalog)
		}
		return nil
	}

	output := ListAppsOutput(appInfoList, r.flag.Namespace, r.flag.Catalog, r.flag.InstalledOnly)
	fmt.Fprint(r.stdout, output)
	return nil
}

func (r *runner) listVersions(ctx context.Context, appName string) error {
	var entries *applicationv1alpha1.AppCatalogEntryList
	var deployedVersion string
	var deployedCatalog string
	err := RunWithSpinner(fmt.Sprintf("Listing versions for %s", appName), func() error {
		// Get catalog data service
		catalogDataService, serviceErr := r.getCatalogService()
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
		existingApp, getErr := r.appService.GetApp(ctx, r.flag.Namespace, appName)
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
