package app

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclienttest"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client/fake" //nolint:staticcheck

	"github.com/giantswarm/kubectl-gs/v2/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/v2/pkg/data/domain/app"
	"github.com/giantswarm/kubectl-gs/v2/pkg/output"
	"github.com/giantswarm/kubectl-gs/v2/pkg/scheme"
	"github.com/giantswarm/kubectl-gs/v2/test/kubeconfig"
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
		message           string
	}{
		{
			name: "patch app with the latest AppCatalogEntry CR",
			storage: []runtime.Object{
				newApp("fake-app", "0.0.1", "fake-catalog"),
				newCatalog("fake-catalog"),
				newAppCatalogEntry("fake-app", "0.0.1", "fake-catalog", "false"),
				newAppCatalogEntry("fake-app", "0.1.0", "fake-catalog", "true"),
			},
			flags:   flag{Name: "fake-app", Version: "0.1.0"},
			message: "App 'fake-app' updated to version '0.1.0'\n",
		},
		{
			name: "patch app with the AppCatalogEntry CR (not latest)",
			storage: []runtime.Object{
				newApp("fake-app", "0.0.1", "fake-catalog"),
				newCatalog("fake-catalog"),
				newAppCatalogEntry("fake-app", "0.0.1", "fake-catalog", "false"),
				newAppCatalogEntry("fake-app", "0.1.0", "fake-catalog", "false"),
				newAppCatalogEntry("fake-app", "0.2.0", "fake-catalog", "true"),
			},
			flags:   flag{Name: "fake-app", Version: "0.1.0"},
			message: "App 'fake-app' updated to version '0.1.0'\n",
		},
		{
			name: "patch app without AppCatalogEntry CR, but available in catalog",
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
			message:           "App 'fake-app' updated to version '0.0.1'\n",
		},
		{
			name: "patch app with nonexisting version",
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
			name:         "patch nonexisting app",
			storage:      []runtime.Object{},
			flags:        flag{Name: "bad-app", Version: "0.1.0"},
			errorMatcher: IsNotFound,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("case %d: %s", i, tc.name), func(t *testing.T) {
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

			out := new(bytes.Buffer)
			runner := &runner{
				commonConfig: commonconfig.New(genericclioptions.NewTestConfigFlags().WithClientConfig(fakeKubeConfig)),
				service:      newAppService(t, tc.storage...),
				flag:         flag,
				stdout:       out,
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

			diff := cmp.Diff(tc.message, out.String())
			if diff != "" {
				t.Fatalf("value not expected, got:\n %s", diff)
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
			Repositories: []applicationv1alpha1.CatalogSpecRepository{
				{
					Type: "helm",
					URL:  fmt.Sprintf("http://%s/", address),
				},
			},
		},
	}

	return c
}

func newAppService(t *testing.T, object ...runtime.Object) *app.Service {
	clientScheme, err := scheme.NewScheme()
	if err != nil {
		t.Fatalf("unexpected error: %s", microerror.Pretty(err, true))
	}

	clients := k8sclienttest.NewClients(k8sclienttest.ClientsConfig{
		CtrlClient: fake.NewClientBuilder().WithScheme(clientScheme).WithRuntimeObjects(object...).Build(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", microerror.Pretty(err, true))
	}

	appService, err := app.New(app.Config{
		Client: clients.CtrlClient(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", microerror.Pretty(err, true))
	}

	return appService
}
