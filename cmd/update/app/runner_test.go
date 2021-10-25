package app

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	applicationv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/giantswarm/kubectl-gs/pkg/data/domain/app"
	"github.com/giantswarm/kubectl-gs/pkg/output"
	"github.com/giantswarm/kubectl-gs/test/kubeconfig"
)

const (
	address = "127.0.0.1:63283"
)

func Test_run(t *testing.T) {
	var testCases = []struct {
		name              string
		storage           []runtime.Object
		flags             flag
		errorMatcher      func(error) bool
		chartResponseCode int
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

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var err error

			var server *httptest.Server
			if tc.chartResponseCode != 0 {
				server = httptest.NewUnstartedServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
					rw.WriteHeader(tc.chartResponseCode)
				}))

				l, err := net.Listen("tcp", address)
				if err != nil {
					panic(fmt.Sprintf("httptest: failed to listen on a port: %v", err))
				}
				server.Listener.Close()
				server.Listener = l
				server.Start()
			}
			ctx := context.TODO()

			fakeKubeConfig := kubeconfig.CreateFakeKubeConfig()

			flag := &tc.flags
			flag.print = genericclioptions.NewPrintFlags("").WithDefaultOutput(output.TypeDefault)
			flag.config = genericclioptions.NewTestConfigFlags().WithClientConfig(fakeKubeConfig)

			out := new(bytes.Buffer)
			runner := &runner{
				service: app.NewFakeService(tc.storage),
				flag:    flag,
				stdout:  out,
			}

			err = runner.run(ctx, nil, []string{})
			if server != nil {
				server.Close()
			}

			if tc.errorMatcher != nil {
				if !tc.errorMatcher(err) {
					t.Fatalf("error not matching expected matcher, got: %s", errors.Cause(err))
				}

				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}
		})
	}
}

func newApp(appName, appVersion, catalogName string) *applicationv1alpha1.App {
	c := &applicationv1alpha1.App{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "application.giantswarm.io/v1alpha1",
			Kind:       "App",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      appName,
			Namespace: "default",
		},
		Spec: applicationv1alpha1.AppSpec{
			Catalog: catalogName,
			Name:    appName,
			Version: appVersion,
		},
	}

	return c
}

func newAppCatalogEntry(appName, appVersion, catalogName, latest string) *applicationv1alpha1.AppCatalogEntry {
	c := &applicationv1alpha1.AppCatalogEntry{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "application.giantswarm.io/v1alpha1",
			Kind:       "AppCatalogEntry",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s-app-%s", catalogName, appName, appVersion),
			Namespace: "default",
			Labels: map[string]string{
				label.CatalogName:          catalogName,
				label.AppKubernetesName:    appName,
				label.AppKubernetesVersion: appVersion,
				"latest":                   latest,
			},
		},
		Spec: applicationv1alpha1.AppCatalogEntrySpec{
			AppName:    appName,
			AppVersion: appVersion,
			Catalog: applicationv1alpha1.AppCatalogEntrySpecCatalog{
				Name:      catalogName,
				Namespace: "default",
			},
			Version: appVersion,
		},
	}

	return c
}

func newCatalog(catalogName string) *applicationv1alpha1.Catalog {
	c := &applicationv1alpha1.Catalog{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "application.giantswarm.io/v1alpha1",
			Kind:       "Catalog",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      catalogName,
			Namespace: "default",
		},
		Spec: applicationv1alpha1.CatalogSpec{
			Storage: applicationv1alpha1.CatalogSpecStorage{
				Type: "helm",
				URL:  fmt.Sprintf("http://%s/", address),
			},
		},
	}

	return c
}
