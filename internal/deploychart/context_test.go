package deploychart

import "testing"

func TestClusterNameFromContext(t *testing.T) {
	tests := []struct {
		context string
		want    string
	}{
		{"gs-golem", "golem"},
		{"teleport.giantswarm.io-golem", "golem"},
		{"gs-grizzly", "grizzly"},
		{"teleport.giantswarm.io-grizzly", "grizzly"},
		{"custom-context", "custom-context"},
		{"mymc01", "mymc01"},
		// Edge: prefix alone without name returns full context.
		{"gs-", "gs-"},
	}

	for _, tc := range tests {
		t.Run(tc.context, func(t *testing.T) {
			got := ClusterNameFromContext(tc.context)
			if got != tc.want {
				t.Errorf("ClusterNameFromContext(%q) = %q, want %q", tc.context, got, tc.want)
			}
		})
	}
}
