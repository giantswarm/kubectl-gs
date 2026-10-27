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
			name: "with base, release and cluster app version",
			flag: flag{
				Base:              "bases/clusters/capa/template",
				ManagementCluster: "demomc",
				Name:              "demowc",
				Organization:      "demoorg",
				RepositoryName:    "gitops-demo",
				Release:           "36.1.0",
				ClusterAppVersion: "2.3.0",
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
				ClusterAppVersion: "2.3.0",
			},
			expectInvalid: true,
		},
		{
			name: "with base and without cluster app version",
			flag: flag{
				Base:              "bases/clusters/capa/template",
				ManagementCluster: "demomc",
				Name:              "demowc",
				Organization:      "demoorg",
				RepositoryName:    "gitops-demo",
				Release:           "36.1.0",
			},
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
