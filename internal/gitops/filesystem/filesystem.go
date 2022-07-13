package filesystem

import (
	"fmt"
	"os"
	"strings"

	"github.com/imdario/mergo"
	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/microerror"
)

func NewCreator(config CreatorConfig) *Creator {
	return &Creator{
		dryRun:    config.DryRun,
		fs:        &afero.Afero{Fs: afero.NewOsFs()},
		fsObjects: config.FsObjects,
		path:      config.Path,
		stdout:    config.Stdout,
	}
}

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

	return nil
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
	rawData, err := c.fs.ReadFile(path)
	if os.IsNotExist(err) {
		//noop
	} else if err != nil {
		return microerror.Mask(err)
	}

	var src map[string]interface{}
	var dst map[string]interface{}

	err = yaml.Unmarshal(rawData, &src)
	if err != nil {
		return microerror.Mask(err)
	}

	err = yaml.Unmarshal(data, &dst)
	if err != nil {
		return microerror.Mask(err)
	}

	err = mergo.Merge(&dst, src, mergo.WithAppendSlice)
	if err != nil {
		return microerror.Mask(err)
	}

	bytes, err := yaml.Marshal(dst)
	if err != nil {
		return microerror.Mask(err)
	}

	err = c.fs.WriteFile(path, bytes, 0600)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
