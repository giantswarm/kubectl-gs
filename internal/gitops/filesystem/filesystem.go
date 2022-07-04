package filesystem

import (
	"fmt"
	"os"

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
		path:   config.Path,
		dryRun: config.DryRun,
	}
}

func NewDir(name string) *Dir {
	return &Dir{
		dirs:  make([]*Dir, 0),
		files: make([]*File, 0),
		name:  name,
	}
}

func (c *Creator) Write(d *Dir) error {
	if c.dryRun == true {
		d.print(c.path)
		return nil
	}

	err := d.write(c.path)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (d *Dir) print(path string) {
	path = fmt.Sprintf("%s/%s", path, d.name)

	fmt.Println(path)
	for _, file := range d.files {
		fmt.Printf("%s/%s\n", path, file.name)
		fmt.Println(string(file.data))
	}

	for _, dir := range d.dirs {
		dir.print(path)
	}
}

func (d *Dir) write(path string) error {
	var err error

	path = fmt.Sprintf("%s/%s", path, d.name)

	err = os.Mkdir(path, 0755)
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
		err = dir.write(path)
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
