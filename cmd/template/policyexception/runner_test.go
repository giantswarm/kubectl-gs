package policyexception

import (
	"bytes"
	"context"
	goflag "flag"
	"testing"

	"github.com/giantswarm/micrologger"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	polexdraftv1alpha1 "github.com/giantswarm/exception-recommender/api/v1alpha1"

	"github.com/giantswarm/kubectl-gs/v2/test/goldenfile"
	"github.com/giantswarm/kubectl-gs/v2/test/kubeclient"
)

var update = goflag.Bool("update", false, "update .golden reference test files")

// Test_run uses golden files.
//
// go test ./cmd/template/policyexception -run Test_run -update
func Test_run(t *testing.T) {
	testCases := []struct {
		name               string
		flags              *flag
		args               []string
		clusterName        string
		expectedGoldenFile string
		errorMatcher       func(error) bool
		storage            []runtime.Object
	}{
		{
			name: "template polex",
			flags: &flag{
				Draft: "test-app",
			},
			args:               nil,
			expectedGoldenFile: "run_template_polex.golden",
			storage:            []runtime.Object{newPolexDraft("test-app", "policy-exceptions")},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			out := new(bytes.Buffer)
			// tc.flags.print = genericclioptions.NewPrintFlags("").WithDefaultOutput(output.TypeDefault)

			logger, err := micrologger.New(micrologger.Config{})
			if err != nil {
				t.Fatalf("failed to create logger: %s", err.Error())
			}

			runner := &runner{
				flag:   tc.flags,
				logger: logger,
				stdout: out,
			}

			k8sClient := kubeclient.FakeK8sClient(tc.storage...)
			err = runner.run(ctx, k8sClient.CtrlClient(), tc.args)
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
				t.Fatalf("no difference from golden file %s expected, got:\n %s", tc.expectedGoldenFile, diff)
			}
		})
	}
}

func newPolexDraft(name string, namespace string) *polexdraftv1alpha1.PolicyExceptionDraft {
	p := &polexdraftv1alpha1.PolicyExceptionDraft{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "policy.giantswarm.io",
			Kind:       "PolicyExceptionDraft",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: polexdraftv1alpha1.PolicyExceptionDraftSpec{
			Targets: []polexdraftv1alpha1.Target{
				{
					Kind:       "Deployment",
					Names:      []string{"test-app*"},
					Namespaces: []string{"test-app"},
				},
			},
			Policies: []string{"restrict-policy"},
		},
	}
	return p
}
