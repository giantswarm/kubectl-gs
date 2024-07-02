package orgs

import (
	"bytes"
	"testing"
	"time"

	"github.com/giantswarm/microerror"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	v1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/giantswarm/kubectl-gs/v4/pkg/commonconfig"
	"github.com/giantswarm/kubectl-gs/v4/pkg/data/domain/organization"
	"github.com/giantswarm/kubectl-gs/v4/pkg/output"
	"github.com/giantswarm/kubectl-gs/v4/test/goldenfile"
	"github.com/giantswarm/kubectl-gs/v4/test/kubeclient"
	"github.com/giantswarm/kubectl-gs/v4/test/kubeconfig"
)

func Test_run(t *testing.T) {
	testCases := []struct {
		name               string
		storage            []runtime.Object
		args               []string
		isAdmin            bool
		permittedResources []v1.ResourceRule
		expectedGoldenFile string
		errorMatcher       func(error) bool
	}{
		{
			name:               "case 0: get orgs for admin",
			isAdmin:            true,
			args:               nil,
			expectedGoldenFile: "run_get_orgs_admin.golden",
			storage: []runtime.Object{
				newOrgResource("test-1", "org-test-1", time.Now()).Organization,
				newOrgResource("test-2", "org-test-2", time.Now()).Organization,
				newOrgResource("test-3", "org-test-3", time.Now()).Organization,
			},
		},
		{
			name:               "case 1: get orgs for non-admin",
			args:               nil,
			expectedGoldenFile: "run_get_orgs_non_admin.golden",
			storage: []runtime.Object{
				newOrgResource("test-1", "org-test-1", time.Now()).Organization,
				newOrgResource("test-2", "org-test-2", time.Now()).Organization,
				newOrgResource("test-3", "org-test-3", time.Now()).Organization,
			},
			permittedResources: []v1.ResourceRule{
				{
					Verbs:         []string{"get"},
					Resources:     []string{"organizations"},
					APIGroups:     []string{"security.giantswarm.io/v1alpha1"},
					ResourceNames: []string{"test-2"},
				},
			},
		},
		{
			name:               "case 2: get orgs for admin, with empty response",
			isAdmin:            true,
			args:               nil,
			expectedGoldenFile: "run_get_orgs_empty.golden",
		},
		{
			name:               "case 3: get orgs for non-admin, with no permitted resources",
			args:               nil,
			expectedGoldenFile: "run_get_orgs_empty.golden",
			storage: []runtime.Object{
				newOrgResource("test-1", "org-test-1", time.Now()).Organization,
				newOrgResource("test-2", "org-test-2", time.Now()).Organization,
				newOrgResource("test-3", "org-test-3", time.Now()).Organization,
			},
		},
		{
			name:               "case 4: get org by name",
			args:               []string{"test-1"},
			expectedGoldenFile: "run_get_org_by_name.golden",
			storage: []runtime.Object{
				newOrgResource("test-1", "org-test-1", time.Now()).Organization,
			},
		},
		{
			name:         "case 5: get org by name, not available in storage",
			args:         []string{"test-2"},
			errorMatcher: IsNotFound,
			storage: []runtime.Object{
				newOrgResource("test-1", "org-test-1", time.Now()).Organization,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fakeKubeConfig := kubeconfig.CreateFakeKubeConfigFromConfig(kubeconfig.CreateValidTestConfig())

			flg := &flag{
				print: genericclioptions.NewPrintFlags("").WithDefaultOutput(output.TypeDefault),
			}

			out := new(bytes.Buffer)
			runner := &runner{
				commonConfig: commonconfig.New(genericclioptions.NewTestConfigFlags().WithClientConfig(fakeKubeConfig)),
				flag:         flg,
				service:      newOrgService(t, tc.isAdmin, tc.permittedResources, tc.storage...),
				stdout:       out,
			}

			err := runner.Run(nil, tc.args)
			if tc.errorMatcher != nil {
				if !tc.errorMatcher(err) {
					t.Fatalf("error not matching expected matcher, got: %s", errors.Cause(err))
				}

				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			var expectedResult []byte
			{
				gf := goldenfile.New("testdata", tc.expectedGoldenFile)
				expectedResult, err = gf.Read()
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}
			}

			diff := cmp.Diff(string(expectedResult), out.String())
			if diff != "" {
				t.Fatalf("value not expected, got:\n %s", diff)
			}
		})
	}
}

func newOrgService(t *testing.T, isAdmin bool, permittedResources []v1.ResourceRule, object ...runtime.Object) *organization.Service {
	client := kubeclient.FakeK8sClient(object...)
	client.AddSubjectAccess(isAdmin)
	client.AddSubjectResourceRules("default", permittedResources)

	service, err := organization.New(organization.Config{
		Client: client,
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", microerror.Pretty(err, true))
	}

	return service.(*organization.Service)
}
