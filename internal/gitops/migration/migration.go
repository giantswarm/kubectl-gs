package migration

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/metadata"
)

// unversionedHint says how to recover from entries recorded without a
// structure version. It is the one case that calls for editing the metadata
// file by hand.
var unversionedHint = "If you know the version they were generated with, set their structureVersion in " + metadata.FileName +
	" by hand. Otherwise remove their entries and run `kubectl gs gitops check --adopt` to record them as current."

// NewPlan works out the steps that bring every layer of md to target, using
// the given migrations. It fails without planning anything when a layer cannot
// be brought there.
func NewPlan(md *metadata.RepositoryMetadata, migrations []Migration, target int) (*Plan, error) {
	byFrom := map[int]Migration{}
	for _, m := range migrations {
		byFrom[m.From] = m
	}

	if md.StructureVersion > target {
		return nil, microerror.Maskf(
			newerLayerError,
			"the repository is recorded at repository structure version %d, newer than this kubectl-gs knows (version %d). Upgrade kubectl-gs first.",
			md.StructureVersion,
			target,
		)
	}

	// Checked over every entry, duplicates included: collapsing them first
	// would hide an entry recorded at a newer version behind an older one.
	var unversioned, newer []string
	for _, l := range md.Layers {
		switch {
		case l.StructureVersion <= 0:
			unversioned = append(unversioned, l.Path)
		case l.StructureVersion > target:
			newer = append(newer, l.Path)
		}
	}
	unversioned, newer = uniq(unversioned), uniq(newer)
	if len(unversioned) > 0 {
		return nil, microerror.Maskf(
			unversionedLayerError,
			"%d layer(s) are recorded without a structure version, so there is no telling how to upgrade them:\n  %s\n%s",
			len(unversioned),
			strings.Join(unversioned, "\n  "),
			unversionedHint,
		)
	}
	if len(newer) > 0 {
		return nil, microerror.Maskf(
			newerLayerError,
			"%d layer(s) were generated with a newer repository structure than this kubectl-gs knows (version %d). Upgrade kubectl-gs first. The layers:\n  %s",
			len(newer),
			target,
			strings.Join(newer, "\n  "),
		)
	}

	// The repository level version is only the highest version any part of
	// it was generated with. When it is missing but every layer has one, the
	// layers say all there is to know, and recording the plan sets it. With
	// no layers to go by, there is no telling.
	if md.StructureVersion <= 0 && len(md.Layers) == 0 {
		return nil, microerror.Maskf(
			unversionedRepositoryError,
			"the repository is recorded without a structure version, and without any layers to tell it from.\nIf you know the version it was generated with, set the top level structureVersion in %s by hand.",
			metadata.FileName,
		)
	}

	layers := dedupe(md.Layers)

	plan := &Plan{Target: target}

	for _, l := range layers {
		if l.StructureVersion == target {
			continue
		}

		for v := l.StructureVersion; v < target; v++ {
			m, ok := byFrom[v]
			if !ok {
				return nil, microerror.Maskf(missingMigrationError, "no migration registered from repository structure version %d to %d.", v, v+1)
			}

			if appliesTo(m, l.Kind) {
				// The layer as the migration finds it: already at the version
				// the migration starts from.
				sl := l
				sl.StructureVersion = v
				plan.Steps = append(plan.Steps, Step{Layer: sl, Migration: m})
			}
		}

		plan.Layers = append(plan.Layers, l)
	}

	sort.SliceStable(plan.Steps, func(i, j int) bool {
		if plan.Steps[i].Migration.From != plan.Steps[j].Migration.From {
			return plan.Steps[i].Migration.From < plan.Steps[j].Migration.From
		}
		return plan.Steps[i].Layer.Path < plan.Steps[j].Layer.Path
	})

	return plan, nil
}

// Empty reports whether no layer needs upgrading. The repository level record
// may still be behind.
func (p *Plan) Empty() bool {
	return len(p.Layers) == 0
}

