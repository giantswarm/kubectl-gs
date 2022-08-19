package creator

import (
	"bytes"
	"fmt"
	"os"

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
func NewFsObject(path string, data []byte, perm os.FileMode) *FsObject {
	fo := &FsObject{
		RelativePath: path,
		Data:         data,
		Permission:   perm,
	}

	fo.ensurePermissions()

	return fo
}

// createDirectory creates a new directory, if not already exists.
func (c *Creator) createDirectory(path string, perm os.FileMode) error {
	err := c.fs.Mkdir(path, perm)
	if os.IsExist(err) {
		//noop
	} else if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// createFile creates a new file.
func (c *Creator) createFile(path string, data []byte, perm os.FileMode, force bool) error {
	var err error

	// user may try to re-run some command against already existing
	// layer of the structure, but the file there may contain changes
	// made by other commands, or made by other users, this could result
	// in losing them, so it's better to skip if file exist already.
	exist, err := c.fs.Exists(path)
	if err != nil {
		return microerror.Mask(err)
	}

	if exist && !force {
		return nil
	}

	err = c.fs.WriteFile(path, data, perm)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (c *Creator) createFileWithOverride(path string, data []byte, perm os.FileMode) error {
	return c.createFile(path, data, perm, true)
}

func (c *Creator) createFileWithoutOverride(path string, data []byte, perm os.FileMode) error {
	return c.createFile(path, data, perm, false)
}

// ensurePermissions sets default permissions if the one provided
// are empty.
func (fo *FsObject) ensurePermissions() {
	if fo.Permission != 0 {
		return
	}

	if len(fo.Data) <= 1 {
		fo.Permission = defaultDirPerm
	} else {
		fo.Permission = defaultFilePerm
	}
}

// isDir checks path against pre-configured suffixes
func (fo *FsObject) isDir() bool {
	return len(fo.Data) <= 1
}

// print prints the creator's file system objects.
func (c *Creator) print() {
	for _, o := range c.fsObjects {

		// Print path to the directory to be created
		if o.isDir() {
			fmt.Fprintf(c.stdout, "%s/%s\n", c.path, o.RelativePath)
			continue
		}

		data := bytes.TrimSpace(o.Data)
		if len(data) == 0 {
			continue
		}

		// Print path to the file, and then the file content
		fmt.Fprintf(c.stdout, "%s/%s\n", c.path, o.RelativePath)
		fmt.Fprintf(c.stdout, "%s\n\n", string(data))
	}

	for n, m := range c.postModifiers {
		rawYaml, err := c.fs.ReadFile(fmt.Sprintf("%s/%s", c.path, n))
		if err != nil {
			// Very simple way to inform user what is going to happen when
			// command is executed without the `dry-run` flag. May not be
			// the best way in the future, but it is something to start with,
			// and serves the purpose of giving user a hint.
			fmt.Fprintln(c.stdout, err)
			continue
		}

		edited, err := m.Execute(rawYaml)
		if err != nil {
			return
		}

		fmt.Fprintf(c.stdout, "%s/%s\n", c.path, n)
		fmt.Fprintln(c.stdout, string(edited))
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

	for n, m := range c.postModifiers {
		file := fmt.Sprintf("%s/%s", c.path, n)

		rawYaml, err := c.fs.ReadFile(file)
		if err != nil {
			return microerror.Mask(err)
		}

		stat, err := c.fs.Stat(file)
		if err != nil {
			return microerror.Mask(err)
		}

		edited, err := m.Execute(rawYaml)
		if err != nil {
			return microerror.Mask(err)
		}

		err = c.createFileWithOverride(file, edited, stat.Mode())
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
