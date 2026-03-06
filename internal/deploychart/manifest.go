package deploychart

import (
	"encoding/json"
	"fmt"
	"time"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	"github.com/fluxcd/pkg/apis/meta"
	sourcev1beta2 "github.com/fluxcd/source-controller/api/v1beta2"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type OCIRepositoryOptions struct {
	Name        string
	Namespace   string
	ClusterName string
	URL         string
	Version     string
	AutoUpgrade string
	Interval    string
}

type HelmReleaseOptions struct {
	Name              string
	Namespace         string
	ClusterName       string
	ChartName         string
	TargetNamespace   string
	Interval          string
	Values            map[string]any
	ManagementCluster bool
}

func BuildOCIRepository(opts OCIRepositoryOptions) *sourcev1beta2.OCIRepository {
	interval := parseDuration(opts.Interval)

	repo := &sourcev1beta2.OCIRepository{
		TypeMeta: metav1.TypeMeta{
			APIVersion: sourcev1beta2.GroupVersion.String(),
			Kind:       sourcev1beta2.OCIRepositoryKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      opts.Name,
			Namespace: opts.Namespace,
			Labels: map[string]string{
				"giantswarm.io/cluster": opts.ClusterName,
			},
		},
		Spec: sourcev1beta2.OCIRepositorySpec{
			Interval: metav1.Duration{Duration: interval},
			URL:      opts.URL,
			Provider: "generic",
		},
	}

	if opts.Version != "" {
		repo.Spec.Reference = buildOCIRef(opts.Version, opts.AutoUpgrade)
	}

	return repo
}

func BuildHelmRelease(opts HelmReleaseOptions) *helmv2.HelmRelease {
	interval := parseDuration(opts.Interval)

	hr := &helmv2.HelmRelease{
		TypeMeta: metav1.TypeMeta{
			APIVersion: helmv2.GroupVersion.String(),
			Kind:       helmv2.HelmReleaseKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      opts.Name,
			Namespace: opts.Namespace,
			Labels: map[string]string{
				"giantswarm.io/cluster": opts.ClusterName,
			},
		},
		Spec: helmv2.HelmReleaseSpec{
			Interval:        metav1.Duration{Duration: interval},
			ReleaseName:     opts.ChartName,
			TargetNamespace: opts.TargetNamespace,
			Install: &helmv2.Install{
				CreateNamespace: true,
			},
			ChartRef: &helmv2.CrossNamespaceSourceReference{
				Kind: sourcev1beta2.OCIRepositoryKind,
				Name: opts.Name,
			},
		},
	}

	if !opts.ManagementCluster {
		hr.Spec.KubeConfig = &meta.KubeConfigReference{
			SecretRef: meta.SecretKeyReference{
				Name: fmt.Sprintf("%s-kubeconfig", opts.ClusterName),
			},
		}
	}

	if opts.Values != nil {
		raw, _ := json.Marshal(opts.Values)
		hr.Spec.Values = &apiextensionsv1.JSON{Raw: raw}
	}

	return hr
}

func buildOCIRef(version, autoUpgrade string) *sourcev1beta2.OCIRepositoryRef {
	if autoUpgrade == "" {
		return &sourcev1beta2.OCIRepositoryRef{
			Tag: version,
		}
	}

	return &sourcev1beta2.OCIRepositoryRef{
		SemVer: SemverRange(version, autoUpgrade),
	}
}

// SemverRange computes a semver wildcard constraint for auto-upgrade
// using Masterminds/semver wildcard syntax.
func SemverRange(version, autoUpgrade string) string {
	major, minor, _ := parseSemver(version)

	switch autoUpgrade {
	case "patch":
		return fmt.Sprintf("%d.%d.x", major, minor)
	case "minor":
		return fmt.Sprintf("%d.x", major)
	case "all":
		return "*"
	}
	return version
}

// parseSemver extracts major, minor, patch from a version string.
// It handles versions with or without a "v" prefix.
func parseSemver(version string) (int, int, int) {
	if len(version) > 0 && version[0] == 'v' {
		version = version[1:]
	}
	var major, minor, patch int
	fmt.Sscanf(version, "%d.%d.%d", &major, &minor, &patch)
	return major, minor, patch
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 10 * time.Minute
	}
	return d
}
