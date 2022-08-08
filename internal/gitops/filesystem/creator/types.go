package creator

import (
	"io"

	"github.com/spf13/afero"

	"github.com/giantswarm/kubectl-gs/internal/gitops/filesystem/modifier"
)

var (
	filesExt = []string{
		".yaml",
		".asc",
	}
)

type FsObject struct {
	RelativePath string
	Data         []byte
}

type CreatorConfig struct {
	DryRun        bool
	FsObjects     []*FsObject
	Path          string
	PostModifiers map[string]modifier.Modifier
	PreValidators map[string]Validator
	Stdout        io.Writer
}

type Creator struct {
	dryRun        bool
	fs            *afero.Afero
	fsObjects     []*FsObject
	path          string
	postModifiers map[string]modifier.Modifier
	preValidators map[string]Validator
	stdout        io.Writer
}
