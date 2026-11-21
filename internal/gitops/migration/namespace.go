package migration

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	"go.yaml.in/yaml/v4"
	apiyaml "k8s.io/apimachinery/pkg/util/yaml"

	gitopskey "github.com/giantswarm/kubectl-gs/v6/internal/gitops/key"
)

// managementCluster is what the organisation's namespace Kustomization takes
// from the management cluster's.
type managementCluster struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Spec struct {
		Decryption *struct {
			SecretRef struct {
				Name string `json:"name"`
			} `json:"secretRef"`
		} `json:"decryption"`
		SourceRef struct {
			Name string `json:"name"`
		} `json:"sourceRef"`
	} `json:"spec"`
}

// addNamespace gives an organisation its namespace Kustomization, applying
// an empty `namespace` directory, and the kustomization.yaml that lists it
// for the management cluster's Kustomization. The workload clusters'
// Kustomizations stay where they are, see version3.
func addNamespace(fs *afero.Afero, repoPath, orgPath string, env Env) ([]string, error) {
	orgDir := filepath.Join(repoPath, orgPath)

	var changes []string
	for _, step := range []func(*afero.Afero, string, string, Env) ([]string, error){
		addNamespaceKustomization,
		addNamespaceDirectory,
		listNamespaceKustomization,
	} {
		c, err := step(fs, repoPath, orgDir, env)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		changes = append(changes, c...)
	}

	return changes, nil
}

func addNamespaceKustomization(fs *afero.Afero, repoPath, orgDir string, _ Env) ([]string, error) {
	path := filepath.Join(orgDir, gitopskey.NamespaceFileName())
	if exists, err := fs.Exists(path); err != nil || exists {
		return nil, microerror.Mask(err)
	}

	mcDir := filepath.Dir(filepath.Dir(orgDir))
	mcFile := filepath.Join(mcDir, gitopskey.FluxKustomizationFileName(filepath.Base(mcDir)))

	var mc managementCluster
	data, err := fs.ReadFile(mcFile)
	if err == nil {
		err = apiyaml.Unmarshal(data, &mc)
	}
	if err != nil || mc.Metadata.Name == "" || mc.Spec.SourceRef.Name == "" {
		return nil, microerror.Maskf(unexpectedFileError, "can't read the name and the GitRepository of the management cluster's Kustomization from %s, add %s by hand, see %s", rel(repoPath, mcFile), rel(repoPath, path), namespaceDocs)
	}

	err = fs.WriteFile(path, []byte(namespaceKustomization(rel(repoPath, orgDir), filepath.Base(mcDir), mc)), 0644)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return []string{fmt.Sprintf("%s: added, the organisation's namespace Kustomization, applying namespace/", rel(repoPath, path))}, nil
}

func namespaceKustomization(orgPath, mcName string, mc managementCluster) string {
	org := filepath.Base(orgPath)

	decryption := ""
	if mc.Spec.Decryption != nil {
		decryption = fmt.Sprintf("  decryption:\n    provider: sops\n    secretRef:\n      name: %s\n", mc.Spec.Decryption.SecretRef.Name)
	}

	return fmt.Sprintf(`apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  labels:
    giantswarm.io/organization-namespace: "true"
  name: %[1]s-organization-%[2]s
  namespace: default
spec:
%[3]s  # Applies namespace/: everything the organisation keeps in its namespace. It
  # fails until organization-operator has created the namespace, then
  # succeeds on its next retry. The management cluster's Kustomization reads
  # the same GitRepository, so Flux applies a commit here only once that one
  # is Ready at it.
  dependsOn:
    - name: %[4]s
  interval: 10m
  path: "./%[5]s/namespace"
  postBuild:
    substitute:
      organization: "%[2]s"
  prune: true
  retryInterval: 1m
  serviceAccountName: automation
  sourceRef:
    kind: GitRepository
    name: %[6]s
  # Every object in namespace/ goes into the organisation's namespace.
  targetNamespace: org-%[2]s
  timeout: 2m
`, mcName, org, decryption, mc.Metadata.Name, filepath.ToSlash(orgPath), mc.Spec.SourceRef.Name)
}

func addNamespaceDirectory(fs *afero.Afero, repoPath, orgDir string, _ Env) ([]string, error) {
	path := filepath.Join(orgDir, gitopskey.NamespaceDirName(), gitopskey.SigsKustomizationFileName())
	if exists, err := fs.Exists(path); err != nil || exists {
		return nil, microerror.Mask(err)
	}

	err := fs.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	err = fs.WriteFile(path, []byte(`apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
# Everything the organisation keeps in its namespace, which the Kustomization
# in ../namespace.yaml applies into that namespace: the files it lists name no
# namespace. The workload clusters' Kustomizations, ../workload-clusters, go
# here once they have been moved, see the upgrade's manual steps.
resources: []
`), 0644)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return []string{fmt.Sprintf("%s: added, lists nothing yet", rel(repoPath, path))}, nil
}

// listNamespaceKustomization lists namespace.yaml in the organisation's
// kustomization.yaml, and writes one that lists what the management cluster's
// Kustomization applied from the directory, when there is none.
func listNamespaceKustomization(fs *afero.Afero, repoPath, orgDir string, env Env) ([]string, error) {
	path := filepath.Join(orgDir, gitopskey.SigsKustomizationFileName())

	exists, err := fs.Exists(path)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if exists {
		return addResource(fs, repoPath, path, gitopskey.NamespaceFileName())
	}

	names, err := listDir(fs, orgDir)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	var kept []string
	for _, n := range names {
		if n != gitopskey.NamespaceDirName() {
			kept = append(kept, n)
		}
	}

	resources, err := resourcesOf(fs, repoPath, orgDir, kept, env)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	for i, r := range resources {
		resources[i] = strings.TrimSuffix(r, "/")
	}
	sort.Strings(resources)

	err = fs.WriteFile(path, []byte(organizationKustomization(resources)), 0644)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return []string{fmt.Sprintf("%s: added, lists %s", rel(repoPath, path), strings.Join(resources, ", "))}, nil
}

func organizationKustomization(resources []string) string {
	var b strings.Builder
	b.WriteString("apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\n")
	b.WriteString("# The management cluster's Kustomization applies these. What goes into the\n# organisation's namespace is what namespace/kustomization.yaml lists, which\n# the Kustomization in namespace.yaml applies.\nresources:\n")
	for _, r := range resources {
		fmt.Fprintf(&b, "  - %s\n", r)
	}

	return b.String()
}

// addResource adds the resource to the kustomization.yaml at path, unless it
// lists it already.
func addResource(fs *afero.Afero, repoPath, path, resource string) ([]string, error) {
	kustomization, err := readYAML(fs, path)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	resources := mapValue(kustomization, "resources")
	if resources == nil {
		resources = &yaml.Node{Kind: yaml.SequenceNode}
		kustomization.Content = append(kustomization.Content, scalar("resources"), resources)
	}
	for _, r := range sequence(resources) {
		if strings.TrimPrefix(r.Value, "./") == resource {
			return nil, nil
		}
	}
	resources.Style = 0
	resources.Content = append(resources.Content, scalar(resource))

	err = writeYAML(fs, path, kustomization)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return []string{fmt.Sprintf("%s: lists %s", rel(repoPath, path), resource)}, nil
}
