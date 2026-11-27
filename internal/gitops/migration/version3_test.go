package migration

import (
	"strings"
	"testing"
)

func Test_Version3_ManualStepsPointAtTheTemplate(t *testing.T) {
	for _, s := range []string{"prune: true", "docs/how-to/upgrade-fork.md#move-the-workload-cluster-kustomizations", "three commits"} {
		if !strings.Contains(version3.ManualSteps, s) {
			t.Errorf("expected the manual steps to mention %q", s)
		}
	}
	if strings.Contains(version3.ManualSteps, "sourceRef") {
		t.Errorf("expected no paraphrase of the steps, got:\n%s", version3.ManualSteps)
	}
}
