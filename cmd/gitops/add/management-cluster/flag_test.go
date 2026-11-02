package mcluster

import (
	"testing"

	"github.com/giantswarm/microerror"
)

func TestFlagValidate(t *testing.T) {
	tests := []struct {
		name    string
		flag    flag
		wantErr bool
	}{
		{
			name: "skip encryption",
			flag: flag{
				Name:           "demomc",
				RepositoryName: "gitops-demo",
				SkipEncryption: true,
			},
		},
		{
			name: "generate master key",
			flag: flag{
				GenerateMasterKey: true,
				Name:              "demomc",
				RepositoryName:    "gitops-demo",
			},
		},
		{
			name: "generate master key and skip encryption",
			flag: flag{
				GenerateMasterKey: true,
				Name:              "demomc",
				RepositoryName:    "gitops-demo",
				SkipEncryption:    true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.flag.Validate()
			if tt.wantErr {
				if microerror.Cause(err) != invalidFlagsError {
					t.Fatalf("expected invalidFlagsError, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}
		})
	}
}
