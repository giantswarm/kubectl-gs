package gsrelease

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"path"

	"github.com/ghodss/yaml"
	"github.com/giantswarm/microerror"
)

const (
	configRelativePath         = ".kube/gs"
	releasesConfigRelativePath = ".kube/gs/aws.yaml"
	releasesConfigURL          = "https://raw.githubusercontent.com/giantswarm/releases/master/aws.yaml"
)

type Config struct {
	NoCache bool
}

type GSRelease struct {
	releases []Release
}

type Authority struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Release struct {
	Authorities []Authority `json:"authorities"`
	State       string      `json:"state"`
	Version     string      `json:"version"`
}

func New(c Config) (*GSRelease, error) {

	err := ensureConfigDirExists()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	err = ensureReleasesConfigExists(c.NoCache)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	releases, err := readReleases()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	newReleases := &GSRelease{
		releases: releases,
	}

	return newReleases, nil
}

func (r *GSRelease) Validate(version string) bool {
	for _, release := range r.releases {
		if release.Version == version {
			return true
		}
	}

	return false
}

func ensureConfigDirExists() error {
	usr, err := user.Current()
	if err != nil {
		return microerror.Mask(err)
	}
	configPath := path.Join(usr.HomeDir, releasesConfigRelativePath)

	_, err = os.Stat(configPath)
	if os.IsNotExist(err) {
		err = os.Mkdir(configPath, 0700)
		if err != nil {
			return microerror.Mask(err)
		}
	} else if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func ensureReleasesConfigExists(noCache bool) error {
	usr, err := user.Current()
	if err != nil {
		return microerror.Mask(err)
	}
	releasesConfigPath := path.Join(usr.HomeDir, releasesConfigRelativePath)

	_, err = os.Stat(releasesConfigPath)
	if os.IsNotExist(err) || noCache {
		resp, err := http.Get(releasesConfigURL)
		if err != nil {
			return microerror.Mask(err)
		}
		defer resp.Body.Close()

		out, err := os.Create(releasesConfigPath)
		if err != nil {
			return microerror.Mask(err)
		}
		defer out.Close()

		_, err = io.Copy(out, resp.Body)
		if err != nil {
			return microerror.Mask(err)
		}
	} else if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func readReleases() ([]Release, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	releasesConfigPath := path.Join(usr.HomeDir, releasesConfigRelativePath)

	data, err := ioutil.ReadFile(releasesConfigPath)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var releases []Release
	{
		err = yaml.Unmarshal(data, &releases)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return releases, nil
}
