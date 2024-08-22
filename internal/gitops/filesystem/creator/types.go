package creator

import (
	"io"
	"os"

	"github.com/spf13/afero"

	"github.com/giantswarm/kubectl-gs/v5/internal/gitops/filesystem/modifier"
)

const (
	defaultDirPerm  = 0755
	defaultFilePerm = 0600
)

type Creator struct {
	dryRun        bool
	fs            *afero.Afero
	fsObjects     []*FsObject
	path          string
	postModifiers map[string]modifier.Modifier
	preValidators map[string]func(*afero.Afero, string) error
	stdout        io.Writer
}

type CreatorConfig struct {
	DryRun        bool
	FsObjects     []*FsObject
	Path          string
	PostModifiers map[string]modifier.Modifier
	PreValidators map[string]func(*afero.Afero, string) error
	Stdout        io.Writer
}

type FsObject struct {
	Data         []byte
	Permission   os.FileMode
	RelativePath string
}
