package org

import (
	"errors"
	"testing"
)

const (
	testMC         = "demomc"
	testOrg        = "demoorg"
	testRepository = "gitops-demo"
)

func Test_flag_Validate(t *testing.T) {
	testCases := []struct {
		name          string
		flag          flag
		expectInvalid bool
	}{
		{
			name: "complete",
			flag: flag{ManagementCluster: testMC, Name: testOrg, RepositoryName: testRepository},
		},
		{
			// The organisation's namespace Kustomization reads the repository.
			name:          "without repository",
			flag:          flag{ManagementCluster: testMC, Name: testOrg},
			expectInvalid: true,
		},
		{
			name:          "without management cluster",
			flag:          flag{Name: testOrg, RepositoryName: testRepository},
			expectInvalid: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.flag.Validate()
			if tc.expectInvalid {
				if !errors.Is(err, invalidFlagsError) {
					t.Fatalf("expected invalid flags error, got: %v", err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}
		})
	}
}
