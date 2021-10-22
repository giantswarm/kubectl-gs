package app

import "k8s.io/apimachinery/pkg/runtime"

var testCases = []struct {
	name               string
	storage            []runtime.Object
	flags              flag
	expectedGoldenFile string
	errorMatcher       func(error) bool
	chartResponseCode  int
}{
	{
		name: "case 0: patch app with the latest AppCatalogEntry CR",
		storage: []runtime.Object{
			newApp("fake-app", "0.0.1", "fake-catalog"),
			newCatalog("fake-catalog"),
			newAppCatalogEntry("fake-app", "0.0.1", "fake-catalog", "false"),
			newAppCatalogEntry("fake-app", "0.1.0", "fake-catalog", "true"),
		},
		flags: flag{Name: "fake-app", Version: "0.1.0"},
	},
	{
		name: "case 1: patch app with the AppCatalogEntry CR (not latest)",
		storage: []runtime.Object{
			newApp("fake-app", "0.0.1", "fake-catalog"),
			newCatalog("fake-catalog"),
			newAppCatalogEntry("fake-app", "0.0.1", "fake-catalog", "false"),
			newAppCatalogEntry("fake-app", "0.1.0", "fake-catalog", "false"),
			newAppCatalogEntry("fake-app", "0.2.0", "fake-catalog", "true"),
		},
		flags: flag{Name: "fake-app", Version: "0.1.0"},
	},
	{
		name: "case 2: patch app without AppCatalogEntry CR, but available in catalog",
		storage: []runtime.Object{
			newApp("fake-app", "0.5.0", "fake-catalog"),
			newCatalog("fake-catalog"),
			newAppCatalogEntry("fake-app", "0.1.0", "fake-catalog", "false"),
			newAppCatalogEntry("fake-app", "0.2.0", "fake-catalog", "false"),
			newAppCatalogEntry("fake-app", "0.3.0", "fake-catalog", "false"),
			newAppCatalogEntry("fake-app", "0.4.0", "fake-catalog", "false"),
			newAppCatalogEntry("fake-app", "0.5.0", "fake-catalog", "true"),
		},
		flags:             flag{Name: "fake-app", Version: "0.0.1"},
		chartResponseCode: 200,
	},
	{
		name: "case 3: patch app with nonexisting version",
		storage: []runtime.Object{
			newApp("fake-app", "0.0.1", "fake-catalog"),
			newCatalog("fake-catalog"),
			newAppCatalogEntry("fake-app", "0.0.1", "fake-catalog", "true"),
		},
		flags:             flag{Name: "fake-app", Version: "0.1.0"},
		errorMatcher:      IsNoResources,
		chartResponseCode: 404,
	},
	{
		name:         "case 4: patch nonexisting app",
		storage:      []runtime.Object{},
		flags:        flag{Name: "bad-app", Version: "0.1.0"},
		errorMatcher: IsNotFound,
	},
}
