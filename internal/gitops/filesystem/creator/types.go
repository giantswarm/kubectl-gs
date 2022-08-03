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
	PreValidators map[string]Validator
	Stdout        io.Writer
}

type Creator struct {
	dryRun        bool
	fs            *afero.Afero
	fsObjects     []*FsObject
	path          string
	postModifiers map[string]Modifier
	preValidators map[string]Validator
	stdout        io.Writer
}
