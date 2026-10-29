package wcluster

import (
	"testing"

	"github.com/giantswarm/microerror"
)

func Test_flag_Validate(t *testing.T) {
	testCases := []struct {
		name          string
		flag          flag
		expectInvalid bool
	}{
		{
			name: "without base",
			flag: flag{
				ManagementCluster: "demomc",
				Name:              "demowc",
				Organization:      "demoorg",
				RepositoryName:    "gitops-demo",
			},
		},
		{
			name: "with base and release",
			flag: flag{
				Base:              "bases/clusters/capa/template",
				ManagementCluster: "demomc",
				Name:              "demowc",
				Organization:      "demoorg",
				RepositoryName:    "gitops-demo",
				Release:           "36.1.0",
			},
		},
		{
			name: "with base and without release",
			flag: flag{
				Base:              "bases/clusters/capa/template",
				ManagementCluster: "demomc",
				Name:              "demowc",
				Organization:      "demoorg",
				RepositoryName:    "gitops-demo",
			},
			expectInvalid: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.flag.Validate()
			if tc.expectInvalid {
				if microerror.Cause(err) != invalidFlagsError {
					t.Fatalf("expected invalid flags error, got: %v", err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}
		})
	}
}
