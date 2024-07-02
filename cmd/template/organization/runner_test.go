package organization

import (
	"bytes"
	goflag "flag"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"

	"github.com/giantswarm/kubectl-gs/v4/test/goldenfile"
)

var update = goflag.Bool("update", false, "update .golden reference test files")

// TestRunner_Run uses golden files.
//
// go test ./cmd/template/organization -run TestRunner_Run -update
func TestRunner_Run(t *testing.T) {
	testCases := []struct {
		name               string
		flag               *flag
		expectedGoldenFile string
		errorMatcher       func(error) bool
	}{
		{
			name:         "",
			flag:         &flag{},
			errorMatcher: IsInvalidFlag,
		},
		{
			name: "",
			flag: &flag{
				Name:   "example",
				Output: "",
			},
			expectedGoldenFile: "run_with_name.golden",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out := new(bytes.Buffer)

			r := &runner{
				flag:   tc.flag,
				stdout: out,
			}

			err := r.Run(nil, nil)
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
				if *update {
					expectedResult = out.Bytes()
					err = gf.Update(expectedResult)
					if err != nil {
						t.Fatalf("unexpected error: %s", err.Error())
					}
				} else {
					expectedResult, err = gf.Read()
					if err != nil {
						t.Fatalf("unexpected error: %s", err.Error())
					}
				}
			}

			diff := cmp.Diff(string(expectedResult), out.String())
			if diff != "" {
				t.Fatalf("value not expected, got:\n %s", diff)
			}
		})
	}
}
