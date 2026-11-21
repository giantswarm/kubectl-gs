package gitops

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// templateDocs are the gitops-template pages the help may link to, with the
// anchors each one has. The pages the help linked to before are stubs that
// point here.
var templateDocs = map[string][]string{
	"docs/tutorials/basic/getting-started.md": {
		"1-connect-the-management-cluster",
		"2-add-an-organisation",
		"3-create-a-workload-cluster",
	},
	"docs/how-to/kubectl-gs.md": {
		"management-cluster",
		"organisation",
		"workload-cluster",
		"app",
		"automatic-updates",
	},
	"docs/how-to/sops.md": {
		"add-a-workload-cluster-key-gpg",
	},
	"docs/reference/repo-structure.md": {
		"",
		"structure-version",
		"flux-kustomizations",
		"move-the-workload-cluster-kustomizations",
	},
}

var templateDocsURL = regexp.MustCompile("https://github\\.com/giantswarm/gitops-template/blob/main/([^\\s#\"`]+?)(?:#([^\\s.\"`]+))?\\.?(?:[\\s\"`]|$)")

// stale are what the help must no longer mention: the `mapi` directory and
// the cluster base's `template` directory are gone from the repository
// structure, and an organisation keeps its Secrets in its namespace.
var stale = []string{
	"mapi",
	"bases/clusters/capa/template",
	"organizations/demoorg/secrets/",
}

func Test_help(t *testing.T) {
	logger, err := micrologger.New(micrologger.Config{})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// Building the commands reads no cluster: the config flags are only
	// held for the commands that run against one.
	var configFlags genericclioptions.RESTClientGetter = genericclioptions.NewConfigFlags(true)

	cmd, err := New(Config{
		Logger:      logger,
		ConfigFlags: &configFlags,
		FileSystem:  afero.NewMemMapFs(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	walk(cmd, func(c *cobra.Command) {
		t.Run(c.CommandPath(), func(t *testing.T) {
			help := c.Long + "\n" + c.Example

			checkStale(t, c.CommandPath(), help)
			checkFlags(t, c)
			checkLinks(t, help)
			checkWrap(t, c)
		})
	})
}

// staleAllowed are the stale words a command may still use: the upgrade
// explains what it does to the `mapi` directory.
var staleAllowed = map[string]string{
	"gitops upgrade": "mapi",
}

func checkStale(t *testing.T, command, help string) {
	t.Helper()

	for _, s := range stale {
		if strings.Contains(help, s) && staleAllowed[command] != s {
			t.Errorf("help mentions %q", s)
		}
	}
}

func checkFlags(t *testing.T, c *cobra.Command) {
	t.Helper()

	c.Flags().VisitAll(func(f *pflag.Flag) {
		if strings.Contains(f.Name, "mapi") || strings.Contains(f.Usage, "mapi") {
			t.Errorf("flag --%s refers to the mapi directory", f.Name)
		}
	})
}

func checkLinks(t *testing.T, help string) {
	t.Helper()

	for _, m := range templateDocsURL.FindAllStringSubmatch(help, -1) {
		link := strings.TrimSpace(m[0])

		anchors, ok := templateDocs[m[1]]
		if !ok {
			t.Errorf("help links to %s, which is not one of the current gitops-template pages", link)
			continue
		}
		if !slices.Contains(anchors, m[2]) {
			t.Errorf("help links to %s, whose anchor %q is not on the page", link, m[2])
		}
	}
}

// checkWrap fails on a line of prose longer than 80 characters. Links,
// command lines and the short description, which is a line of its own, may
// be longer.
func checkWrap(t *testing.T, c *cobra.Command) {
	t.Helper()

	for _, line := range strings.Split(c.Long, "\n") {
		if len(line) > 80 && line != c.Short && !strings.Contains(line, "https://") && !strings.HasPrefix(line, "kubectl gs") {
			t.Errorf("help line longer than 80 characters: %q", line)
		}
	}
}

// Test_messageLinks checks the gitops-template links the gitops commands and
// packages print in messages and errors, not only in their help.
func Test_messageLinks(t *testing.T) {
	for _, root := range []string{".", "../../internal/gitops"} {
		sources := os.DirFS(root)
		err := fs.WalkDir(sources, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}

			data, err := fs.ReadFile(sources, path)
			if err != nil {
				return err
			}

			t.Run(filepath.Join(root, path), func(t *testing.T) {
				checkLinks(t, string(data))
			})

			return nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
	}
}

func walk(c *cobra.Command, fn func(*cobra.Command)) {
	fn(c)
	for _, sub := range c.Commands() {
		walk(sub, fn)
	}
}
