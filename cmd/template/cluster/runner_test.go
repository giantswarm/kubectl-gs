package cluster

import (
	"bytes"
	"context"
	goflag "flag"
	"testing"

	"github.com/giantswarm/micrologger"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	capainfrav1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"

	//nolint:staticcheck
	"github.com/giantswarm/kubectl-gs/v5/cmd/template/cluster/common"
	"github.com/giantswarm/kubectl-gs/v5/cmd/template/cluster/flags"
	"github.com/giantswarm/kubectl-gs/v5/pkg/output"
	"github.com/giantswarm/kubectl-gs/v5/test/goldenfile"
	"github.com/giantswarm/kubectl-gs/v5/test/kubeclient"
)

var update = goflag.Bool("update", false, "update .golden reference test files")

// Test_run uses golden files.
//
// go test ./cmd/template/cluster -run Test_run -update
func Test_run(t *testing.T) {
	capaManagementCluster := &capainfrav1.AWSCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "my-mc",
			Namespace:       "org-giantswarm",
			ResourceVersion: "",
		},
		Status: capainfrav1.AWSClusterStatus{
			Network: capainfrav1.NetworkStatus{
				NatGatewaysIPs: []string{
					"1.2.3.4",
					"5.6.7.8",
					"9.10.11.12",
				},
			},
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
			name: "case 0: template cluster gcp",
			flags: &flags.Flag{
				Name:         "test1",
				Provider:     "gcp",
				Description:  "just a test cluster",
				Region:       "the-region",
				Organization: "test",
				App: common.AppConfig{
					ClusterVersion:     "1.0.0",
					ClusterCatalog:     "the-catalog",
					DefaultAppsCatalog: "the-default-catalog",
					DefaultAppsVersion: "2.0.0",
				},
				GCP: common.GCPConfig{
					Project:        "the-project",
					FailureDomains: []string{"failure-domain1-a", "failure-domain1-b"},
					ControlPlane: common.GCPControlPlane{
						ServiceAccount: common.ServiceAccount{
							Email:  "service-account@email",
							Scopes: []string{"scope1", "scope2"},
						},
					},
					MachineDeployment: common.GCPMachineDeployment{
						Name:             "worker1",
						FailureDomain:    "failure-domain2-b",
						InstanceType:     "very-large",
						Replicas:         7,
						RootVolumeSizeGB: 5,
						ServiceAccount: common.ServiceAccount{
							Email:  "service-account@email",
							Scopes: []string{"scope1", "scope2"},
						},
					},
				},
			},
			args:               nil,
			expectedGoldenFile: "run_template_cluster_gcp.golden",
		},
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
					ClusterVersion:     "1.0.0",
					ClusterCatalog:     "the-catalog",
					DefaultAppsCatalog: "the-default-catalog",
					DefaultAppsVersion: "2.0.0",
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
					ClusterVersion:     "1.0.0",
					ClusterCatalog:     "the-catalog",
					DefaultAppsCatalog: "the-default-catalog",
					DefaultAppsVersion: "2.0.0",
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
					ClusterVersion:     "1.0.0",
					ClusterCatalog:     "the-catalog",
					DefaultAppsCatalog: "the-default-catalog",
					DefaultAppsVersion: "2.0.0",
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
			name: "case 5: template cluster capv (cluster-vsphere and default-apps-vsphere)",
			flags: &flags.Flag{
				Name:              "test1",
				Provider:          "vsphere",
				Description:       "yet another test cluster",
				Release:           "27.0.0",
				Organization:      "test",
				KubernetesVersion: "v1.2.3",
				App: common.AppConfig{
					ClusterVersion:     "0.59.0",
					ClusterCatalog:     "foo-catalog",
					DefaultAppsCatalog: "foo-default-catalog",
					DefaultAppsVersion: "3.2.1",
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
					ClusterVersion:     "1.0.0",
					ClusterCatalog:     "the-catalog",
					DefaultAppsCatalog: "the-default-catalog",
					DefaultAppsVersion: "2.0.0",
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
					ClusterVersion:     "1.0.0",
					ClusterCatalog:     "the-catalog",
					DefaultAppsCatalog: "the-default-catalog",
					DefaultAppsVersion: "2.0.0",
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
					ClusterVersion:     "1.0.0",
					ClusterCatalog:     "the-catalog",
					DefaultAppsCatalog: "the-default-catalog",
					DefaultAppsVersion: "2.0.0",
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
