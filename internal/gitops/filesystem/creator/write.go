package creator

import (
	"fmt"
	"os"

	"github.com/giantswarm/microerror"
)

// edit is a file as a post modifier leaves it.
type edit struct {
	path string
	data []byte
	perm os.FileMode
}

// edits runs the post modifiers on the files as they are, or as this run
// creates them, without writing anything.
func (c *Creator) edits() ([]edit, error) {
	var edits []edit
	for n, m := range c.modifiers() {
		path := fmt.Sprintf("%s/%s", c.path, n)

		data, perm, err := c.current(n)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		edited, err := m.Execute(data)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		edits = append(edits, edit{path: path, data: edited, perm: perm})
	}

	return edits, nil
}

// current returns the file at n as it is, or as this run creates it.
func (c *Creator) current(n string) ([]byte, os.FileMode, error) {
	path := fmt.Sprintf("%s/%s", c.path, n)

	exists, err := c.fs.Exists(path)
	if err != nil {
		return nil, 0, microerror.Mask(err)
	}
	if !exists {
		for _, o := range c.fsObjects {
			if o.RelativePath == n {
				return o.Data, o.Permission, nil
			}
		}
	}

	stat, err := c.fs.Stat(path)
	if err != nil {
		return nil, 0, microerror.Mask(err)
	}
	data, err := c.fs.ReadFile(path)
	if err != nil {
		return nil, 0, microerror.Mask(err)
	}

	return data, stat.Mode(), nil
}

// writeObjects creates the run's directories, and its files that don't
// exist yet.
func (c *Creator) writeObjects() error {
	for _, o := range c.fsObjects {
		path := fmt.Sprintf("%s/%s", c.path, o.RelativePath)

		if o.isDir() {
			err := c.createDirectory(path, o.Permission)
			if err != nil {
				return microerror.Mask(err)
			}
			continue
		}

		if len(o.Data) <= 1 {
			continue
		}

		err := c.createFileWithoutOverride(path, o.Data, o.Permission)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
