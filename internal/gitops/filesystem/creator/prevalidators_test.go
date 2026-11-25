package creator

import (
	"errors"
	"io"
	"testing"

	"github.com/spf13/afero"
)

// Test_Create_PreValidatorOrder checks that pre-validators run in path order,
// so the check on a directory runs before the checks on what it holds, and
// its error is the one reported.
func Test_Create_PreValidatorOrder(t *testing.T) {
	errDir := errors.New("directory")
	errChild := errors.New("child")

	validators := map[string]func(*afero.Afero, string) error{
		"a/z": func(*afero.Afero, string) error { return errChild },
		"a":   func(*afero.Afero, string) error { return errDir },
		"a/b": func(*afero.Afero, string) error { return errChild },
	}

	// Map iteration order is random, so run often enough to catch it.
	for range 50 {
		c := NewCreator(CreatorConfig{
			Fs:            afero.NewMemMapFs(),
			Path:          "/repo",
			PreValidators: validators,
			Stdout:        io.Discard,
		})

		err := c.Create()
		if !errors.Is(err, errDir) {
			t.Fatalf("expected the directory's error, got: %v", err)
		}
	}
}
