package creator

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"
)

const (
	dryRunRepoPath = "/tmp/testpath"
)

type testFsObjects struct {
	RelativePath string
	InputData    string
}

func Test_Create(t *testing.T) {
	testCases := []struct {
		name           string
		creator        Creator
		expectedCreate map[string]string
		expectedDryRun string
		testFsObjects  []testFsObjects
	}{
		{
			name: "flawless management cluster (dry-run)",
			creator: Creator{
				dryRun: true,
				path:   dryRunRepoPath,
			},
			expectedDryRun: "testdata/expected/dry-run/case-0-flawless.golden",
			testFsObjects: []testFsObjects{
				testFsObjects{RelativePath: "demomc", InputData: ""},
				testFsObjects{RelativePath: "demomc/demomc.yaml", InputData: "testdata/input/demomc.yaml"},
				testFsObjects{RelativePath: "demomc/.sops.keys", InputData: ""},
				testFsObjects{RelativePath: "demomc/secrets", InputData: ""},
				testFsObjects{RelativePath: "demomc/organizations", InputData: ""},
			},
		},
		{
			name: "flawless organization (dry-run)",
			creator: Creator{
				dryRun: true,
				path:   dryRunRepoPath,
			},
			expectedDryRun: "testdata/expected/dry-run/case-1-flawless.golden",
			testFsObjects: []testFsObjects{
				testFsObjects{RelativePath: "demoorg", InputData: ""},
				testFsObjects{RelativePath: "demoorg/demoorg.yaml", InputData: "testdata/input/demoorg.yaml"},
				testFsObjects{RelativePath: "demoorg/workload-clusters", InputData: ""},
				testFsObjects{RelativePath: "demoorg/workload-clusters/kustomization.yaml", InputData: "testdata/input/0-kustomization.yaml"},
			},
		},
		{
			name: "flawless workload cluster (dry-run)",
			creator: Creator{
				dryRun: true,
				path:   dryRunRepoPath,
			},
			expectedDryRun: "testdata/expected/dry-run/case-2-flawless.golden",
			testFsObjects: []testFsObjects{
				testFsObjects{RelativePath: "workload-clusters", InputData: ""},
				testFsObjects{RelativePath: "workload-clusters/demowc.yaml", InputData: "testdata/input/demowc.yaml"},
				testFsObjects{RelativePath: "workload-clusters/demowc", InputData: ""},
				testFsObjects{RelativePath: "workload-clusters/demowc/apps", InputData: ""},
				testFsObjects{RelativePath: "workload-clusters/demowc/cluster", InputData: ""},
				testFsObjects{RelativePath: "workload-clusters/demowc/cluster/kustomization.yaml", InputData: "testdata/input/1-kustomization.yaml"},
			},
		},
		{
			name: "flawless management cluster (create)",
			creator: Creator{
				dryRun: false,
				fs:     &afero.Afero{Fs: afero.NewOsFs()},
			},
			expectedCreate: map[string]string{
				"demomc":               "",
				"demomc/demomc.yaml":   "testdata/expected/create/0-demomc.golden",
				"demomc/.sops.keys":    "",
				"demomc/secrets":       "",
				"demomc/organizations": "",
			},
			testFsObjects: []testFsObjects{
				testFsObjects{RelativePath: "demomc", InputData: ""},
				testFsObjects{RelativePath: "demomc/demomc.yaml", InputData: "testdata/input/demomc.yaml"},
				testFsObjects{RelativePath: "demomc/.sops.keys", InputData: ""},
				testFsObjects{RelativePath: "demomc/secrets", InputData: ""},
				testFsObjects{RelativePath: "demomc/organizations", InputData: ""},
			},
		},
		{
			name: "flawless organization (create)",
			creator: Creator{
				dryRun: false,
				fs:     &afero.Afero{Fs: afero.NewOsFs()},
			},
			expectedCreate: map[string]string{
				"demoorg":                   "",
				"demoorg/demoorg.yaml":      "testdata/expected/create/1-demoorg.golden",
				"demoorg/workload-clusters": "",
				"demoorg/workload-clusters/kustomization.yaml": "testdata/expected/create/1-kustomization.golden",
			},
			testFsObjects: []testFsObjects{
				testFsObjects{RelativePath: "demoorg", InputData: ""},
				testFsObjects{RelativePath: "demoorg/demoorg.yaml", InputData: "testdata/input/demoorg.yaml"},
				testFsObjects{RelativePath: "demoorg/workload-clusters", InputData: ""},
				testFsObjects{RelativePath: "demoorg/workload-clusters/kustomization.yaml", InputData: "testdata/input/0-kustomization.yaml"},
			},
		},
		{
			name: "flawless workload cluster (create)",
			creator: Creator{
				dryRun: false,
				fs:     &afero.Afero{Fs: afero.NewOsFs()},
			},
			expectedCreate: map[string]string{
				"workload-clusters":                                   "",
				"workload-clusters/demowc.yaml":                       "testdata/expected/create/2-demowc.golden",
				"workload-clusters/demowc":                            "",
				"workload-clusters/demowc/apps":                       "",
				"workload-clusters/demowc/cluster":                    "",
				"workload-clusters/demowc/cluster/kustomization.yaml": "testdata/expected/create/2-kustomization.golden",
			},
			testFsObjects: []testFsObjects{
				testFsObjects{RelativePath: "workload-clusters", InputData: ""},
				testFsObjects{RelativePath: "workload-clusters/demowc.yaml", InputData: "testdata/input/demowc.yaml"},
				testFsObjects{RelativePath: "workload-clusters/demowc", InputData: ""},
				testFsObjects{RelativePath: "workload-clusters/demowc/apps", InputData: ""},
				testFsObjects{RelativePath: "workload-clusters/demowc/cluster", InputData: ""},
				testFsObjects{RelativePath: "workload-clusters/demowc/cluster/kustomization.yaml", InputData: "testdata/input/1-kustomization.yaml"},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("case %d: %s", i, tc.name), func(t *testing.T) {
			var err error

			var tmpDir string

			fsObjects := make([]*FsObject, 0)
			for _, tfo := range tc.testFsObjects {
				if tfo.InputData == "" {
					fsObjects = append(fsObjects, NewFsObject(tfo.RelativePath, nil, 0))
					continue
				}

				data, err := ioutil.ReadFile(tfo.InputData)
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}

				fsObjects = append(
					fsObjects,
					NewFsObject(tfo.RelativePath, data, 0),
				)
			}

			tc.creator.fsObjects = fsObjects

			if !tc.creator.dryRun {
				tmpDir, err = ioutil.TempDir("", "creator-test")

				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}

				err = os.Mkdir(tmpDir+"/management-clusters", 0755)
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}

				tc.creator.path = fmt.Sprintf("%s/%s", tmpDir, tc.creator.path)
				defer os.RemoveAll(tmpDir)
			}

			out := new(bytes.Buffer)
			tc.creator.stdout = out

			err = tc.creator.Create()
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			if tc.expectedDryRun != "" {
				expected, err := ioutil.ReadFile(tc.expectedDryRun)
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}

				e := strings.TrimRight(string(expected), "\n")
				g := strings.TrimRight(out.String(), "\n")

				if !bytes.Equal([]byte(g), []byte(e)) {
					t.Fatalf("want matching files \n%s\n", cmp.Diff(e, g))
				}
			}

			if tc.expectedCreate == nil {
				return
			}

			for p, c := range tc.expectedCreate {
				if _, err := os.Stat(fmt.Sprintf("%s/%s", tc.creator.path, p)); err != nil {
					t.Fatalf("expected ni, got error: %s", err.Error())
				}

				if c == "" {
					continue
				}

				got, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", tc.creator.path, p))
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}

				expected, err := ioutil.ReadFile(c)
				if err != nil {
					t.Fatalf("unexpected error: %s", err.Error())
				}

				if !bytes.Equal(got, expected) {
					t.Fatalf("want matching files \n%s\n", cmp.Diff(string(expected), string(got)))
				}
			}

		})
	}
}
