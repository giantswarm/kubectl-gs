package gsrelease

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/giantswarm/microerror"
)

const (
	defaultBranch = "master"
)

const (
	releasesAWSBaseURLFmt   = "https://raw.githubusercontent.com/giantswarm/releases/%s/aws/"
	releasesAWSConfigURLFmt = "https://raw.githubusercontent.com/giantswarm/releases/%s/aws/kustomization.yaml"
)

type Config struct {
	Branch string
}

type GSRelease struct {
	releases []Release
}

func New(config Config) (*GSRelease, error) {
	if config.Branch == "" {
		config.Branch = defaultBranch
	}

	releases, err := readReleases(config.Branch)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	g := &GSRelease{
		releases: releases,
	}

	return g, nil
}

func (g *GSRelease) ReleaseComponents(version string) map[string]string {
	var releaseVersion string
	{
		if strings.HasPrefix(version, "v") {
			releaseVersion = version
		} else {
			releaseVersion = fmt.Sprintf("v%s", version)
		}
	}

	releaseComponents := make(map[string]string)

	for _, release := range g.releases {
		if release.Metadata.Name == releaseVersion {
			for _, component := range release.Spec.Components {
				releaseComponents[component.Name] = component.Version
			}
		}
	}

	return releaseComponents
}

func (g *GSRelease) Validate(version string) bool {
	var releaseVersion string
	{
		if strings.HasPrefix(version, "v") {
			releaseVersion = version
		} else {
			releaseVersion = fmt.Sprintf("v%s", version)
		}

	}

	for _, release := range g.releases {
		if release.Metadata.Name == releaseVersion {
			return true
		}
	}

	return false
}

func readReleases(branch string) ([]Release, error) {
	var b []byte
	{
		resp, err := http.Get(fmt.Sprintf(releasesAWSConfigURLFmt, branch))
		if err != nil {
			return nil, microerror.Mask(err)
		}
		defer resp.Body.Close()

		b, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	r := struct {
		Resources []string `yaml:"resources"`
	}{}
	{
		err := yaml.Unmarshal(b, &r)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var releases []Release
	{
		for _, v := range r.Resources {
			resp, err := http.Get(fmt.Sprintf(releasesAWSBaseURLFmt, branch) + v + "/release.yaml")
			if err != nil {
				return nil, microerror.Mask(err)
			}
			defer resp.Body.Close()

			bodyBytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, microerror.Mask(err)
			}

			release := Release{}
			err = yaml.Unmarshal(bodyBytes, &release)
			if err != nil {
				return nil, microerror.Mask(err)
			}

			releases = append(releases, release)
		}
	}

	return releases, nil
}
