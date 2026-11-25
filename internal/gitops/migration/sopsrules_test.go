package migration

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/metadata"
)

const (
	mcRule      = "  - path_regex: management-clusters/demomc/secrets/.*\\.enc\\.yaml\n    pgp: AAAA\n"
	mapiRule    = "  - path_regex: " + layoutWC + "/mapi/.*\\.enc\\.yaml\n    pgp: BBBB\n"
	broadRule   = "  - path_regex: " + layoutWCs + "/.+/.*\\.enc\\.yaml\n    pgp: CCCC\n"
	bothRule    = "  - path_regex: " + layoutWC + "/(mapi/)?(apps|cluster)/.*\\.enc\\.yaml\n    pgp: BBBB\n"
	sopsRulesOf = "creation_rules:\n"
)

// upgradeWithSops runs the upgrade on a cluster with an encrypted file in
// mapi, and the given .sops.yaml, if any.
func upgradeWithSops(t *testing.T, sops string) (func() bool, error) {
	t.Helper()

	files := v2MapiCluster()
	files[layoutWC+"/mapi/apps/hello-world/secret.enc.yaml"] = secret
	if sops != "" {
		files[".sops.yaml"] = sops
	}
	fs := layoutRepo(t, files)
	layer := metadata.Layer{Kind: metadata.LayerWorkloadCluster, Path: layoutWC, StructureVersion: 2}

	_, err := version3.Apply(fs, testRepo, layer, ignoring)
	mapiKept := func() bool {
		ok, _ := fs.DirExists(filepath.Join(testRepo, layoutWC, "mapi", "apps"))
		return ok
	}

	return mapiKept, err
}

// Test_Layout_RefusesSopsRules checks that a cluster isn't upgraded while a
// .sops.yaml rule picks the key for files under mapi/ that it doesn't pick
// once they have moved up: sops would refuse new encrypted files there, or
// encrypt them with another rule's key.
func Test_Layout_RefusesSopsRules(t *testing.T) {
	testCases := map[string]struct {
		sops string
		rule string
	}{
		"a rule for mapi/ only":                 {sops: sopsRulesOf + mcRule + mapiRule, rule: layoutWC + "/mapi/.*\\.enc\\.yaml"},
		"a rule for mapi/ before a broader one": {sops: sopsRulesOf + mapiRule + broadRule, rule: layoutWC + "/mapi/.*\\.enc\\.yaml"},
		"not a sops config":                     {sops: "creation_rules: nope\n"},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			mapiKept, err := upgradeWithSops(t, tc.sops)
			if !IsUnexpectedFile(err) || !strings.Contains(err.Error(), ".sops.yaml") || !strings.Contains(err.Error(), tc.rule) {
				t.Fatalf("expected the upgrade refused, naming .sops.yaml and the rule %q, got: %v", tc.rule, err)
			}
			if !mapiKept() {
				t.Errorf("expected mapi left as it was")
			}
		})
	}
}

// Test_Layout_UpgradesWithSopsRules checks that rules picking the same key
// for files under mapi/ and where they move don't hold the upgrade up.
func Test_Layout_UpgradesWithSopsRules(t *testing.T) {
	testCases := map[string]string{
		"no .sops.yaml":            "",
		"a rule for every cluster": sopsRulesOf + mcRule + broadRule,
		"a rule for both":          sopsRulesOf + bothRule,
	}

	for name, sops := range testCases {
		t.Run(name, func(t *testing.T) {
			_, err := upgradeWithSops(t, sops)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
		})
	}
}