// Run applies the plan to the repository cloned at repoPath, and records the
// upgraded layers in md.
//
// The migrations write straight to fs. There is no rollback: when a step
// fails, the steps before it have already changed files, and undoing that is
// left to git, which is why `kubectl gs gitops upgrade` wants a clean working
// tree before it starts.
//
// With dryRun, the migrations run against an in-memory copy of the repository
// instead, and the result lists the files a real run would change. Nothing is
// written.
//
// It does not save md: the caller does, once the files are written.
func (p *Plan) Run(fs *afero.Afero, repoPath string, md *metadata.RepositoryMetadata, dryRun bool) (*Result, error) {
	result := &Result{}

	if len(p.Steps) > 0 {
		target := fs
		var before map[string]file

		if dryRun {
			var err error
			target, err = stage(fs, repoPath)
			if err != nil {
				return nil, microerror.Mask(err)
			}

			before, err = snapshot(target, repoPath)
			if err != nil {
				return nil, microerror.Mask(err)
			}
		}

		for _, s := range p.Steps {
			changes, err := s.Migration.Apply(target, repoPath, s.Layer)
			if err != nil {
				return nil, microerror.Mask(&StepError{Step: s, Err: err})
			}

			result.Changes = append(result.Changes, StepChanges{Step: s, Changes: changes})
		}

		if dryRun {
			after, err := snapshot(target, repoPath)
			if err != nil {
				return nil, microerror.Mask(err)
			}

			result.Files = changes(before, after)
		}
	}

	record(md, p)

	return result, nil
}

// record moves the upgraded layers to the plan's target in md, and collapses
// duplicate entries for the same layer, whether upgraded or not.
func record(md *metadata.RepositoryMetadata, p *Plan) {
	upgraded := map[layerKey]bool{}
	for _, l := range p.Layers {
		upgraded[key(l)] = true
	}

	layers := dedupe(md.Layers)
	for i, l := range layers {
		if upgraded[key(l)] {
			layers[i].StructureVersion = p.Target
			layers[i].GeneratedWith = metadata.GeneratedWith()
		}
	}
	md.Layers = layers
	md.Sort()

	// Not md.Stamp(): that moves the repository to the version this build
	// produces, where the plan's target is what was asked for.
	md.APIVersion = metadata.APIVersion
	md.Kind = metadata.Kind
	md.GeneratedWith = metadata.GeneratedWith()
	if md.StructureVersion < p.Target {
		md.StructureVersion = p.Target
	}
}

type layerKey struct {
	kind string
	path string
}

func key(l metadata.Layer) layerKey {
	return layerKey{kind: l.Kind, path: l.Path}
}

// dedupe returns the layers sorted by path, with a single entry per kind and
// path. When a layer is recorded more than once, e.g. after a merge conflict
// in the metadata file, the lowest version wins: upgrading it again is safer
// than skipping a migration it never got.
func dedupe(layers []metadata.Layer) []metadata.Layer {
	byKey := map[layerKey]metadata.Layer{}
	for _, l := range layers {
		if seen, ok := byKey[key(l)]; ok && seen.StructureVersion <= l.StructureVersion {
			continue
		}
		byKey[key(l)] = l
	}

	out := make([]metadata.Layer, 0, len(byKey))
	for _, l := range byKey {
		out = append(out, l)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Kind < out[j].Kind
	})

	return out
}

func appliesTo(m Migration, kind string) bool {
	return slices.Contains(m.Kinds, kind)
}

// uniq drops repeated paths, keeping the order they were found in.
func uniq(paths []string) []string {
	seen := map[string]bool{}
	out := paths[:0]
	for _, p := range paths {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}

	return out
}

// String renders a step for the upgrade report.
func (s Step) String() string {
	return fmt.Sprintf("%s %s: %d -> %d", s.Layer.Kind, s.Layer.Path, s.Migration.From, s.Migration.From+1)
}
