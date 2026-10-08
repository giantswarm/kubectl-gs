package migration

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
)

// FileOp says what an upgrade does to a file.
type FileOp string

const (
	FileAdded    FileOp = "added"
	FileModified FileOp = "modified"
	FileRemoved  FileOp = "removed"
)

// FileChange is one file an upgrade adds, modifies or removes, relative to the
// repository root.
type FileChange struct {
	Op   FileOp
	Path string
}

// stage returns an in-memory copy of the repository cloned at repoPath, for a
// dry run to run the migrations against. It is only ever read back to report
// what a real run would change, never written to disk.
//
// Only directories and regular files are copied: `.git` is skipped, and so are
// symlinks, sockets and the like, which kubectl-gs never generates. So a dry
// run can differ from a real run for a migration reading or writing through a
// symlink. The copy does not enforce file permissions either, so a dry run can
// succeed where a real run fails on a file the user cannot write, and every
// file has to be readable for it to start.
func stage(fs *afero.Afero, repoPath string) (*afero.Afero, error) {
	staging := &afero.Afero{Fs: afero.NewMemMapFs()}

	err := walkManaged(fs, repoPath, func(path string, info os.FileInfo) error {
		if info.IsDir() {
			return staging.MkdirAll(path, info.Mode().Perm())
		}

		data, err := fs.ReadFile(path)
		if err != nil {
			return microerror.Maskf(stagingError, "copying the repository into memory for the dry run: %s", err)
		}

		return staging.WriteFile(path, data, info.Mode().Perm())
	})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return staging, nil
}

type file struct {
	data []byte
	perm os.FileMode
}

// snapshot reads the regular files walkManaged sees under root, keyed by their
// path relative to root.
func snapshot(fs *afero.Afero, root string) (map[string]file, error) {
	files := map[string]file{}

	err := walkManaged(fs, root, func(path string, info os.FileInfo) error {
		if info.IsDir() {
			return nil
		}

		data, err := fs.ReadFile(path)
		if err != nil {
			return microerror.Maskf(stagingError, "%s", err)
		}

		files[rel(root, path)] = file{data: data, perm: info.Mode().Perm()}
		return nil
	})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return files, nil
}

// changes lists the files that differ between two snapshots, sorted by path.
// A file whose permissions alone changed counts as modified.
func changes(before, after map[string]file) []FileChange {
	out := []FileChange{}

	for p, a := range after {
		b, ok := before[p]
		switch {
		case !ok:
			out = append(out, FileChange{Op: FileAdded, Path: p})
		case !bytes.Equal(a.data, b.data) || a.perm != b.perm:
			out = append(out, FileChange{Op: FileModified, Path: p})
		}
	}
	for p := range before {
		if _, ok := after[p]; !ok {
			out = append(out, FileChange{Op: FileRemoved, Path: p})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Path < out[j].Path
	})

	return out
}

// walkManaged walks the directories and regular files under root, skipping
// `.git`, whether a directory or the file a submodule or worktree has, and
// anything else, such as symlinks, that is not a directory or regular file.
// The root itself is not passed to fn.
func walkManaged(fs *afero.Afero, root string, fn func(path string, info os.FileInfo) error) error {
	return fs.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return microerror.Maskf(stagingError, "reading the repository for the dry run: %s", err)
		}
		if path == root {
			return nil
		}

		if info.Name() == ".git" {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !info.IsDir() && !info.Mode().IsRegular() {
			return nil
		}

		return fn(path, info)
	})
}

func rel(root, path string) string {
	r, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}

	return filepath.ToSlash(r)
}
