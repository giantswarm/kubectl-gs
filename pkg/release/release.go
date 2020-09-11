package release

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/giantswarm/microerror"
)

const (
	defaultBranch = "master"
)

const (
	releasesAWSIndexURLFmt   = "https://raw.githubusercontent.com/giantswarm/releases/%s/aws/kustomization.yaml"
	releasesAWSReleaseURLFmt = "https://raw.githubusercontent.com/giantswarm/releases/%s/aws/%s/release.yaml"

	firstAWSNodePoolsReleaseMajor = 10
)

type Config struct {
	Branch string
}

type Release struct {
	releases []ReleaseObject
}

func New(config Config) (*Release, error) {
	if config.Branch == "" {
		config.Branch = defaultBranch
	}

	releases, err := readReleases(config.Branch)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r := &Release{
		releases: releases,
	}

	return r, nil
}

func (r *Release) ReleaseComponents(version string) map[string]string {
	var releaseVersion string
	{
		if strings.HasPrefix(version, "v") {
			releaseVersion = version
		} else {
			releaseVersion = fmt.Sprintf("v%s", version)
		}
	}

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

func (r *Release) Validate(version string) bool {
	var releaseVersion string
	{
		if strings.HasPrefix(version, "v") {
			releaseVersion = version
		} else {
			releaseVersion = fmt.Sprintf("v%s", version)
		}

	}

	for _, release := range r.releases {
		if release.Metadata.Name == releaseVersion && release.Spec.State == "active" {
			return true
		}
	}

	return false
}

func readReleases(branch string) ([]ReleaseObject, error) {
	var b []byte
	{
		resp, err := http.Get(fmt.Sprintf(releasesAWSIndexURLFmt, branch))
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

	var releases []ReleaseObject
	{
		for _, v := range r.Resources {
			// skip legacy (< 10)
			versionParts := strings.Split(strings.TrimPrefix(v, "v"), ".")
			if i, _ := strconv.Atoi(versionParts[0]); i < firstAWSNodePoolsReleaseMajor {
				continue
			}

			resp, err := http.Get(fmt.Sprintf(releasesAWSReleaseURLFmt, branch, v))
			if err != nil {
				return nil, microerror.Mask(err)
			}
			defer resp.Body.Close()

			bodyBytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, microerror.Mask(err)
			}

			release := ReleaseObject{}
			err = yaml.Unmarshal(bodyBytes, &release)
			if err != nil {
				return nil, microerror.Mask(err)
			}

			releases = append(releases, release)
		}
	}

	return releases, nil
}
