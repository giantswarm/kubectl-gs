package release

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/blang/semver"

	"github.com/giantswarm/kubectl-gs/internal/key"

	"gopkg.in/yaml.v3"

	"github.com/giantswarm/microerror"
)

const (
	defaultBranch = "master"
)

const (
	releasesAWSIndexURLFmt   = "https://raw.githubusercontent.com/giantswarm/releases/%s/aws/kustomization.yaml"
	releasesAWSReleaseURLFmt = "https://raw.githubusercontent.com/giantswarm/releases/%s/aws/%s/release.yaml"

	releasesAzureIndexURLFmt   = "https://raw.githubusercontent.com/giantswarm/releases/%s/azure/kustomization.yaml"
	releasesAzureReleaseURLFmt = "https://raw.githubusercontent.com/giantswarm/releases/%s/azure/%s/release.yaml"

	firstAWSNodePoolsRelease   = "10.0.0"
	firstAzureNodePoolsRelease = "12.2.0"
)

type Config struct {
	Branch   string
	Provider string
}

type Release struct {
	releases []ReleaseObject
}

func New(config Config) (*Release, error) {
	if config.Branch == "" {
		config.Branch = defaultBranch
	}

	releases, err := readReleases(config.Branch, config.Provider)
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

func readReleases(branch, provider string) ([]ReleaseObject, error) {
	var err error

	var firstNPReleaseVersion semver.Version
	var releasesIndexURLFmt string
	var releaseURLFmt string
	{
		switch provider {
		case key.ProviderAWS:
			firstNPReleaseVersion = semver.MustParse(firstAWSNodePoolsRelease)
			releasesIndexURLFmt = releasesAWSIndexURLFmt
			releaseURLFmt = releasesAWSReleaseURLFmt
		case key.ProviderAzure:
			firstNPReleaseVersion = semver.MustParse(firstAzureNodePoolsRelease)
			releasesIndexURLFmt = releasesAzureIndexURLFmt
			releaseURLFmt = releasesAzureReleaseURLFmt
		}
	}

	var b []byte
	{
		resp, err := http.Get(fmt.Sprintf(releasesIndexURLFmt, branch))
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
			var releaseVersion semver.Version
			if releaseVersion, err = semver.ParseTolerant(v); firstNPReleaseVersion.Compare(releaseVersion) < 0 || err != nil {
				continue
			}

			resp, err := http.Get(fmt.Sprintf(releaseURLFmt, branch, v))
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
