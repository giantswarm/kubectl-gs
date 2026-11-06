package wcluster

import (
	"encoding/json"
	"path"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/appcr"
)

// clusterChartName returns the release chart the base deploys, for example
// `release-aws`, read from the URL of its OCIRepository. It follows the base's
// resources, so an overlay returns the chart of the base below. Other
// OCIRepositories the base may ship are ignored. It returns an empty string
// when the base has no release chart.
//
// A base deploying an App CR is refused, as kubectl-gs only generates
// HelmReleases. That includes an overlay whose base below does, like
// gitops-template's bases/clusters/capa/release, which patches the cluster App
// to a release chart.
func clusterChartName(fs afero.Fs, baseDir string) (string, error) {
	files, err := appcr.Resources(fs, baseDir)
	if err != nil {
		return "", microerror.Mask(err)
	}

	var charts []string
	for _, filePath := range files {
		data, err := afero.ReadFile(fs, filePath)
		if err != nil {
			return "", microerror.Mask(err)
		}
		documents, err := appcr.Documents(data, filePath)
		if err != nil {
			return "", microerror.Mask(err)
		}

		for _, document := range documents {
			var object struct {
				APIVersion string `json:"apiVersion"`
				Kind       string `json:"kind"`
				Spec       struct {
					URL string `json:"url"`
				} `json:"spec"`
			}
			err = json.Unmarshal(document, &object)
			if err != nil {
				return "", microerror.Maskf(invalidConfigError, "cannot parse %s: %s", filePath, err)
			}

			switch {
			case appcr.IsApp(object.APIVersion, object.Kind):
				return "", microerror.Maskf(invalidConfigError, "base %s deploys an App CR in %s, which kubectl-gs no longer supports. Move it to a HelmRelease first, see %s", baseDir, filePath, appcr.MigrationDocs)
			case object.Kind == "OCIRepository" && strings.HasPrefix(object.APIVersion, "source.toolkit.fluxcd.io/"):
				chart := path.Base(object.Spec.URL)
				if strings.HasPrefix(chart, releaseChartPrefix) {
					charts = append(charts, chart)
				}
			}
		}
	}

	if len(charts) > 1 {
		return "", microerror.Maskf(invalidConfigError, "base %s has more than one release chart: %s", baseDir, strings.Join(charts, ", "))
	}
	if len(charts) == 0 {
		return "", nil
	}

	return charts[0], nil
}

const releaseChartPrefix = "release-"
