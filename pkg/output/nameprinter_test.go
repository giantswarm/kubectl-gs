package output

import (
	"bytes"
	"flag"
	"testing"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/giantswarm/kubectl-gs/v2/test/goldenfile"
)

var update = flag.Bool("update", false, "update .golden reference test files")

// Test_run uses golden files.
//
// go test ./pkg/output -run TestPrintResourceNames -update
func TestPrintResourceNames(t *testing.T) {
	testCases := []struct {
		name               string
		resource           runtime.Object
		expectedGoldenFile string
	}{
		{
			name:               "case 0: regular resource",
			resource:           newResource("asbv2"),
			expectedGoldenFile: "regular_resource.golden",
		},
		{
			name: "case 1: list resource",
			resource: newListResource(
				newResource("asbv2"),
				newResource("ffs1s"),
				newResource("d01s1"),
			),
			expectedGoldenFile: "list_resource.golden",
		},
		{
			name: "case 2: nested list resource",
			resource: newListResource(
				newListResource(
					newListResource(
						newResource("s0a1l"),
						newResource("asd40"),
						newResource("afd12"),
					),
					newListResource(
						newResource("asbv2"),
						newResource("ffs1s"),
						newResource("d01s1"),
					),
					newListResource(
						newResource("asbv2"),
						newResource("ffs1s"),
						newResource("d01s1"),
					),
				),
				newListResource(
					newListResource(
						newResource("1290fd"),
						newResource("3450sd"),
						newResource("33fdas"),
					),
					newListResource(
						newResource("66fdf"),
						newResource("55kll"),
						newResource("33ssd"),
					),
					newListResource(
						newResource("01sad"),
						newResource("asd01"),
						newResource("34dsa"),
					),
				),
			),
			expectedGoldenFile: "nested_list_resource.golden",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out := new(bytes.Buffer)
			err := PrintResourceNames(out, tc.resource)
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			gf := goldenfile.New("testdata", tc.expectedGoldenFile)
			expectedResult, err := gf.Read()
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			if *update {
				err = gf.Update(out.Bytes())
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

type TestResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

func (tr *TestResource) DeepCopyObject() runtime.Object {
	return tr
}

type TestResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []runtime.Object `json:"items"`
}

func (tr *TestResourceList) DeepCopyObject() runtime.Object {
	return tr
}

func newListResource(resources ...runtime.Object) *TestResourceList {
	resourceList := &TestResourceList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "List",
			APIVersion: "v1",
		},
		Items: resources,
	}

	return resourceList
}

func newResource(name string) *TestResource {
	r := &TestResource{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	r.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "testing.domain.coolio.com",
		Version: "v1alpha3",
		Kind:    "TestResource",
	})

	return r
}
