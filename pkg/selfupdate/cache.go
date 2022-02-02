package selfupdate

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/giantswarm/microerror"
	"sigs.k8s.io/yaml"
)

const (
	cacheValidationDuration = 1 * time.Hour
	cacheFileName           = "selfupdate.yaml"
	cacheFileMode           = 0600
)

type cache struct {
	LastUpdate    time.Time `json:"last_update"`
	LatestVersion string    `json:"latest_version"`
}

func (c *cache) IsExpired() bool {
	expectedExpiration := c.LastUpdate.Add(cacheValidationDuration)

	return expectedExpiration.Before(time.Now().UTC())
}

func (c *cache) Persist(cacheDir string) error {
	c.LastUpdate = time.Now().UTC()

	serialized, err := yaml.Marshal(c)
	if err != nil {
		return microerror.Mask(err)
	}

	err = os.MkdirAll(cacheDir, 0700)
	if err != nil {
		return microerror.Mask(err)
	}

	out := filepath.Join(cacheDir, cacheFileName)

	err = ioutil.WriteFile(out, serialized, os.FileMode(cacheFileMode))
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (c *cache) Restore(fromPath string) error {
	in := filepath.Join(fromPath, cacheFileName)

	serialized, err := ioutil.ReadFile(in)
	if err != nil {
		return microerror.Mask(err)
	}

	err = yaml.Unmarshal(serialized, c)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
