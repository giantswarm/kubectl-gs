package migration

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	"go.yaml.in/yaml/v4"

	gitopskey "github.com/giantswarm/kubectl-gs/v6/internal/gitops/key"
)

const sopsRuleMoved = "%s: the rule with path_regex `%s` picks the key for %s, but not for %s, where the upgrade moves it: sops would refuse new encrypted files there, or encrypt them with another rule's key. Files already encrypted keep their keys either way. Change the rule so that it matches both places, for example by replacing mapi/ with (mapi/)?(%s)/, which keeps it out of out-of-band/, and run the upgrade again. Once it has run, (mapi/)? can go"

// sopsConfig is what the migration reads of a .sops.yaml.
type sopsConfig struct {
	CreationRules []struct {
		PathRegex string `yaml:"path_regex"`
	} `yaml:"creation_rules"`
}

// refuseSopsRules fails when the .sops.yaml sops uses for the cluster picks
// the key for a file under `mapi` with a rule that doesn't pick it once the
// file has moved up. It never changes .sops.yaml: it isn't part of any
// layer, and a rule can't be rewritten safely without knowing what the user
// meant it to match.
func refuseSopsRules(fs *afero.Afero, repoPath, wcDir string, files []string) error {
	configPath, err := findSopsConfig(fs, repoPath, wcDir)
	if err != nil || configPath == "" {
		return microerror.Mask(err)
	}

	rules, err := sopsRules(fs, repoPath, configPath)
	if err != nil {
		return microerror.Mask(err)
	}

	configDir := filepath.Dir(configPath)
	mapiDir := filepath.Join(wcDir, gitopskey.MapiDirName())
	for _, f := range files {
		// The file itself, and a new encrypted file next to it.
		for _, name := range []string{f, filepath.Join(filepath.Dir(f), "new.enc.yaml")} {
			from := filepath.ToSlash(rel(configDir, filepath.Join(mapiDir, name)))
			to := filepath.ToSlash(rel(configDir, filepath.Join(wcDir, name)))

			before, after := firstRule(rules, from), firstRule(rules, to)
			if before >= 0 && before != after {
				return microerror.Maskf(unexpectedFileError, sopsRuleMoved, rel(repoPath, configPath), rules[before].String(), from, to, strings.Join(movedDirs(files), "|"))
			}
		}
	}

	return nil
}

// findSopsConfig returns the .sops.yaml sops finds for the files of the
// cluster: the first one in the cluster's directory or above it, up to the
// repository root. It returns an empty string when there is none.
func findSopsConfig(fs *afero.Afero, repoPath, wcDir string) (string, error) {
	for dir := wcDir; ; dir = filepath.Dir(dir) {
		path := filepath.Join(dir, gitopskey.SopsConfigFileName())
		exists, err := fs.Exists(path)
		if err != nil {
			return "", microerror.Mask(err)
		}
		if exists {
			return path, nil
		}
		if dir == repoPath || dir == filepath.Dir(dir) {
			return "", nil
		}
	}
}

// sopsRules returns the path_regex of each creation rule, in order. A rule
// without one matches every file, as in sops.
func sopsRules(fs *afero.Afero, repoPath, path string) ([]*regexp.Regexp, error) {
	data, err := fs.ReadFile(path)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var config sopsConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, microerror.Maskf(unexpectedFileError, "cannot read the creation rules of %s, so there is no telling which keys files get once they move up from mapi/: %s", rel(repoPath, path), err)
	}

	rules := make([]*regexp.Regexp, 0, len(config.CreationRules))
	for _, r := range config.CreationRules {
		re, err := regexp.Compile(r.PathRegex)
		if err != nil {
			return nil, microerror.Maskf(unexpectedFileError, "%s: path_regex `%s` is not a regular expression sops can use: %s", rel(repoPath, path), r.PathRegex, err)
		}
		rules = append(rules, re)
	}

	return rules, nil
}

// firstRule returns the index of the rule sops uses for the path, relative
// to the .sops.yaml, or -1 when none matches.
func firstRule(rules []*regexp.Regexp, path string) int {
	for i, r := range rules {
		if r.MatchString(path) {
			return i
		}
	}

	return -1
}

// movedDirs returns the directories that move up from `mapi`.
func movedDirs(files []string) []string {
	var dirs []string
	for _, name := range topLevel(files) {
		for _, f := range files {
			if strings.HasPrefix(filepath.ToSlash(f), name+"/") {
				dirs = append(dirs, name)
				break
			}
		}
	}

	return dirs
}
