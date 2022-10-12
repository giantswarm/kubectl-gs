package goldenfile

import (
	"os"
	"path/filepath"

	"github.com/giantswarm/microerror"
)

// GoldenFile helps perform snapshot testing, to avoid long
// test payloads in test files.
type GoldenFile struct {
	path string
}

func New(basePath string, filePath string) *GoldenFile {
	path := filepath.Join(basePath, filePath)

	gf := &GoldenFile{
		path: path,
	}

	return gf
}

func (gf *GoldenFile) Read() ([]byte, error) {
	file, err := gf.readFile(gf.path)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return file, nil
}

func (gf *GoldenFile) Update(data []byte) error {
	err := os.WriteFile(gf.path, data, 0600)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (gf *GoldenFile) readFile(path string) ([]byte, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return file, nil
}
