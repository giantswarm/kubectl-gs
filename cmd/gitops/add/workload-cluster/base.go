package wcluster

import (
	"bytes"
	"io"
	"path"
	"path/filepath"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	apiyaml "k8s.io/apimachinery/pkg/util/yaml"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/appcr"
)

// clusterChartName returns the release chart the base deploys, for example
// `release-aws`, read from the URL of its OCIRepository. Other OCIRepositories
// the base may ship are ignored. It returns an empty string when the base has
// no release chart, for example when it is an overlay of another base.
//
// A base deploying the cluster with an App CR is refused, as kubectl-gs only
// generates HelmReleases.
func clusterChartName(fs afero.Fs, baseDir string) (string, error) {
	files, err := afero.ReadDir(fs, baseDir)
	if err != nil {
		return "", microerror.Mask(err)
	}

	var charts []string
	for _, file := range files {
		ext := filepath.Ext(file.Name())
		if file.IsDir() || (ext != ".yaml" && ext != ".yml") {
			continue
		}

		filePath := filepath.Join(baseDir, file.Name())
		data, err := afero.ReadFile(fs, filePath)
		if err != nil {
			return "", microerror.Mask(err)
		}

		decoder := apiyaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), 4096)
		for {
			var object struct {
				APIVersion string `json:"apiVersion"`
				Kind       string `json:"kind"`
				Spec       struct {
					Catalog string `json:"catalog"`
					Name    string `json:"name"`
					URL     string `json:"url"`
				} `json:"spec"`
			}
			err = decoder.Decode(&object)
			if err == io.EOF {
				break
			} else if err != nil {
				return "", microerror.Maskf(invalidConfigError, "cannot parse %s: %s", filePath, err)
			}

			switch {
			case appcr.IsApp(object.APIVersion, object.Kind):
				if isClusterApp(object.Spec.Catalog, object.Spec.Name) {
					return "", microerror.Maskf(invalidConfigError, "base %s deploys the cluster with an App CR in %s, which kubectl-gs no longer supports. Move the cluster to a %s HelmRelease first, see %s", baseDir, filePath, releaseChartPrefix+"<provider>", appcr.MigrationDocs)
				}
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

// isClusterApp follows app-admission-controller, which only treats Apps from
// the cluster catalogs as cluster Apps. A `cluster-autoscaler` App from
// another catalog isn't one.
func isClusterApp(catalog, chart string) bool {
	if catalog != "cluster" && catalog != "cluster-test" {
		return false
	}
	return strings.HasPrefix(chart, "cluster-") || strings.HasPrefix(chart, releaseChartPrefix)
}
