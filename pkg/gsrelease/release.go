package gsrelease

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"path"

	"gopkg.in/yaml.v3"

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

type Release struct {
	Kind     string          `json:"kind"`
	Metadata ReleaseMetadata `json:"metadata"`
	Spec     ReleaseSpec     `json:"spec"`
	Version  string          `json:"apiVersion"`
}

type ReleaseMetadata struct {
	Name        string            `json:"name"`
	Annotations map[string]string `json:"annotations"`
}

type ReleaseSpec struct {
	Date       string       `json:"date"`
	Apps       []Apps       `json:"apps"`
	Components []Components `json:"components"`
	State      string       `json:"state"`
	Version    string       `json:"version"`
}

type Components struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Apps struct {
	ComponentVersion string `json:"componentVersion"`
	Name             string `json:"name"`
	Version          string `json:"version"`
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

func (r *GSRelease) ReleaseComponents(releaseVersion string) map[string]string {
	releaseComponents := make(map[string]string)

	for _, release := range r.releases {
		if release.Metadata.Name == releaseVersion {
			for _, component := range release.Spec.Components {
				releaseComponents[component.Name] = component.Version
			}
		}
	}

	return releaseComponents
}

func (r *GSRelease) Validate(version string) bool {
	releaseVersion := fmt.Sprintf("v%s", version)
	for _, release := range r.releases {
		if release.Metadata.Name == releaseVersion {
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
	configPath := path.Join(usr.HomeDir, configRelativePath)

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

	dec := yaml.NewDecoder(bytes.NewReader(data))

	var releases []Release
	{
		for {
			var release Release
			if dec.Decode(&release) != nil {
				break
			}
			releases = append(releases, release)
		}
	}

	return releases, nil
}
