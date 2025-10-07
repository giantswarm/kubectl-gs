package nodepool

import (
	"bytes"
	"testing"

	"github.com/giantswarm/micrologger"
	"github.com/pkg/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/giantswarm/kubectl-gs/v5/pkg/output"
)

func Test_run(t *testing.T) {
	testCases := []struct {
		name         string
		flags        *flag
		errorMatcher func(error) bool
	}{
		{
			name: "Unsupported provider",
			flags: &flag{
				ClusterName:  "test1",
				Provider:     "unsupported",
				Organization: "test",
			},
			errorMatcher: IsInvalidFlag,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out := new(bytes.Buffer)
			tc.flags.print = genericclioptions.NewPrintFlags("").WithDefaultOutput(output.TypeDefault)

			logger, err := micrologger.New(micrologger.Config{})
			if err != nil {
				t.Fatalf("failed to create logger: %s", err.Error())
			}

			runner := &runner{
				flag:   tc.flags,
				logger: logger,
				stdout: out,
				stderr: out,
			}

			err = runner.Run(nil, nil)
			if tc.errorMatcher != nil {
				if !tc.errorMatcher(err) {
					t.Fatalf("error not matching expected matcher, got: %s", errors.Cause(err))
				}
				t.Log(err.Error())

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}
		})
	}
}
