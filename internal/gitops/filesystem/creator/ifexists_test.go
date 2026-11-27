package creator

import (
	"bytes"
	"errors"
	"testing"

	"github.com/spf13/afero"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/modifier"
)

type appendModifier struct{}

func (appendModifier) Execute(data []byte) ([]byte, error) {
	return append(data, []byte("changed\n")...), nil
}

// Test_Creator_IfExists checks that a modifier wrapped in IfExists changes a
// file that exists, and leaves out one that doesn't, without failing.
func Test_Creator_IfExists(t *testing.T) {
	for _, dryRun := range []bool{false, true} {
		fs := afero.NewMemMapFs()
		err := afero.WriteFile(fs, "/repo/present.yaml", []byte("kind: A\n"), 0600)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		var out bytes.Buffer
		err = NewCreator(CreatorConfig{
			DryRun: dryRun,
			Fs:     fs,
			Path:   repoPath,
			PostModifiers: map[string]modifier.Modifier{
				"present.yaml": modifier.IfExists{Modifier: appendModifier{}},
				"missing.yaml": modifier.IfExists{Modifier: appendModifier{}},
			},
			Stdout: &out,
		}).Create()
		if err != nil {
			t.Fatalf("dry run %t: unexpected error: %s", dryRun, err)
		}

		if ok, _ := afero.Exists(fs, "/repo/missing.yaml"); ok {
			t.Errorf("dry run %t: expected no missing.yaml", dryRun)
		}
		if bytes.Contains(out.Bytes(), []byte("missing.yaml")) {
			t.Errorf("dry run %t: expected no mention of missing.yaml, got:\n%s", dryRun, out.String())
		}

		data, _ := afero.ReadFile(fs, "/repo/present.yaml")
		changed := bytes.Contains(data, []byte("changed")) || bytes.Contains(out.Bytes(), []byte("changed"))
		if !changed {
			t.Errorf("dry run %t: expected present.yaml to be changed", dryRun)
		}
	}
}

// Test_Creator_MissingModifierTarget checks that a run whose modifier would
// change a file that doesn't exist fails before it writes anything.
func Test_Creator_MissingModifierTarget(t *testing.T) {
	fs := afero.NewMemMapFs()

	err := NewCreator(CreatorConfig{
		Fs:   fs,
		Path: repoPath,
		FsObjects: []*FsObject{
			NewFsObject("new.yaml", []byte("kind: B\n"), 0),
		},
		PostModifiers: map[string]modifier.Modifier{
			"missing.yaml": appendModifier{},
		},
		Stdout: &bytes.Buffer{},
	}).Create()
	if !errors.Is(err, ValidationError) {
		t.Fatalf("expected a validation error, got: %v", err)
	}
	if ok, _ := afero.Exists(fs, "/repo/new.yaml"); ok {
		t.Errorf("expected nothing written")
	}
}

type failingModifier struct{}

func (failingModifier) Execute([]byte) ([]byte, error) {
	return nil, errors.New("can't decrypt")
}

// Test_Creator_FailingModifierWritesNothing checks that a run whose post
// modifier fails, such as one that can't read an encrypted file, writes
// neither its new files nor the other modifiers' changes.
func Test_Creator_FailingModifierWritesNothing(t *testing.T) {
	fs := afero.NewMemMapFs()
	for path, data := range map[string]string{"/repo/a.yaml": "kind: A\n", "/repo/m.yaml": "kind: M\n", "/repo/z.yaml": "kind: Z\n"} {
		err := afero.WriteFile(fs, path, []byte(data), 0600)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
	}

	err := NewCreator(CreatorConfig{
		Fs:   fs,
		Path: repoPath,
		FsObjects: []*FsObject{
			NewFsObject("new.asc", []byte("public key"), 0),
		},
		PostModifiers: map[string]modifier.Modifier{
			"a.yaml": appendModifier{},
			"m.yaml": failingModifier{},
			"z.yaml": appendModifier{},
		},
		Stdout: &bytes.Buffer{},
	}).Create()
	if err == nil {
		t.Fatalf("expected the modifier's error")
	}

	if ok, _ := afero.Exists(fs, "/repo/new.asc"); ok {
		t.Errorf("expected no new file")
	}
	for _, path := range []string{"/repo/a.yaml", "/repo/z.yaml"} {
		data, _ := afero.ReadFile(fs, path)
		if bytes.Contains(data, []byte("changed")) {
			t.Errorf("expected %s unchanged, got:\n%s", path, data)
		}
	}
}
