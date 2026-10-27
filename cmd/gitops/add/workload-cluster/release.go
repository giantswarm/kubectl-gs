package wcluster

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/giantswarm/microerror"
	releasev1alpha1 "github.com/giantswarm/releases/sdk/api/v1alpha1"
	"github.com/spf13/afero"
	apiyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"
)

const (
	releaseURLFormat         = "https://raw.githubusercontent.com/giantswarm/releases/master/%s/v%s/release.yaml"
	archivedReleaseURLFormat = "https://raw.githubusercontent.com/giantswarm/releases/master/%s/archived/v%s/release.yaml"
	fetchTimeout             = 30 * time.Second
)

// fetchFunc returns the body of the document at url. It returns an error
// matched by IsNotFound when the server answers with 404.
type fetchFunc func(ctx context.Context, url string) ([]byte, error)

func httpFetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	client := &http.Client{Timeout: fetchTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, microerror.Maskf(notFoundError, "%s was not found", url)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, microerror.Maskf(fetchFailedError, "fetching %s returned %s", url, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return body, nil
}

// clusterChartName returns the chart name of the cluster App in the base
// directory, for example `cluster-aws`.
func clusterChartName(fs afero.Fs, baseDir string) (string, error) {
	files, err := afero.ReadDir(fs, baseDir)
	if err != nil {
		return "", microerror.Mask(err)
	}

	for _, file := range files {
		ext := filepath.Ext(file.Name())
		if file.IsDir() || (ext != ".yaml" && ext != ".yml") {
			continue
		}

		data, err := afero.ReadFile(fs, filepath.Join(baseDir, file.Name()))
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
				return "", microerror.Maskf(invalidBaseError, "cannot parse %s: %s", filepath.Join(baseDir, file.Name()), err)
			}

			if object.Kind != "App" || !strings.HasPrefix(object.APIVersion, "application.giantswarm.io/") {
				continue
			}
			if object.Spec.Name == "" {
				return "", microerror.Maskf(invalidBaseError, "the App in %s has no spec.name", filepath.Join(baseDir, file.Name()))
			}

			return object.Spec.Name, nil
		}
	}

	return "", microerror.Maskf(invalidBaseError, "base %s has no App resource", baseDir)
}

// releaseDirectory returns the giantswarm/releases directory that holds the
// releases of a cluster chart: capa for cluster-aws, and the provider name for
// the others, such as azure, vsphere, cloud-director, eks and aks.
func releaseDirectory(chartName string) string {
	if chartName == "cluster-aws" {
		return "capa"
	}
	return strings.TrimPrefix(chartName, "cluster-")
}

// releaseClusterAppVersion returns the version of the cluster chart that a
// release ships, from the release's Release resource in giantswarm/releases.
// Older releases are moved to the provider's archived directory.
func releaseClusterAppVersion(ctx context.Context, fetch fetchFunc, chartName, release string) (string, error) {
	release = strings.TrimPrefix(release, "v")
	dir := releaseDirectory(chartName)

	data, err := fetch(ctx, fmt.Sprintf(releaseURLFormat, dir, release))
	if IsNotFound(err) {
		data, err = fetch(ctx, fmt.Sprintf(archivedReleaseURLFormat, dir, release))
	}
	if IsNotFound(err) {
		return "", microerror.Maskf(notFoundError, "release v%s was not found for provider %s in giantswarm/releases, pass --%s to skip the lookup", release, dir, flagClusterAppVersion)
	} else if err != nil {
		return "", microerror.Mask(err)
	}

	var r releasev1alpha1.Release
	err = yaml.Unmarshal(data, &r)
	if err != nil {
		return "", microerror.Mask(err)
	}

	component, ok := r.LookupComponentSpec(chartName)
	if !ok || component.Version == "" {
		return "", microerror.Maskf(notFoundError, "release v%s for provider %s has no %s component", release, dir, chartName)
	}

	return component.Version, nil
}
