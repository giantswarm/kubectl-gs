package errorprinter

import (
	"errors"
	"testing"

	"github.com/giantswarm/microerror"
	"github.com/google/go-cmp/cmp"

	"github.com/giantswarm/kubectl-gs/test/goldenfile"
)

func TestFormat(t *testing.T) {
	testCases := []struct {
		name               string
		creator            func() error
		expectedGoldenFile string
	}{
		{
			name: "case 0: generic error",
			creator: func() error {
				err := errors.New("something went wrong")

				return err
			},
			expectedGoldenFile: "print_generic_error.golden",
		},
		{
			name: "case 1: generic empty error",
			creator: func() error {
				err := errors.New("")

				return err
			},
			expectedGoldenFile: "print_generic_empty_error.golden",
		},
		{
			name: "case 2: generic error with 'error:' prefix in message",
			creator: func() error {
				err := errors.New("error: something went wrong")

				return err
			},
			expectedGoldenFile: "print_generic_error_with_prefix.golden",
		},
		{
			name: "case 3: simple microerror",
			creator: func() error {
				err := errors.New("something went wrong")

				return microerror.Mask(err)
			},
			expectedGoldenFile: "print_simple_microerror.golden",
		},
		{
			name: "case 4: custom microerror",
			creator: func() error {
				err := &microerror.Error{
					Kind: "somethingWentWrongError",
				}

				return microerror.Mask(err)
			},
			expectedGoldenFile: "print_custom_microerror.golden",
		},
		{
			name: "case 5: custom microerror, with additional message",
			creator: func() error {
				err := &microerror.Error{
					Kind: "somethingWentWrongError",
				}

				return microerror.Maskf(err, "something bad happened, and we had to crash")
			},
			expectedGoldenFile: "print_custom_microerror_with_additional_message.golden",
		},
		{
			name: "case 6: custom microerror, with additional multiline message",
			creator: func() error {
				err := &microerror.Error{
					Kind: "somethingWentWrongError",
				}

				return microerror.Maskf(err, "something bad happened, and we had to crash\nthat's the first time it happened, really")
			},
			expectedGoldenFile: "print_custom_microerror_with_additional_multiline_message.golden",
		},
		{
			name: "case 7: custom microerror, with stack trace",
			creator: func() error {
				err := &microerror.Error{
					Kind: "somethingWentWrongError",
				}

				// Let's build up this stack trace.
				newErr := microerror.Mask(err)
				for i := 0; i < 10; i++ {
					newErr = microerror.Mask(newErr)
				}

				return microerror.Mask(newErr)
			},
			expectedGoldenFile: "print_custom_microerror_with_stack_trace.golden",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ep := New()
			newErr := tc.creator()
			result := ep.Format(newErr)

			gf := goldenfile.New("testdata", tc.expectedGoldenFile)
			expectedResult, err := gf.Read()
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			diff := cmp.Diff(string(expectedResult), result)
			if diff != "" {
				t.Fatalf("value not expected, got:\n %s", diff)
			}
		})
	}
}
