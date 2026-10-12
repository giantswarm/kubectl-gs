package cluster

import (
	"bytes"
	"context"
	goflag "flag"
	"fmt"
	"strings"
	"testing"

	applicationv1alpha1 "github.com/giantswarm/apiextensions-application/api/v1alpha1"
	"github.com/giantswarm/micrologger"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	//nolint:staticcheck
	"github.com/giantswarm/kubectl-gs/v6/cmd/template/cluster/common"
	"github.com/giantswarm/kubectl-gs/v6/cmd/template/cluster/flags"
	"github.com/giantswarm/kubectl-gs/v6/pkg/output"
	"github.com/giantswarm/kubectl-gs/v6/test/goldenfile"
	"github.com/giantswarm/kubectl-gs/v6/test/kubeclient"
)

var update = goflag.Bool("update", false, "update .golden reference test files")

// Test_run uses golden files.
//
// go test ./cmd/template/cluster -run Test_run -update
func Test_run(t *testing.T) {
	capaManagementCluster := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "infrastructure.cluster.x-k8s.io/v1beta2",
			"kind":       "AWSCluster",
			"metadata": map[string]interface{}{
				"name":      "my-mc",
				"namespace": "org-giantswarm",
			},
			"status": map[string]interface{}{
				"network": map[string]interface{}{
					"natGatewaysIPs": []interface{}{
						"1.2.3.4",
						"5.6.7.8",
						"9.10.11.12",
					},
				},
			},
		},
	}

	// The management cluster's control plane, whose API server trusts the
	// management cluster's Dex.
	mcControlPlane := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "controlplane.cluster.x-k8s.io/v1beta2",
			"kind":       "KubeadmControlPlane",
			"metadata": map[string]interface{}{
				"name":      "my-mc",
				"namespace": "org-giantswarm",
			},
			"spec": map[string]interface{}{
				"kubeadmConfigSpec": map[string]interface{}{
					"clusterConfiguration": map[string]interface{}{
						"apiServer": map[string]interface{}{
							"extraArgs": []interface{}{
								apiServerArg("oidc-client-id", "dex-k8s-authenticator"),
								apiServerArg("oidc-groups-claim", "groups"),
								apiServerArg("oidc-issuer-url", "https://dex.my-mc.example.com"),
								apiServerArg("oidc-username-claim", "email"),
							},
						},
					},
				},
			},
		},
	}

	// Catalog entries for the release-<provider> charts which are published
	// for release v35.0.0. Release versions without such an entry, e.g. a
	// hand-crafted Release CR used for testing, must fall back to the
	// cluster-<provider> chart.
	releaseCatalogEntries := []runtime.Object{
		newAppCatalogEntry("cluster", "release-aws", "35.0.0"),
		newAppCatalogEntry("cluster", "release-aks", "35.0.0"),
	}

	testCases := []struct {
		name               string
		flags              *flags.Flag
		args               []string
		clusterName        string
		expectedGoldenFile string
		errorMatcher       func(error) bool
		// noMCControlPlane leaves the management cluster's control plane
		// out of the fake management cluster.
		noMCControlPlane bool
	}{
		{
			name: "case 1: template cluster capa",
			flags: &flags.Flag{
				Name:                     "test1",
				Provider:                 "capa",
				Description:              "just a test cluster",
				Release:                  "25.0.0",
				Region:                   "the-region",
				Organization:             "test",
				ControlPlaneInstanceType: "control-plane-instance-type",
				App: common.AppConfig{
					ClusterVersion: "1.0.0",
					ClusterCatalog: "the-catalog",
				},
				AWS: common.AWSConfig{
					MachinePool: common.AWSMachinePoolConfig{
						Name:             "worker1",
						AZs:              []string{"eu-west-1a", "eu-west-1b"},
						InstanceType:     "big-one",
						MaxSize:          5,
						MinSize:          2,
						RootVolumeSizeGB: 200,
						CustomNodeLabels: []string{"label=value"},
					},
					AWSClusterRoleIdentityName: "default",
					NetworkVPCCIDR:             "10.123.0.0/16",
					PublicSubnetMask:           20,
					PrivateSubnetMask:          18,
					NetworkAZUsageLimit:        3,
				},
			},
			args:               nil,
			expectedGoldenFile: "run_template_cluster_capa.golden",
		},
		{
			name: "case 2: template proxy-private cluster capa with defaults",
			flags: &flags.Flag{
				Name:                     "test1",
				Provider:                 "capa",
				ManagementCluster:        "my-mc",
				Description:              "just a test cluster",
				Release:                  "25.0.0",
				Region:                   "the-region",
				Organization:             "test",
				ControlPlaneInstanceType: "control-plane-instance-type",
				App: common.AppConfig{
					ClusterVersion: "1.0.0",
					ClusterCatalog: "the-catalog",
				},
				AWS: common.AWSConfig{
					ClusterType: "proxy-private",
					MachinePool: common.AWSMachinePoolConfig{
						Name:             "worker1",
						AZs:              []string{"eu-west-1a", "eu-west-1b"},
						InstanceType:     "big-one",
						MaxSize:          5,
						MinSize:          2,
						RootVolumeSizeGB: 200,
						CustomNodeLabels: []string{"label=value"},
					},
					AWSClusterRoleIdentityName: "default",
					NetworkVPCCIDR:             "10.123.0.0/16",
					PrivateSubnetMask:          18,
					NetworkAZUsageLimit:        2,
					HttpsProxy:                 "https://internal-a1c90e5331e124481a14fb7ad80ae8eb-1778512673.eu-west-2.elb.amazonaws.com:4000",
					HttpProxy:                  "http://internal-a1c90e5331e124481a14fb7ad80ae8eb-1778512673.eu-west-2.elb.amazonaws.com:4000",
					NoProxy:                    "test-domain.com",
					ControlPlaneLoadBalancerIngressAllowCIDRBlocks: []string{""},
				},
			},
			args:               nil,
			expectedGoldenFile: "run_template_cluster_capa_2.golden",
		},
		{
			name: "case 3: template proxy-private cluster capa",
			flags: &flags.Flag{
				Name:                     "test1",
				Provider:                 "capa",
				ManagementCluster:        "my-mc",
				Description:              "just a test cluster",
				Release:                  "25.0.0",
				Region:                   "the-region",
				Organization:             "test",
				ControlPlaneInstanceType: "control-plane-instance-type",
				App: common.AppConfig{
					ClusterVersion: "1.0.0",
					ClusterCatalog: "the-catalog",
				},
				AWS: common.AWSConfig{
					ClusterType: "proxy-private",
					MachinePool: common.AWSMachinePoolConfig{
						Name:             "worker1",
						AZs:              []string{"eu-west-1a", "eu-west-1b"},
						InstanceType:     "big-one",
						MaxSize:          5,
						MinSize:          2,
						RootVolumeSizeGB: 200,
						CustomNodeLabels: []string{"label=value"},
					},
					AWSClusterRoleIdentityName: "other-identity",
					NetworkVPCCIDR:             "10.123.0.0/16",
					PrivateSubnetMask:          18,
					NetworkAZUsageLimit:        2,
					APIMode:                    "public",
					TopologyMode:               "UserManaged",
					PrefixListID:               "pl-123456789abc",
					TransitGatewayID:           "tgw-987987987987def",
					HttpsProxy:                 "https://internal-a1c90e5331e124481a14fb7ad80ae8eb-1778512673.eu-west-2.elb.amazonaws.com:4000",
					HttpProxy:                  "http://internal-a1c90e5331e124481a14fb7ad80ae8eb-1778512673.eu-west-2.elb.amazonaws.com:4000",
					NoProxy:                    "test-domain.com",
					ControlPlaneLoadBalancerIngressAllowCIDRBlocks: []string{"7.7.7.7/32"},
				},
			},
			args:               nil,
			expectedGoldenFile: "run_template_cluster_capa_3.golden",
		},
		{
			name: "case 4: template cluster capz",
			flags: &flags.Flag{
				Name:                     "test1",
				Provider:                 "capz",
				Description:              "just a test cluster",
				Region:                   "northeurope",
				Release:                  "25.0.0",
				Organization:             "test",
				ControlPlaneInstanceType: "B2s",
				App: common.AppConfig{
					ClusterVersion: "0.17.0",
					ClusterCatalog: "the-catalog",
				},
				Azure: common.AzureConfig{
					SubscriptionID: "12345678-ebb8-4b1f-8f96-d950d9e7aaaa",
				},
			},
			args:               nil,
			expectedGoldenFile: "run_template_cluster_capz.golden",
		},
		{
			name: "case 5: template cluster capv (cluster-vsphere)",
			flags: &flags.Flag{
				Name:              "test1",
				Provider:          "vsphere",
				Description:       "yet another test cluster",
				Release:           "27.0.0",
				Organization:      "test",
				KubernetesVersion: "v1.2.3",
				App: common.AppConfig{
					ClusterVersion: "0.59.0",
					ClusterCatalog: "foo-catalog",
				},
				VSphere: common.VSphereConfig{
					ServiceLoadBalancerCIDR: "1.2.3.4/32",
					ResourcePool:            "foopool",
					NetworkName:             "foonet",
					SvcLbIpPoolName:         "svc-foo-pool",
					CredentialsSecretName:   "foosecret",
					ControlPlane: common.VSphereControlPlane{
						VSphereMachineTemplate: common.VSphereMachineTemplate{
							DiskGiB:   42,
							MemoryMiB: 42000,
							NumCPUs:   6,
							Replicas:  5,
						},
						IpPoolName: "foo-pool",
					},
					Worker: common.VSphereMachineTemplate{
						DiskGiB:   43,
						MemoryMiB: 43000,
						NumCPUs:   7,
						Replicas:  4,
					},
				},
			},
			args:               nil,
			expectedGoldenFile: "run_template_cluster_capv_1.golden",
		},
		{
			name: "case 6: template cluster capv (unified cluster-vsphere)",
			flags: &flags.Flag{
				Name:              "test1",
				Provider:          "vsphere",
				Description:       "yet another test cluster",
				Release:           "27.0.0",
				Organization:      "test",
				KubernetesVersion: "v1.2.3",
				App: common.AppConfig{
					ClusterVersion: "1.2.3",
					ClusterCatalog: "foo-catalog",
				},
				VSphere: common.VSphereConfig{
					ServiceLoadBalancerCIDR: "1.2.3.4/32",
					ResourcePool:            "foopool",
					NetworkName:             "foonet",
					SvcLbIpPoolName:         "svc-foo-pool",
					CredentialsSecretName:   "foosecret",
					ControlPlane: common.VSphereControlPlane{
						VSphereMachineTemplate: common.VSphereMachineTemplate{
							DiskGiB:   42,
							MemoryMiB: 42000,
							NumCPUs:   6,
							Replicas:  5,
						},
						IpPoolName: "foo-pool",
					},
					Worker: common.VSphereMachineTemplate{
						DiskGiB:   43,
						MemoryMiB: 43000,
						NumCPUs:   7,
						Replicas:  4,
					},
				},
			},
			args:               nil,
			expectedGoldenFile: "run_template_cluster_capv_2.golden",
		},
		{
			name: "case 7: template cluster capa with custom network CIDR",
			flags: &flags.Flag{
				Name:                     "test6",
				Provider:                 "capa",
				Description:              "just a test cluster",
				Release:                  "25.0.0",
				Region:                   "the-region",
				Organization:             "test",
				ControlPlaneInstanceType: "control-plane-instance-type",
				App: common.AppConfig{
					ClusterVersion: "1.0.0",
					ClusterCatalog: "the-catalog",
				},
				AWS: common.AWSConfig{
					MachinePool: common.AWSMachinePoolConfig{
						Name:             "worker1",
						AZs:              []string{"eu-west-1a", "eu-west-1b"},
						InstanceType:     "big-one",
						MaxSize:          5,
						MinSize:          2,
						RootVolumeSizeGB: 200,
						CustomNodeLabels: []string{"label=value"},
					},
					AWSClusterRoleIdentityName: "default",
					NetworkVPCCIDR:             "192.168.0.0/16",
					PublicSubnetMask:           20,
					PrivateSubnetMask:          18,
					NetworkAZUsageLimit:        2,
				},
			},
			args:               nil,
			expectedGoldenFile: "run_template_cluster_capa_6.golden",
		},
		{
			name: "case 8: template cluster capa with custom network CIDR 3 AZ",
			flags: &flags.Flag{
				Name:                     "test7",
				Provider:                 "capa",
				Description:              "just a test cluster",
				Release:                  "25.0.0",
				Region:                   "the-region",
				Organization:             "test",
				ControlPlaneInstanceType: "control-plane-instance-type",
				App: common.AppConfig{
					ClusterVersion: "1.0.0",
					ClusterCatalog: "the-catalog",
				},
				AWS: common.AWSConfig{
					MachinePool: common.AWSMachinePoolConfig{
						Name:             "worker1",
						AZs:              []string{"eu-west-1a", "eu-west-1b"},
						InstanceType:     "big-one",
						MaxSize:          5,
						MinSize:          2,
						RootVolumeSizeGB: 200,
						CustomNodeLabels: []string{"label=value"},
					},
					AWSClusterRoleIdentityName: "default",
					NetworkVPCCIDR:             "192.168.0.0/18",
					PublicSubnetMask:           22,
					PrivateSubnetMask:          20,
					NetworkAZUsageLimit:        3,
				},
			},
			args:               nil,
			expectedGoldenFile: "run_template_cluster_capa_7.golden",
		},
		{
			name: "case 9: template cluster capa with custom network CIDR 1 AZ",
			flags: &flags.Flag{
				Name:                     "test8",
				Provider:                 "capa",
				Description:              "just a test cluster",
				Release:                  "25.0.0",
				Region:                   "the-region",
				Organization:             "test",
				ControlPlaneInstanceType: "control-plane-instance-type",
				App: common.AppConfig{
					ClusterVersion: "1.0.0",
					ClusterCatalog: "the-catalog",
				},
				AWS: common.AWSConfig{
					MachinePool: common.AWSMachinePoolConfig{
						Name:             "worker1",
						AZs:              []string{"eu-west-1a", "eu-west-1b"},
						InstanceType:     "big-one",
						MaxSize:          5,
						MinSize:          2,
						RootVolumeSizeGB: 200,
						CustomNodeLabels: []string{"label=value"},
					},
					AWSClusterRoleIdentityName: "default",
					NetworkVPCCIDR:             "10.10.0.0/12",
					PublicSubnetMask:           21,
					PrivateSubnetMask:          16,
					NetworkAZUsageLimit:        1,
				},
			},
			args:               nil,
			expectedGoldenFile: "run_template_cluster_capa_8.golden",
		},
		{
			name: "case 11: template cluster aks",
			flags: &flags.Flag{
				Name:              "test-aks",
				Provider:          "aks",
				Description:       "an AKS test cluster",
				Release:           "35.0.0",
				Region:            "westeurope",
				Organization:      "test",
				ManagementCluster: "my-mc",
				App: common.AppConfig{
					ClusterCatalog: "cluster",
				},
				Azure: common.AzureConfig{
					SubscriptionID:           "6b1f6e4a-6d0e-4aa4-9a5a-fbaca65a23b3",
					ClusterIdentityName:      "my-aks-identity",
					ClusterIdentityNamespace: "org-test",
				},
			},
			args:               nil,
			expectedGoldenFile: "run_template_cluster_aks.golden",
			// AKS renders no OIDC settings, so it never asks for them.
			noMCControlPlane: true,
		},
		{
			name: "case 10: template cluster capa with release version",
			flags: &flags.Flag{
				Name:                     "test10",
				Provider:                 "capa",
				Description:              "cluster using release version directly",
				Release:                  "35.0.0",
				Region:                   "the-region",
				Organization:             "test",
				ControlPlaneInstanceType: "control-plane-instance-type",
				App: common.AppConfig{
					ClusterCatalog: "cluster",
				},
				AWS: common.AWSConfig{
					MachinePool: common.AWSMachinePoolConfig{
						Name:             "worker1",
						AZs:              []string{"eu-west-1a", "eu-west-1b"},
						InstanceType:     "big-one",
						MaxSize:          5,
						MinSize:          2,
						RootVolumeSizeGB: 200,
						CustomNodeLabels: []string{"label=value"},
					},
					AWSClusterRoleIdentityName: "default",
					NetworkVPCCIDR:             "10.123.0.0/16",
					PublicSubnetMask:           20,
					PrivateSubnetMask:          18,
					NetworkAZUsageLimit:        3,
				},
			},
			args:               nil,
			expectedGoldenFile: "run_template_cluster_capa_9.golden",
		},
		{
			name: "case 13: template cluster capa with a release version which has no release chart",
			flags: &flags.Flag{
				Name:                     "test13",
				Provider:                 "capa",
				Description:              "cluster using a hand-crafted release for testing",
				Release:                  "35.0.0-andreas",
				Region:                   "the-region",
				Organization:             "test",
				ControlPlaneInstanceType: "control-plane-instance-type",
				App: common.AppConfig{
					ClusterCatalog: "cluster",
				},
				AWS: common.AWSConfig{
					MachinePool: common.AWSMachinePoolConfig{
						Name:             "worker1",
						AZs:              []string{"eu-west-1a", "eu-west-1b"},
						InstanceType:     "big-one",
						MaxSize:          5,
						MinSize:          2,
						RootVolumeSizeGB: 200,
						CustomNodeLabels: []string{"label=value"},
					},
					AWSClusterRoleIdentityName: "default",
					NetworkVPCCIDR:             "10.123.0.0/16",
					PublicSubnetMask:           20,
					PrivateSubnetMask:          18,
					NetworkAZUsageLimit:        3,
				},
			},
			args:               nil,
			expectedGoldenFile: "run_template_cluster_capa_11.golden",
		},
		{
			name: "case 14: template cluster capa with release version and explicit cluster chart version",
			flags: &flags.Flag{
				Name:                     "test14",
				Provider:                 "capa",
				Description:              "cluster testing a cluster-aws dev build",
				Release:                  "35.0.0",
				Region:                   "the-region",
				Organization:             "test",
				ControlPlaneInstanceType: "control-plane-instance-type",
				App: common.AppConfig{
					ClusterCatalog: "cluster",
					ClusterVersion: "9.0.1-dev.private-karpenter.2026-08-20.16-33-07.h62652f9",
				},
				AWS: common.AWSConfig{
					MachinePool: common.AWSMachinePoolConfig{
						Name:             "worker1",
						AZs:              []string{"eu-west-1a", "eu-west-1b"},
						InstanceType:     "big-one",
						MaxSize:          5,
						MinSize:          2,
						RootVolumeSizeGB: 200,
						CustomNodeLabels: []string{"label=value"},
					},
					AWSClusterRoleIdentityName: "default",
					NetworkVPCCIDR:             "10.123.0.0/16",
					PublicSubnetMask:           20,
					PrivateSubnetMask:          18,
					NetworkAZUsageLimit:        3,
				},
			},
			args:               nil,
			expectedGoldenFile: "run_template_cluster_capa_12.golden",
		},
		{
			name: "case 11: template cluster capa with small VPC for single AZ",
			flags: &flags.Flag{
				Name:                     "test11",
				Provider:                 "capa",
				Description:              "small single-AZ VPC layout",
				Release:                  "25.0.0",
				Region:                   "the-region",
				Organization:             "test",
				ControlPlaneInstanceType: "control-plane-instance-type",
				App: common.AppConfig{
					ClusterVersion: "1.0.0",
					ClusterCatalog: "the-catalog",
				},
				AWS: common.AWSConfig{
					MachinePool: common.AWSMachinePoolConfig{
						Name:             "worker1",
						AZs:              []string{"eu-west-1a"},
						InstanceType:     "big-one",
						MaxSize:          5,
						MinSize:          2,
						RootVolumeSizeGB: 200,
						CustomNodeLabels: []string{"label=value"},
					},
					AWSClusterRoleIdentityName: "default",
					NetworkVPCCIDR:             "10.85.0.0/24",
					PublicSubnetMask:           26,
					PrivateSubnetMask:          25,
					NetworkAZUsageLimit:        1,
				},
			},
			args:               nil,
			expectedGoldenFile: "run_template_cluster_capa_10.golden",
		},
		{
			name: "case 12: template cluster capa with VPC too small for requested AZs",
			flags: &flags.Flag{
				Name:                     "test12",
				Provider:                 "capa",
				Description:              "VPC too small for 3 AZs with default subnet sizes",
				Release:                  "25.0.0",
				Region:                   "the-region",
				Organization:             "test",
				ControlPlaneInstanceType: "control-plane-instance-type",
				App: common.AppConfig{
					ClusterVersion: "1.0.0",
					ClusterCatalog: "the-catalog",
				},
				AWS: common.AWSConfig{
					MachinePool: common.AWSMachinePoolConfig{
						Name:             "worker1",
						AZs:              []string{"eu-west-1a", "eu-west-1b", "eu-west-1c"},
						InstanceType:     "big-one",
						MaxSize:          5,
						MinSize:          2,
						RootVolumeSizeGB: 200,
						CustomNodeLabels: []string{"label=value"},
					},
					AWSClusterRoleIdentityName: "default",
					NetworkVPCCIDR:             "10.85.0.0/24",
					PublicSubnetMask:           20,
					PrivateSubnetMask:          18,
					NetworkAZUsageLimit:        3,
				},
			},
			args: nil,
			errorMatcher: func(err error) bool {
				return err != nil && strings.Contains(err.Error(), "too small to host 3 availability zones")
			},
		},
		{
			name: "case 13: template cluster capa with a non-default service priority",
			flags: &flags.Flag{
				Name:                     "test13",
				Provider:                 "capa",
				Description:              "just a test cluster",
				Release:                  "25.0.0",
				Region:                   "the-region",
				Organization:             "test",
				ControlPlaneInstanceType: "control-plane-instance-type",
				ServicePriority:          "medium",
				App: common.AppConfig{
					ClusterVersion: "1.0.0",
					ClusterCatalog: "the-catalog",
				},
				AWS: common.AWSConfig{
					MachinePool: common.AWSMachinePoolConfig{
						Name:             "worker1",
						AZs:              []string{"eu-west-1a", "eu-west-1b"},
						InstanceType:     "big-one",
						MaxSize:          5,
						MinSize:          2,
						RootVolumeSizeGB: 200,
						CustomNodeLabels: []string{"label=value"},
					},
					AWSClusterRoleIdentityName: "default",
					NetworkVPCCIDR:             "10.123.0.0/16",
					PublicSubnetMask:           20,
					PrivateSubnetMask:          18,
					NetworkAZUsageLimit:        3,
				},
			},
			args:               nil,
			expectedGoldenFile: "run_template_cluster_capa_service_priority.golden",
		},
		{
			name:               "case 14: template cluster capa with OIDC disabled",
			flags:              oidcTestFlags(func(f *flags.Flag) { f.DisableOIDC = true }),
			expectedGoldenFile: "run_template_cluster_capa_oidc_disabled.golden",
			noMCControlPlane:   true,
		},
		{
			name: "case 15: template cluster capa with an explicit OIDC issuer",
			flags: oidcTestFlags(func(f *flags.Flag) {
				f.OIDC = common.OIDC{IssuerURL: testIssuerURL, ClientID: testClientID}
			}),
			expectedGoldenFile: "run_template_cluster_capa_oidc_explicit.golden",
			noMCControlPlane:   true,
		},
		{
			name: "case 16: template cluster capa with the management cluster's OIDC issuer and an explicit groups claim",
			flags: oidcTestFlags(func(f *flags.Flag) {
				f.OIDC = common.OIDC{GroupsClaim: "roles"}
			}),
			expectedGoldenFile: "run_template_cluster_capa_oidc_groups_claim.golden",
		},
		{
			name:             "case 17: template cluster capa without a management cluster control plane",
			flags:            oidcTestFlags(nil),
			noMCControlPlane: true,
			errorMatcher: func(err error) bool {
				return common.IsManagementClusterOIDC(err) &&
					strings.Contains(err.Error(), "no KubeadmControlPlane found in namespace org-giantswarm") &&
					strings.Contains(err.Error(), "--disable-oidc")
			},
		},
		{
			name: "case 18: template cluster cloud-director",
			flags: &flags.Flag{
				Name:         "test-vcd",
				Provider:     "cloud-director",
				Description:  "a Cloud Director test cluster",
				Release:      "25.0.0",
				Organization: "test",
				App: common.AppConfig{
					ClusterCatalog: "cluster",
					ClusterVersion: "1.0.0",
				},
				CloudDirector: common.CloudDirectorConfig{
					VipSubnet:             "10.205.9.254/24",
					CredentialsSecretName: "vcd-credentials",
					Org:                   "my-org",
					Ovdc:                  "my-ovdc",
					OvdcNetwork:           "my-network",
					Site:                  "https://vcd.example.com",
					ControlPlane: common.CloudDirectorControlPlane{
						Replicas: 3,
						MachineTemplate: common.CloudDirectorMachineTemplate{
							DiskSizeGB:   30,
							SizingPolicy: "m1.medium",
						},
					},
					Worker: common.CloudDirectorMachineTemplate{
						DiskSizeGB:   30,
						SizingPolicy: "m1.medium",
						Replicas:     3,
					},
				},
			},
			expectedGoldenFile: "run_template_cluster_capvcd.golden",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			out := new(bytes.Buffer)
			tc.flags.Print = genericclioptions.NewPrintFlags("").WithDefaultOutput(output.TypeDefault)

			logger, err := micrologger.New(micrologger.Config{})
			if err != nil {
				t.Fatalf("failed to create logger: %s", err.Error())
			}

			runner := &runner{
				flag:   tc.flags,
				logger: logger,
				stdout: out,
			}

			k8sClient := kubeclient.FakeK8sClient(releaseCatalogEntries...)
			if tc.flags.Provider == "capa" {
				err = k8sClient.CtrlClient().Create(ctx, capaManagementCluster.DeepCopy())
				if err != nil {
					t.Fatalf("failed to fake AWSCluster object: %s", err.Error())
				}
			}
			if !tc.noMCControlPlane {
				err = k8sClient.CtrlClient().Create(ctx, mcControlPlane.DeepCopy())
				if err != nil {
					t.Fatalf("failed to fake KubeadmControlPlane object: %s", err.Error())
				}
			}
			err = runner.run(ctx, k8sClient)
			if tc.errorMatcher != nil {
				if !tc.errorMatcher(err) {
					t.Fatalf("error not matching expected matcher, got: %s", errors.Cause(err))
				}

				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err.Error())
			}

			var expectedResult []byte
			{
				gf := goldenfile.New("testdata", tc.expectedGoldenFile)
				if *update {
					err = gf.Update(out.Bytes())
					if err != nil {
						t.Fatalf("unexpected error: %s", err.Error())
					}
					expectedResult = out.Bytes()
				} else {
					expectedResult, err = gf.Read()
					if err != nil {
						t.Fatalf("unexpected error: %s", err.Error())
					}
				}
			}

			diff := cmp.Diff(string(expectedResult), out.String())
			if diff != "" {
				t.Fatalf("no difference from golden file %s expected, got:\n %s", tc.expectedGoldenFile, diff)
			}
		})
	}
}

