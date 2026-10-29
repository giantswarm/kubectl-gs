package wcluster

import (
	"bytes"
	"io"
	"path/filepath"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	apiyaml "k8s.io/apimachinery/pkg/util/yaml"
)

// clusterChartName returns the chart of the cluster App in the base directory,
// for example `cluster-aws` or `release-aws`. Other Apps the base may ship are
// ignored. It returns an empty string when the base has no cluster App.
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

		path := filepath.Join(baseDir, file.Name())
		data, err := afero.ReadFile(fs, path)
		if err != nil {
			return "", microerror.Mask(err)
		}

		decoder := apiyaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), 4096)
		for {
			var object struct {
				APIVersion string `json:"apiVersion"`
				Kind       string `json:"kind"`
				Spec       struct {
					Name string `json:"name"`
				} `json:"spec"`
			}
			err = decoder.Decode(&object)
			if err == io.EOF {
				break
			} else if err != nil {
				return "", microerror.Maskf(invalidConfigError, "cannot parse %s: %s", path, err)
			}

			if object.Kind != "App" || !strings.HasPrefix(object.APIVersion, "application.giantswarm.io/") {
				continue
			}
			if isClusterChart(object.Spec.Name) {
				charts = append(charts, object.Spec.Name)
			}
		}
	}

	if len(charts) > 1 {
		return "", microerror.Maskf(invalidConfigError, "base %s has more than one cluster App: %s", baseDir, strings.Join(charts, ", "))
	}
	if len(charts) == 0 {
		return "", nil
	}

	return charts[0], nil
}

func isClusterChart(name string) bool {
	return strings.HasPrefix(name, "cluster-") || isReleaseChart(name)
}

// isReleaseChart tells whether the chart is a release-<provider> chart. Its
// version is the release version, and it carries `global.release.version` in
// its default values.
func isReleaseChart(name string) bool {
	return strings.HasPrefix(name, "release-")
}
