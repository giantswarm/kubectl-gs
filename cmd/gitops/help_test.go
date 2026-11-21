package gitops

import (
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
	},
}

var templateDocsURL = regexp.MustCompile(`https://github\.com/giantswarm/gitops-template/blob/main/([^\s#]+?)(?:#([^\s.]+))?\.?(?:\s|$)`)

// stale are what the help must no longer mention: the `mapi` directory and
// the cluster base's `template` directory are gone from the repository
// structure.
var stale = []string{
	"mapi",
	"bases/clusters/capa/template",
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

			checkStale(t, help)
			checkFlags(t, c)
			checkLinks(t, help)
		})
	})
}

func checkStale(t *testing.T, help string) {
	t.Helper()

	for _, s := range stale {
		if strings.Contains(help, s) {
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

func walk(c *cobra.Command, fn func(*cobra.Command)) {
	fn(c)
	for _, sub := range c.Commands() {
		walk(sub, fn)
	}
}