// apiServerArg is one API server extra argument of a KubeadmControlPlane.
func apiServerArg(name, value string) interface{} {
	return map[string]interface{}{"name": name, "value": value}
}

// oidcTestFlags returns the flags of a plain CAPA cluster, changed by modify.
func oidcTestFlags(modify func(*flags.Flag)) *flags.Flag {
	f := &flags.Flag{
		Name:                     "test-oidc",
		Provider:                 "capa",
		Description:              "OIDC test cluster",
		Release:                  "25.0.0",
		Region:                   "the-region",
		Organization:             "test",
		ControlPlaneInstanceType: "control-plane-instance-type",
		App: common.AppConfig{
			ClusterCatalog: "cluster",
		},
		AWS: common.AWSConfig{
			MachinePool: common.AWSMachinePoolConfig{
				Name:         "worker1",
				AZs:          []string{"eu-west-1a"},
				InstanceType: "big-one",
				MaxSize:      5,
				MinSize:      2,
			},
			AWSClusterRoleIdentityName: "default",
			NetworkAZUsageLimit:        3,
		},
	}
	if modify != nil {
		modify(f)
	}
	return f
}

func newAppCatalogEntry(catalog, app, version string) *applicationv1alpha1.AppCatalogEntry {
	return &applicationv1alpha1.AppCatalogEntry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s-%s", catalog, app, version),
			Namespace: "giantswarm",
			Labels: map[string]string{
				"app.kubernetes.io/name":            app,
				"application.giantswarm.io/catalog": catalog,
			},
		},
		Spec: applicationv1alpha1.AppCatalogEntrySpec{
			AppName: app,
			Version: version,
		},
	}
}

