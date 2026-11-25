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

// Test_Layout_SopsRules checks that a cluster isn't upgraded while a
// .sops.yaml rule picks the key for files under mapi/ that it doesn't pick
// once they have moved up: sops would refuse new encrypted files there, or
// encrypt them with another rule's key.
func Test_Layout_SopsRules(t *testing.T) {
	testCases := []struct {
		name    string
		sops    string
		refused bool
	}{
		{name: "no .sops.yaml"},
		{name: "a rule for every cluster", sops: sopsRulesOf + mcRule + broadRule},
		{name: "a rule for mapi/ only", sops: sopsRulesOf + mcRule + mapiRule, refused: true},
		{name: "a rule for mapi/ before a broader one", sops: sopsRulesOf + mapiRule + broadRule, refused: true},
		{name: "a rule for both", sops: sopsRulesOf + bothRule},
		{name: "not a sops config", sops: "creation_rules: nope\n", refused: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			files := v2MapiCluster()
			files[layoutWC+"/mapi/apps/hello-world/secret.enc.yaml"] = secret
			if tc.sops != "" {
				files[".sops.yaml"] = tc.sops
			}
			fs := layoutRepo(t, files)
			layer := metadata.Layer{Kind: metadata.LayerWorkloadCluster, Path: layoutWC, StructureVersion: 2}

			_, err := version3.Apply(fs, testRepo, layer, ignoring)
			if !tc.refused {
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
				return
			}

			if !IsUnexpectedFile(err) || !strings.Contains(err.Error(), ".sops.yaml") {
				t.Fatalf("expected the upgrade refused, naming .sops.yaml, got: %v", err)
			}
			if strings.Contains(tc.sops, mapiRule) && !strings.Contains(err.Error(), layoutWC+"/mapi/.*\\.enc\\.yaml") {
				t.Errorf("expected the rule named, got: %s", err)
			}
			if ok, _ := fs.DirExists(filepath.Join(testRepo, layoutWC, "mapi", "apps")); !ok {
				t.Errorf("expected mapi left as it was")
			}
		})
	}
}
