package filesystem

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/afero"

	"github.com/giantswarm/microerror"
)

func (d *Dir) AddDirectory(dir *Dir) {
	d.dirs = append(d.dirs, dir)
}

func (d *Dir) AddFile(name string, content []byte) {
	d.files = append(d.files, &File{
		name: name,
		data: content,
	})
}

func NewCreator(config CreatorConfig) *Creator {
	return &Creator{
		directory: config.Directory,
		dryRun:    config.DryRun,
		fs:        afero.NewOsFs(),
		path:      config.Path,
		stdout:    config.Stdout,
	}
}

func NewDir(name string) *Dir {
	return &Dir{
		dirs:  make([]*Dir, 0),
		files: make([]*File, 0),
		name:  name,
	}
}

func (c *Creator) Create() error {
	if c.dryRun == true {
		c.directory.print(c.path, c.stdout)
		return nil
	}

	err := c.directory.write(c.path, c.fs)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (d *Dir) print(path string, stdout io.Writer) {
	path = fmt.Sprintf("%s/%s", path, d.name)

	fmt.Fprintln(stdout, path)
	for _, file := range d.files {
		fmt.Fprintf(stdout, "%s/%s\n", path, file.name)
		fmt.Println(string(file.data))
	}

	for _, dir := range d.dirs {
		dir.print(path, stdout)
	}
}

func (d *Dir) write(path string, fs afero.Fs) error {
	var err error

	path = fmt.Sprintf("%s/%s", path, d.name)

	err = fs.Mkdir(path, 0755)
	if err != nil {
		return microerror.Mask(err)
	}

	for _, file := range d.files {
		err = file.Write(path)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	for _, dir := range d.dirs {
		err = dir.write(path, fs)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func (f *File) Write(path string) error {
	err := os.WriteFile(fmt.Sprintf("%s/%s", path, f.name), f.data, 0600)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
