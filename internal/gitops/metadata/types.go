// Package metadata keeps track of what `kubectl gs gitops` has generated in a
// GitOps repository, and of the version of the repository structure it was
// generated with.
//
// The information is kept in a single file at the repository root, rather than
// in the generated files themselves, for three reasons:
//
//  1. post modifiers round trip YAML through sigs.k8s.io/yaml, which drops
//     comments, so a header comment would not survive,
//  2. the creator never overwrites an already existing file, so a marker
//     written once could never be refreshed,
//  3. the creator tells directories from files by the length of their data,
//     so adding bytes to every object would break that.
//
// The file lives outside the scope Flux reconciles, which starts at
// `management-clusters/MC_NAME`, and is therefore never applied to a cluster.
package metadata

const (
	// APIVersion and Kind make the metadata file self describing, and give us
	// room to change its schema later on.
	APIVersion = "gitops.giantswarm.io/v1alpha1"
	Kind       = "RepositoryMetadata"

	// FileName is the name of the metadata file, relative to the repository root.
	FileName = ".gitops-metadata.yaml"

	// StructureVersion is the version of the repository structure this
	// kubectl-gs generates. Bump it whenever the generated structure changes,
	// and record what changed in the gitops-template CHANGELOG, so that
	// `kubectl gs gitops check` can tell users their repository is behind.
	StructureVersion = 1
)

// Layer kinds. These name the units `kubectl gs gitops add` creates, and are
// the granularity at which the structure version is tracked.
const (
	LayerClusterBase       = "cluster-base"
	LayerManagementCluster = "management-cluster"
	LayerOrganization      = "organization"
	LayerWorkloadCluster   = "workload-cluster"
	LayerApp               = "app"
)

// RepositoryMetadata is the content of the repository metadata file.
type RepositoryMetadata struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`

	// StructureVersion is the highest structure version any part of this
	// repository has been generated with.
	StructureVersion int `json:"structureVersion"`

	// GeneratedWith names the kubectl-gs build that last touched the
	// repository, e.g. `kubectl-gs/5.7.3`. Informational only, and omitted
	// from a file nothing has generated yet, such as the one the
	// gitops-template repository ships to pin forks at a structure version.
	GeneratedWith string `json:"generatedWith,omitempty"`

	// Layers records every part of the repository kubectl-gs has generated.
	// Kept sorted by path, so that the file produces a stable diff.
	Layers []Layer `json:"layers"`
}

// Layer records one generated part of the repository.
type Layer struct {
	Kind string `json:"kind"`

	// Path is the directory the layer owns, relative to the repository root.
	Path string `json:"path"`

	StructureVersion int    `json:"structureVersion"`
	GeneratedWith    string `json:"generatedWith"`
}
