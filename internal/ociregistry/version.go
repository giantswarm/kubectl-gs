package ociregistry

import (
	"fmt"
	"sort"

	"github.com/Masterminds/semver/v3"
)

// LatestSemverTag finds the latest semver-compatible tag from a list of tags.
// Tags that are not valid semver are silently ignored.
// Returns an error if no valid semver tags are found.
func LatestSemverTag(tags []string) (string, error) {
	var versions []*semver.Version

	for _, tag := range tags {
		v, err := semver.NewVersion(tag)
		if err != nil {
			continue
		}
		// Skip pre-release versions.
		if v.Prerelease() != "" {
			continue
		}
		versions = append(versions, v)
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("no valid semver tags found")
	}

	sort.Sort(semver.Collection(versions))

	latest := versions[len(versions)-1]

	return latest.Original(), nil
}
