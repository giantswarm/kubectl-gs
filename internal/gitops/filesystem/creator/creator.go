package creator

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/afero"

	"github.com/giantswarm/microerror"
)

func (c *Creator) Create() error {
	if c.dryRun {
		c.print()
		return nil
	}

	err := c.write()
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func NewCreator(config CreatorConfig) *Creator {
	return &Creator{
		dryRun:        config.DryRun,
		fs:            &afero.Afero{Fs: afero.NewOsFs()},
		fsObjects:     config.FsObjects,
		path:          config.Path,
		postModifiers: config.PostModifiers,
		stdout:        config.Stdout,
	}
}

func NewFsObject(path string, data []byte) *FsObject {
	return &FsObject{
		RelativePath: path,
		Data:         data,
	}
}

func (c *Creator) createDirectory(path string) error {
	err := c.fs.Mkdir(path, 0755)
	if os.IsExist(err) {
		//noop
	} else if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (c *Creator) createFile(path string, data []byte) error {
	err := c.fs.WriteFile(path, data, 0600)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (c *Creator) print() {
	for _, o := range c.fsObjects {
		if !strings.HasSuffix(o.RelativePath, yamlExt) {
			fmt.Fprintf(c.stdout, "%s/%s\n", c.path, o.RelativePath)
			continue
		}

		if len(o.Data) <= 1 {
			continue
		}

		fmt.Fprintf(c.stdout, "%s/%s\n", c.path, o.RelativePath)
		fmt.Fprintln(c.stdout, string(o.Data))
	}
}

func (c *Creator) write() error {
	for _, o := range c.fsObjects {
		if !strings.HasSuffix(o.RelativePath, yamlExt) {
			err := c.createDirectory(fmt.Sprintf("%s/%s", c.path, o.RelativePath))
			if err != nil {
				return microerror.Mask(err)
			}
			continue
		}

		if len(o.Data) <= 1 {
			continue
		}

		err := c.createFile(fmt.Sprintf("%s/%s", c.path, o.RelativePath), o.Data)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	for n, m := range c.postModifiers {
		rawYaml, err := c.fs.ReadFile(fmt.Sprintf("%s/%s", c.path, n))
		if err != nil {
			return microerror.Mask(err)
		}

		edited, err := m.Execute(rawYaml)
		if err != nil {
			return microerror.Mask(err)
		}

		err = c.createFile(fmt.Sprintf("%s/%s", c.path, n), edited)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
