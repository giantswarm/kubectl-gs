package wcluster

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	apiyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"
)

const appKind = "App"

// clusterChartName returns the chart of the cluster App in the base directory,
// for example `cluster-aws` or `release-aws`. Other Apps the base may ship are
// ignored. It returns an empty string when the base has no cluster App, for
// example when it is an overlay of another base, unless the overlay patches
// in a `release-<provider>` chart.
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
					Catalog string `json:"catalog"`
					Name    string `json:"name"`
				} `json:"spec"`
			}
			err = decoder.Decode(&object)
			if err == io.EOF {
				break
			} else if err != nil {
				return "", microerror.Maskf(invalidConfigError, "cannot parse %s: %s", path, err)
			}

			if object.Kind != appKind || !strings.HasPrefix(object.APIVersion, "application.giantswarm.io/") {
				continue
			}
			if isClusterApp(object.Spec.Catalog, object.Spec.Name) {
				charts = append(charts, object.Spec.Name)
			}
		}
	}

	if len(charts) > 1 {
		return "", microerror.Maskf(invalidConfigError, "base %s has more than one cluster App: %s", baseDir, strings.Join(charts, ", "))
	}
	if len(charts) == 0 {
		return releaseChartFromPatches(fs, baseDir)
	}

	return charts[0], nil
}

// isClusterApp follows app-admission-controller, which only treats Apps from
// the cluster catalogs as cluster Apps. A `cluster-autoscaler` App from
// another catalog isn't one.
func isClusterApp(catalog, chart string) bool {
	if catalog != "cluster" && catalog != "cluster-test" {
		return false
	}
	return strings.HasPrefix(chart, "cluster-") || strings.HasPrefix(chart, "release-")
}

// releaseChartFromPatches returns the `release-<provider>` chart that the
// kustomization of an overlay base patches into the App of the base below,
// as gitops-template's bases/clusters/capa/release does. Such a base needs
// neither values nor a reminder to set the release in them. Patches to a
// `cluster-<provider>` chart are left out on purpose: the overlay may set its
// own values, which the values written for that chart would replace.
func releaseChartFromPatches(fs afero.Fs, baseDir string) (string, error) {
	data, err := afero.ReadFile(fs, filepath.Join(baseDir, "kustomization.yaml"))
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	} else if err != nil {
		return "", microerror.Mask(err)
	}

	var kustomization struct {
		Patches []struct {
			Path   string `json:"path"`
			Patch  string `json:"patch"`
			Target struct {
				Kind string `json:"kind"`
			} `json:"target"`
		} `json:"patches"`
	}
	err = yaml.Unmarshal(data, &kustomization)
	if err != nil {
		return "", microerror.Maskf(invalidConfigError, "cannot parse the kustomization of %s: %s", baseDir, err)
	}

	for _, p := range kustomization.Patches {
		patch := []byte(p.Patch)
		if p.Path != "" {
			patch, err = afero.ReadFile(fs, filepath.Join(baseDir, p.Path))
			if err != nil {
				return "", microerror.Mask(err)
			}
		}

		// A JSON 6902 patch is a list of operations and applies to its target.
		var operations []struct {
			Op    string `json:"op"`
			Path  string `json:"path"`
			Value any    `json:"value"`
		}
		if yaml.Unmarshal(patch, &operations) == nil {
			if p.Target.Kind != appKind {
				continue
			}
			for _, o := range operations {
				chart, ok := o.Value.(string)
				if (o.Op == "replace" || o.Op == "add") && o.Path == "/spec/name" && ok && strings.HasPrefix(chart, "release-") {
					return chart, nil
				}
			}
			continue
		}

		// A strategic merge patch names its kind itself.
		var object struct {
			Kind string `json:"kind"`
			Spec struct {
				Name string `json:"name"`
			} `json:"spec"`
		}
		if yaml.Unmarshal(patch, &object) == nil && object.Kind == appKind && strings.HasPrefix(object.Spec.Name, "release-") {
			return object.Spec.Name, nil
		}
	}

	return "", nil
}