func Test_useReleaseChart(t *testing.T) {
	testCases := []struct {
		name           string
		provider       string
		release        string
		clusterVersion string
		catalogEntries []runtime.Object
		expected       bool
		expectedWarn   string
	}{
		{
			name:           "case 0: legacy cluster chart version",
			provider:       "capa",
			release:        "25.0.0",
			catalogEntries: []runtime.Object{newAppCatalogEntry("cluster", "release-aws", "35.0.0")},
			expected:       false,
		},
		{
			name:           "case 1: published release version",
			provider:       "capa",
			release:        "35.0.0",
			catalogEntries: []runtime.Object{newAppCatalogEntry("cluster", "release-aws", "35.0.0")},
			expected:       true,
		},
		{
			name:           "case 2: release version without release chart falls back",
			provider:       "capa",
			release:        "35.0.0-andreas",
			catalogEntries: []runtime.Object{newAppCatalogEntry("cluster", "release-aws", "35.0.0")},
			expected:       false,
			expectedWarn:   "no release-aws chart with version 35.0.0-andreas found",
		},
		{
			name:           "case 3: explicit cluster chart version wins",
			provider:       "capa",
			release:        "35.0.0",
			clusterVersion: "9.0.1-dev.private-karpenter.2026-08-20.16-33-07.h62652f9",
			catalogEntries: []runtime.Object{newAppCatalogEntry("cluster", "release-aws", "35.0.0")},
			expected:       false,
		},
		{
			name:           "case 4: release chart of another provider does not count",
			provider:       "capz",
			release:        "35.0.0",
			catalogEntries: []runtime.Object{newAppCatalogEntry("cluster", "release-aws", "35.0.0")},
			expected:       false,
			expectedWarn:   "no release-azure chart with version 35.0.0 found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			stderr := new(bytes.Buffer)

			r := &runner{
				flag:   &flags.Flag{Provider: tc.provider},
				stderr: stderr,
			}
			config := common.ClusterConfig{
				ReleaseVersion: tc.release,
				App: common.AppConfig{
					ClusterCatalog: "cluster",
					ClusterVersion: tc.clusterVersion,
				},
			}

			result := r.useReleaseChart(ctx, kubeclient.FakeK8sClient(tc.catalogEntries...), config)
			if result != tc.expected {
				t.Fatalf("useReleaseChart() = %v, want %v", result, tc.expected)
			}

			if tc.expectedWarn == "" && stderr.Len() > 0 {
				t.Fatalf("unexpected warning: %s", stderr.String())
			}
			if tc.expectedWarn != "" && !strings.Contains(stderr.String(), tc.expectedWarn) {
				t.Fatalf("expected warning containing %q, got %q", tc.expectedWarn, stderr.String())
			}
		})
	}
}
