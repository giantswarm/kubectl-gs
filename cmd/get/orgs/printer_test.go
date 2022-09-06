package orgs

import (
	"bytes"
	"testing"
	"time"

	"github.com/giantswarm/apiextensions/v6/pkg/apis/security/v1alpha1"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/giantswarm/kubectl-gs/pkg/data/domain/organization"
	"github.com/giantswarm/kubectl-gs/pkg/output"
	"github.com/giantswarm/kubectl-gs/test/goldenfile"
)

func Test_printOutput(t *testing.T) {
	testCases := []struct {
		name               string
		orgRes             organization.Resource
		outputType         string
		expectedGoldenFile string
	}{
		{
			name:               "case 0: print list of orgs, with table output",
			outputType:         output.TypeDefault,
			expectedGoldenFile: "print_list_of_orgs_table_output.golden",
			orgRes: newOrgCollection(
				*newOrgResource("test-1", "org-test-1", time.Now().Format(time.RFC3339)),
				*newOrgResource("test-2", "org-test-2", time.Now().Format(time.RFC3339)),
				*newOrgResource("test-3", "org-test-3", time.Now().Format(time.RFC3339)),
			),
		},
		{
			name:               "case 1: print list of orgs, with JSON output",
			outputType:         output.TypeJSON,
			expectedGoldenFile: "print_list_of_orgs_json_output.golden",
			orgRes: newOrgCollection(
				*newOrgResource("test-1", "org-test-1", "2022-08-18T08:07:48Z"),
				*newOrgResource("test-2", "org-test-2", "2022-08-18T08:07:48Z"),
				*newOrgResource("test-3", "org-test-3", "2022-08-18T08:07:48Z"),
			),
		},
		{
			name:               "case 2: print list of orgs, with YAML output",
			outputType:         output.TypeYAML,
			expectedGoldenFile: "print_list_of_orgs_yaml_output.golden",
			orgRes: newOrgCollection(
				*newOrgResource("test-1", "org-test-1", "2022-08-18T08:07:48Z"),
				*newOrgResource("test-2", "org-test-2", "2022-08-18T08:07:48Z"),
				*newOrgResource("test-3", "org-test-3", "2022-08-18T08:07:48Z"),
			),
		},
		{
			name:               "case 3: print list of orgs, with name output",
			outputType:         output.TypeName,
			expectedGoldenFile: "print_list_of_orgs_name_output.golden",
			orgRes: newOrgCollection(
				*newOrgResource("test-1", "org-test-1", "2022-08-18T08:07:48Z"),
				*newOrgResource("test-2", "org-test-2", "2022-08-18T08:07:48Z"),
				*newOrgResource("test-3", "org-test-3", "2022-08-18T08:07:48Z"),
			),
		},
		{
			name:               "case 4: print single org, with table output",
			outputType:         output.TypeDefault,
			expectedGoldenFile: "print_single_org_table_output.golden",
			orgRes:             newOrgResource("test-1", "org-test-1", time.Now().Format(time.RFC3339)),
		},
		{
			name:               "case 4: print single org, with JSON output",
			outputType:         output.TypeJSON,
			expectedGoldenFile: "print_single_org_json_output.golden",
			orgRes:             newOrgResource("test-1", "org-test-1", "2022-08-18T08:07:48Z"),
		},
		{
			name:               "case 4: print single org, with YAML output",
			outputType:         output.TypeYAML,
			expectedGoldenFile: "print_single_org_yaml_output.golden",
			orgRes:             newOrgResource("test-1", "org-test-1", "2022-08-18T08:07:48Z"),
		},
		{
			name:               "case 4: print single org, with name output",
			outputType:         output.TypeName,
			expectedGoldenFile: "print_single_org_name_output.golden",
			orgRes:             newOrgResource("test-1", "org-test-1", "2022-08-18T08:07:48Z"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			flag := &flag{
				print: genericclioptions.NewPrintFlags("").WithDefaultOutput(tc.outputType),
			}
			out := new(bytes.Buffer)
			runner := &runner{
				flag:   flag,
				stdout: out,
			}

			err := runner.printOutput(tc.orgRes)
			if err != nil {
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

func newOrgResource(name, namespace, created string) *organization.Organization {
	location, _ := time.LoadLocation("UTC")
	parsedCreationDate, _ := time.ParseInLocation(time.RFC3339, created, location)
	creationTimestamp := metav1.NewTime(parsedCreationDate)
	return &organization.Organization{
		Organization: &v1alpha1.Organization{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "security.giantswarm.io/v1alpha1",
				Kind:       "Organization",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:              name,
				CreationTimestamp: creationTimestamp,
			},
			Spec: v1alpha1.OrganizationSpec{},
			Status: v1alpha1.OrganizationStatus{
				Namespace: namespace,
			},
		},
	}
}

func newOrgCollection(orgs ...organization.Organization) *organization.Collection {
	return &organization.Collection{
		Items: orgs,
	}
}
