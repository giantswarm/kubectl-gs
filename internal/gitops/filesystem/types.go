package filesystem

import (
	"io"

	"github.com/spf13/afero"
)

const (
	yamlExt = ".yaml"
)

type FsObject struct {
	RelativePath string
	Data         []byte
}

type CreatorConfig struct {
	DryRun    bool
	FsObjects []*FsObject
	Path      string
	Stdout    io.Writer
}

type Creator struct {
	dryRun    bool
	fs        afero.Fs
	fsObjects []*FsObject
	path      string
	stdout    io.Writer
}
