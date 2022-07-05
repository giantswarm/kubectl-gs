package filesystem

import (
	"io"

	"github.com/spf13/afero"
)

type Dir struct {
	dirs  []*Dir
	files []*File
	name  string
}

type File struct {
	data []byte
	name string
}

type CreatorConfig struct {
	Directory *Dir
	Path      string
	DryRun    bool
	Stdout    io.Writer
}

type Creator struct {
	directory *Dir
	dryRun    bool
	fs        afero.Fs
	path      string
	stdout    io.Writer
}
