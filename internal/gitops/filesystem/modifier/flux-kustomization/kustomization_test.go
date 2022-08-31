package fluxkus

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_KustomizationModifier(t *testing.T) {
	testCases := []struct {
		name     string
		expected []byte
		input    []byte
		modifier KustomizationModifier
	}{
		{
			name: "add decryption",
			expected: []byte(`apiVersion: kustomize.toolkit.fluxcd.io/v1beta2
kind: Kustomization
metadata:
  name: demomc-gitops
  namespace: default
spec:
  decryption:
    provider: sops
    secretRef:
      name: sops-gpg-master
  interval: 1m0s
  path: ./management-clusters/demomc
  prune: true
  serviceAccountName: automation
  sourceRef:
    kind: GitRepository
    name: gitops-demo
  timeout: 2m0s
`),
			input: []byte(`apiVersion: kustomize.toolkit.fluxcd.io/v1beta2
kind: Kustomization
metadata:
  name: demomc-gitops
  namespace: default
spec:
  interval: 1m0s
  path: ./management-clusters/demomc
  prune: true
  serviceAccountName: automation
  sourceRef:
    kind: GitRepository
    name: gitops-demo
  timeout: 2m0s
`),
			modifier: KustomizationModifier{
				DecryptionToAdd: "sops-gpg-master",
			},
		},
		{
			name: "don't add if exists decryption",
			expected: []byte(`apiVersion: kustomize.toolkit.fluxcd.io/v1beta2
kind: Kustomization
metadata:
  name: demomc-gitops
  namespace: default
spec:
  decryption:
    provider: sops
    secretRef:
      name: sops-gpg-master
  interval: 1m0s
  path: ./management-clusters/demomc
  prune: true
  serviceAccountName: automation
  sourceRef:
    kind: GitRepository
    name: gitops-demo
  timeout: 2m0s
`),
			input: []byte(`apiVersion: kustomize.toolkit.fluxcd.io/v1beta2
kind: Kustomization
metadata:
  name: demomc-gitops
  namespace: default
spec:
  decryption:
    provider: sops
    secretRef:
      name: sops-gpg-master
  interval: 1m0s
  path: ./management-clusters/demomc
  prune: true
  serviceAccountName: automation
  sourceRef:
    kind: GitRepository
    name: gitops-demo
  timeout: 2m0s
`),
			modifier: KustomizationModifier{
				DecryptionToAdd: "sops-gpg-master",
			},
		},
		{
			name: "change back to the right one",
			expected: []byte(`apiVersion: kustomize.toolkit.fluxcd.io/v1beta2
kind: Kustomization
metadata:
  name: demomc-gitops
  namespace: default
spec:
  decryption:
    provider: sops
    secretRef:
      name: sops-gpg-master
  interval: 1m0s
  path: ./management-clusters/demomc
  prune: true
  serviceAccountName: automation
  sourceRef:
    kind: GitRepository
    name: gitops-demo
  timeout: 2m0s
`),
			input: []byte(`apiVersion: kustomize.toolkit.fluxcd.io/v1beta2
kind: Kustomization
metadata:
  name: demomc-gitops
  namespace: default
spec:
  decryption:
    provider: sops
    secretRef:
      name: sops-gpg-other
  interval: 1m0s
  path: ./management-clusters/demomc
  prune: true
  serviceAccountName: automation
  sourceRef:
    kind: GitRepository
    name: gitops-demo
  timeout: 2m0s
`),
			modifier: KustomizationModifier{
				DecryptionToAdd: "sops-gpg-master",
			},
		},
		{
			name: "adding postBuild variables",
			expected: []byte(`apiVersion: kustomize.toolkit.fluxcd.io/v1beta2
kind: Kustomization
metadata:
  name: demomc-gitops
  namespace: default
spec:
  interval: 1m0s
  path: ./management-clusters/demomc
  postBuild:
    substitute:
      cluster_release: 1.0.0
      default_apps_release: 1.0.0
  prune: true
  serviceAccountName: automation
  sourceRef:
    kind: GitRepository
    name: gitops-demo
  timeout: 2m0s
`),
			input: []byte(`apiVersion: kustomize.toolkit.fluxcd.io/v1beta2
kind: Kustomization
metadata:
  name: demomc-gitops
  namespace: default
spec:
  interval: 1m0s
  path: ./management-clusters/demomc
  prune: true
  serviceAccountName: automation
  sourceRef:
    kind: GitRepository
    name: gitops-demo
  timeout: 2m0s
`),
			modifier: KustomizationModifier{
				PostBuildEnvs: map[string]string{
					"cluster_release":      "1.0.0",
					"default_apps_release": "1.0.0",
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("case %d: %s", i, tc.name), func(t *testing.T) {
			got, err := tc.modifier.Execute(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			if !bytes.Equal(got, tc.expected) {
				t.Fatalf("want matching files \n%s\n", cmp.Diff(string(tc.expected), string(got)))
			}
		})
	}
}
