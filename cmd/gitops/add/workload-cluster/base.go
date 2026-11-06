package wcluster

import (
	"bytes"
	"encoding/json"
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
// generates HelmReleases. That includes an overlay whose base below does,
// like gitops-template's bases/clusters/capa/release, which patches the App to
// a release chart.
func clusterChartName(fs afero.Fs, baseDir string) (string, error) {
	err := refuseClusterApps(fs, baseDir)
	if err != nil {
		return "", microerror.Mask(err)
	}

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

		objects, err := decodeObjects(data, filePath)
		if err != nil {
			return "", microerror.Mask(err)
		}
		for _, object := range objects {
			if object.Kind == "OCIRepository" && strings.HasPrefix(object.APIVersion, "source.toolkit.fluxcd.io/") {
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

type object struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Spec       struct {
		Catalog string `json:"catalog"`
		Name    string `json:"name"`
		URL     string `json:"url"`
	} `json:"spec"`
}

func decodeObjects(data []byte, filePath string) ([]object, error) {
	var objects []object
	decoder := apiyaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), 4096)
	for {
		var document json.RawMessage
		err := decoder.Decode(&document)
		if err == io.EOF {
			return objects, nil
		} else if err != nil {
			return nil, microerror.Maskf(invalidConfigError, "cannot parse %s: %s", filePath, err)
		}

		// A JSON 6902 patch file in the base is a list of operations, not a
		// resource. Empty documents hold nothing either.
		trimmed := bytes.TrimSpace(document)
		if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) || trimmed[0] == '[' {
			continue
		}

		var o object
		err = json.Unmarshal(document, &o)
		if err != nil {
			return nil, microerror.Maskf(invalidConfigError, "cannot parse %s: %s", filePath, err)
		}
		objects = append(objects, o)
	}
}

// refuseClusterApps follows the base's resources, so an overlay is refused
// when the base below deploys the cluster with an App CR.
func refuseClusterApps(fs afero.Fs, baseDir string) error {
	files, err := appcr.Files(fs, baseDir)
	if err != nil {
		return microerror.Mask(err)
	}

	for _, filePath := range files {
		data, err := afero.ReadFile(fs, filePath)
		if err != nil {
			return microerror.Mask(err)
		}
		objects, err := decodeObjects(data, filePath)
		if err != nil {
			return microerror.Mask(err)
		}
		for _, o := range objects {
			if appcr.IsApp(o.APIVersion, o.Kind) && isClusterApp(o.Spec.Catalog, o.Spec.Name) {
				return microerror.Maskf(invalidConfigError, "base %s deploys the cluster with an App CR in %s, which kubectl-gs no longer supports. Move the cluster to a %s HelmRelease first, see %s", baseDir, filePath, releaseChartPrefix+"<provider>", appcr.MigrationDocs)
			}
		}
	}

	return nil
}

// isClusterApp follows app-admission-controller, which only treats Apps from
// the cluster catalogs as cluster Apps. A `cluster-autoscaler` App from
// another catalog isn't one.
func isClusterApp(catalog, chart string) bool {
	if catalog != "cluster" && catalog != "cluster-test" {
		return false
	}
	return strings.HasPrefix(chart, "cluster-") || strings.HasPrefix(chart, releaseChartPrefix)
}
