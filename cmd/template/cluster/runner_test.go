package cluster

import (
	"bytes"
	"context"
	goflag "flag"
	"strings"
	"testing"

	"github.com/giantswarm/micrologger"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	//nolint:staticcheck
	"github.com/giantswarm/kubectl-gs/v6/cmd/template/cluster/common"
	"github.com/giantswarm/kubectl-gs/v6/cmd/template/cluster/flags"
	"github.com/giantswarm/kubectl-gs/v6/pkg/output"
	"github.com/giantswarm/kubectl-gs/v6/test/goldenfile"
	"github.com/giantswarm/kubectl-gs/v6/test/kubeclient"
)

// fakeOCIClient is a minimal ociregistry.Client fake for testing the
// release-<provider> chart availability check without hitting a real
// registry. tags holds "repository:tag" keys that exist.
type fakeOCIClient struct {
	tags map[string]bool
	err  error
}

func (f *fakeOCIClient) ListTags(_ context.Context, _, _ string) ([]string, error) { return nil, nil }

func (f *fakeOCIClient) TagExists(_ context.Context, _, repository, tag string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.tags[repository+":"+tag], nil
}

func (f *fakeOCIClient) GetManifestAnnotations(_ context.Context, _, _, _ string) (map[string]string, error) {
	return nil, nil
}

func (f *fakeOCIClient) Close(_ context.Context) {}

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

	// The release-<provider> chart tags published in gsoci for release v35.0.0.
	// Release versions without such a tag, e.g. a hand-crafted Release CR used
	// for testing, must fall back to the cluster-<provider> chart. This
	// mirrors the real gsoci registry state: no release-eks chart is
	// published yet, so EKS keeps falling back.
	releaseChartOCIClient := &fakeOCIClient{
		tags: map[string]bool{
			"charts/giantswarm/release-aws:35.0.0":            true,
			"charts/giantswarm/release-aks:35.0.0":            true,
			"charts/giantswarm/release-azure:35.0.0":          true,
			"charts/giantswarm/release-vsphere:35.0.0":        true,
			"charts/giantswarm/release-cloud-director:35.0.0": true,
		},
	}

	testCases := []struct {
		name               string
		flags              *flags.Flag
		args               []string
		clusterName        string
		expectedGoldenFile string
		errorMatcher       func(error) bool
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
			name: "case 15: template cluster capz with release version",
			flags: &flags.Flag{
				Name:                     "test-capz-flux",
				Provider:                 "capz",
				Description:              "cluster using release version directly",
				Region:                   "northeurope",
				Release:                  "35.0.0",
				Organization:             "test",
				ControlPlaneInstanceType: "B2s",
				App: common.AppConfig{
					ClusterCatalog: "cluster",
				},
				Azure: common.AzureConfig{
					SubscriptionID: "12345678-ebb8-4b1f-8f96-d950d9e7aaaa",
				},
			},
			args:               nil,
			expectedGoldenFile: "run_template_cluster_capz_flux.golden",
		},
		{
			name: "case 16: template cluster capv (vsphere) with release version",
			flags: &flags.Flag{
				Name:              "test-capv-flux",
				Provider:          "vsphere",
				Description:       "cluster using release version directly",
				Release:           "35.0.0",
				Organization:      "test",
				KubernetesVersion: "v1.2.3",
				App: common.AppConfig{
					ClusterCatalog: "cluster",
				},
				VSphere: common.VSphereConfig{
					NetworkName:           "foonet",
					SvcLbIpPoolName:       "svc-foo-pool",
					CredentialsSecretName: "foosecret",
					ControlPlane: common.VSphereControlPlane{
						VSphereMachineTemplate: common.VSphereMachineTemplate{Replicas: 3},
					},
					Worker: common.VSphereMachineTemplate{Replicas: 3},
				},
			},
			args:               nil,
			expectedGoldenFile: "run_template_cluster_capv_flux.golden",
		},
		{
			name: "case 17: template cluster capvcd (cloud-director) with release version",
			flags: &flags.Flag{
				Name:         "test-capvcd-flux",
				Provider:     "cloud-director",
				Description:  "cluster using release version directly",
				Release:      "35.0.0",
				Organization: "test",
				App: common.AppConfig{
					ClusterCatalog: "cluster",
				},
				CloudDirector: common.CloudDirectorConfig{
					VipSubnet: "10.0.0.0/24",
				},
			},
			args:               nil,
			expectedGoldenFile: "run_template_cluster_capvcd_flux.golden",
		},
		{
			name: "case 18: template cluster eks with release version but no release-eks chart",
			flags: &flags.Flag{
				Name:         "test-eks-fallback",
				Provider:     "eks",
				Description:  "cluster using a release without a published release-eks chart",
				Release:      "35.0.0",
				Organization: "test",
				App: common.AppConfig{
					ClusterCatalog: "cluster",
					ClusterVersion: "1.0.0",
				},
			},
			args:               nil,
			expectedGoldenFile: "run_template_cluster_eks_fallback.golden",
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
				flag:      tc.flags,
				logger:    logger,
				stdout:    out,
				ociClient: releaseChartOCIClient,
			}

			k8sClient := kubeclient.FakeK8sClient()
			if tc.flags.Provider == "capa" {
				err = k8sClient.CtrlClient().Create(ctx, capaManagementCluster.DeepCopy())
				if err != nil {
					t.Fatalf("failed to fake AWSCluster object: %s", err.Error())
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

func Test_useReleaseChart(t *testing.T) {
	// release-aws:35.0.0 is the only tag published in this fake registry,
	// mirroring what's actually available in gsoci.
	publishedTags := map[string]bool{
		"charts/giantswarm/release-aws:35.0.0": true,
	}

	testCases := []struct {
		name           string
		provider       string
		release        string
		clusterVersion string
		expected       bool
		expectedWarn   string
	}{
		{
			name:     "case 0: legacy cluster chart version",
			provider: "capa",
			release:  "25.0.0",
			expected: false,
		},
		{
			name:     "case 1: published release version",
			provider: "capa",
			release:  "35.0.0",
			expected: true,
		},
		{
			name:         "case 2: release version without release chart falls back",
			provider:     "capa",
			release:      "35.0.0-andreas",
			expected:     false,
			expectedWarn: "no release-aws chart with version 35.0.0-andreas found",
		},
		{
			name:           "case 3: explicit cluster chart version wins",
			provider:       "capa",
			release:        "35.0.0",
			clusterVersion: "9.0.1-dev.private-karpenter.2026-08-20.16-33-07.h62652f9",
			expected:       false,
		},
		{
			name:         "case 4: release chart of another provider does not count",
			provider:     "capz",
			release:      "35.0.0",
			expected:     false,
			expectedWarn: "no release-azure chart with version 35.0.0 found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			stderr := new(bytes.Buffer)

			r := &runner{
				flag:      &flags.Flag{Provider: tc.provider},
				stderr:    stderr,
				ociClient: &fakeOCIClient{tags: publishedTags},
			}
			config := common.ClusterConfig{
				ReleaseVersion: tc.release,
				App: common.AppConfig{
					ClusterVersion: tc.clusterVersion,
				},
			}

			result := r.useReleaseChart(ctx, config)
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
