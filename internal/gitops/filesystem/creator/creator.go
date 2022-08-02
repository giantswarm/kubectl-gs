package creator

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/afero"

	"github.com/giantswarm/microerror"
)

// Create creates or prints the file system structure.
// On creating it also executes post modifiers, which is
// not done on printing.
//
// TBD: maybe run post modifiers on printing as well.
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

// NewCreator returns new creator object
func NewCreator(config CreatorConfig) *Creator {
	return &Creator{
		dryRun:        config.DryRun,
		fs:            &afero.Afero{Fs: afero.NewOsFs()},
		fsObjects:     config.FsObjects,
		path:          config.Path,
		postModifiers: config.PostModifiers,
		preValidators: config.PreValidators,
		stdout:        config.Stdout,
	}
}

// NewFsObject returns new file system object
func NewFsObject(path string, data []byte) *FsObject {
	return &FsObject{
		RelativePath: path,
		Data:         data,
	}
}

// createDirectory creates a new directory, if not already exists.
func (c *Creator) createDirectory(path string) error {
	err := c.fs.Mkdir(path, 0755)
	if os.IsExist(err) {
		//noop
	} else if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// createFile creates a new file.
func (c *Creator) createFile(path string, data []byte) error {
	err := c.fs.WriteFile(path, data, 0600)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// print prints the creator's file system objects.
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

// write writes the creator's file system objects into the disk.
func (c *Creator) write() error {
	for n, v := range c.preValidators {
		rawYaml, err := c.fs.ReadFile(fmt.Sprintf("%s/%s", c.path, n))
		if os.IsNotExist(err) {
			continue
		} else if err != nil {
			return microerror.Mask(err)
		}

		err = v.Execute(rawYaml)
		if err != nil {
			return microerror.Mask(err)
		}
	}

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
