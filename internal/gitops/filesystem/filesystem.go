package filesystem

import (
	"fmt"
	"strings"

	"github.com/spf13/afero"

	"github.com/giantswarm/microerror"
)

func NewCreator(config CreatorConfig) *Creator {
	return &Creator{
		dryRun:    config.DryRun,
		fs:        afero.NewOsFs(),
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
	var err error

	for _, o := range c.fsObjects {
		if !strings.HasSuffix(o.RelativePath, yamlExt) {
			err = c.fs.Mkdir(fmt.Sprintf("%s/%s", c.path, o.RelativePath), 0755)
			if err != afero.ErrFileExists {
				//noop
			} else if err != nil {
				return microerror.Mask(err)
			}
			continue
		}

		if len(o.Data) <= 1 {
			continue
		}

		err := afero.WriteFile(c.fs, fmt.Sprintf("%s/%s", c.path, o.RelativePath), o.Data, 0600)
		if err != nil {
			fmt.Println(err)
			return microerror.Mask(err)
		}
	}

	return nil
}
