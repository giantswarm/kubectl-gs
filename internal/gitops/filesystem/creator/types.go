package creator

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
	DryRun        bool
	FsObjects     []*FsObject
	Path          string
	PostModifiers map[string]Modifier
	Stdout        io.Writer
}

type Creator struct {
	dryRun        bool
	fs            *afero.Afero
	fsObjects     []*FsObject
	path          string
	postModifiers map[string]Modifier
	stdout        io.Writer
}
