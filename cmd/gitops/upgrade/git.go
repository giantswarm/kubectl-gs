package upgrade

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
)

var (
	// errGitNotFound means git is not installed.
	errGitNotFound = errors.New("git is not installed")

	// errNotTracked means git tracks nothing under the path: it is outside any
	// repository, or ignored by the one it is in.
	errNotTracked = errors.New("not tracked by git")
)

// dirtyPaths returns what `git status` reports as uncommitted under repoPath.
// Like any porcelain status, the paths are relative to the repository root.
//
// It fails with errNotTracked when git tracks nothing there, as `git status`
// then reports nothing either, even when there are local changes, and the
// restore command would have nothing to restore from. The errors are plain:
// the caller wraps them once.
func dirtyPaths(repoPath string) ([]string, error) {
	tracked, err := git(repoPath, "ls-files", "--", ".")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(tracked) == "" {
		return nil, errNotTracked
	}

	// Scoped to repoPath, like the restore command: a repository nested in a
	// larger one should not be held up by changes elsewhere in it.
	status, err := git(repoPath, "status", "--porcelain", "--untracked-files=normal", "--", ".")
	if err != nil {
		return nil, err
	}

	var paths []string
	for _, line := range strings.Split(strings.TrimRight(status, "\n"), "\n") {
		if line != "" {
			paths = append(paths, line)
		}
	}

	return paths, nil
}

func git(repoPath string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", repoPath, "-c", "core.quotepath=false"}, args...)...)
	cmd.Env = gitEnv(os.Environ())

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if errors.Is(err, exec.ErrNotFound) {
		return "", errGitNotFound
	}
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if strings.Contains(msg, "not a git repository") {
			return "", errNotTracked
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(msg+" "+err.Error()))
	}

	return stdout.String(), nil
}

// repositoryVars point git at a different repository than the one it is run
// in. git runs without them, and every git command suggested to the user unsets
// them, see gitPrefix, so that both see the same repository.
var repositoryVars = []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_COMMON_DIR", "GIT_NAMESPACE"}

// localeVars are replaced by LC_ALL=C.
var localeVars = []string{"LC_ALL", "LANG", "LANGUAGE"}

// gitEnv is the environment git runs in: the user's, but with the messages in
// English, which errNotTracked is detected from, and without the variables
// that point git at a different repository than the one at repoPath.
func gitEnv(environ []string) []string {
	env := make([]string, 0, len(environ)+1)
	for _, kv := range environ {
		name, _, _ := strings.Cut(kv, "=")
		if slices.Contains(repositoryVars, name) || slices.Contains(localeVars, name) {
			continue
		}
		env = append(env, kv)
	}

	return append(env, "LC_ALL=C")
}

// restoreHint is the command that undoes a failed upgrade. It is only safe to
// suggest because the upgrade started from a clean working tree.
func restoreHint(repoPath string) string {
	return gitCommand(repoPath, "checkout -- .") + " && " + gitCommand(repoPath, "clean -fd")
}

// gitCommand renders a git command for the user to run against repoPath.
func gitCommand(repoPath, args string) string {
	return fmt.Sprintf("%sgit -C %s %s", gitPrefix(), shellQuote(repoPath), args)
}

// gitPrefix unsets the repositoryVars the user's shell sets, as upgrade ran git
// without them: otherwise a suggested command would act on a different
// repository than the one that was checked.
func gitPrefix() string {
	var unset []string
	for _, name := range repositoryVars {
		if _, ok := os.LookupEnv(name); ok {
			unset = append(unset, "-u "+name)
		}
	}
	if len(unset) == 0 {
		return ""
	}

	return "env " + strings.Join(unset, " ") + " "
}

// shellQuote quotes a path for pasting into a POSIX shell, when it needs it.
func shellQuote(s string) string {
	if s != "" && !strings.ContainsAny(s, " \t\n'\"\\$`!*?[]{}()<>|&;#~") {
		return s
	}

	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
