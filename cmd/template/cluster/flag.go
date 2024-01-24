package cluster

import (
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/giantswarm/kubectl-gs/v2/cmd/template/cluster/provider"
	"github.com/giantswarm/kubectl-gs/v2/internal/key"
	"github.com/giantswarm/kubectl-gs/v2/pkg/labels"
)

const (
	flagEnableLongNames   = "enable-long-names"
	flagProvider          = "provider"
	flagManagementCluster = "management-cluster"

	// AWS only.
	flagAWSExternalSNAT       = "external-snat"
	flagAWSControlPlaneSubnet = "control-plane-subnet"

	flagAWSClusterRoleIdentityName                       = "aws-cluster-role-identity-name"
	flagNetworkAZUsageLimit                              = "az-usage-limit"
	flagNetworkVPCCidr                                   = "vpc-cidr"
	flagAWSClusterType                                   = "cluster-type"
	flagAWSHttpsProxy                                    = "https-proxy"
	flagAWSHttpProxy                                     = "http-proxy"
	flagAWSNoProxy                                       = "no-proxy"
	flagAWSAPIMode                                       = "api-mode"
	flagAWSVPCMode                                       = "vpc-mode"
	flagAWSTopologyMode                                  = "topology-mode"
	flagAWSPrefixListID                                  = "aws-prefix-list-id"
	flagAWSTransitGatewayID                              = "aws-transit-gateway-id"
	flagAWSControlPlaneLoadBalancerIngressAllowCIDRBlock = "control-plane-load-balancer-ingress-allow-cidr-block"
	flagAWSPublicSubnetMask                              = "public-subnet-size"
	flagAWSPrivateSubnetMask                             = "private-subnet-size"

	flagAWSMachinePoolMinSize          = "machine-pool-min-size"
	flagAWSMachinePoolMaxSize          = "machine-pool-max-size"
	flagAWSMachinePoolName             = "machine-pool-name"
	flagAWSMachinePoolAZs              = "machine-pool-azs"
	flagAWSMachinePoolInstanceType     = "machine-pool-instance-type"
	flagAWSMachinePoolRootVolumeSizeGB = "machine-pool-root-volume-size-gb"
	flagAWSMachinePoolCustomNodeLabels = "machine-pool-custom-node-labels"

	// Azure only
	flagAzureSubscriptionID = "azure-subscription-id"

	// GCP only.
	flagGCPProject                               = "gcp-project"
	flagGCPFailureDomains                        = "gcp-failure-domains"
	flagGCPControlPlaneServiceAccountEmail       = "gcp-control-plane-sa-email"
	flagGCPControlPlaneServiceAccountScopes      = "gcp-control-plane-sa-scopes"
	flagGCPMachineDeploymentName                 = "gcp-machine-deployment-name"
	flagGCPMachineDeploymentInstanceType         = "gcp-machine-deployment-instance-type"
	flagGCPMachineDeploymentFailureDomain        = "gcp-machine-deployment-failure-domain"
	flagGCPMachineDeploymentReplicas             = "gcp-machine-deployment-replicas"
	flagGCPMachineDeploymentRootDiskSize         = "gcp-machine-deployment-disk-size"
	flagGCPMachineDeploymentServiceAccountEmail  = "gcp-machine-deployment-sa-email"
	flagGCPMachineDeploymentServiceAccountScopes = "gcp-machine-deployment-sa-scopes"

	// App-based clusters only.
	flagClusterCatalog     = "cluster-catalog"
	flagClusterVersion     = "cluster-version"
	flagDefaultAppsCatalog = "default-apps-catalog"
	flagDefaultAppsVersion = "default-apps-version"

	// OpenStack only.
	flagOpenStackCloud                      = "cloud"
	flagOpenStackCloudConfig                = "cloud-config"
	flagOpenStackDNSNameservers             = "dns-nameservers"
	flagOpenStackExternalNetworkID          = "external-network-id"
	flagOpenStackNodeCIDR                   = "node-cidr"
	flagOpenStackBastionBootFromVolume      = "bastion-boot-from-volume" //nolint:gosec
	flagOpenStackNetworkName                = "network-name"
	flagOpenStackSubnetName                 = "subnet-name"
	flagOpenStackBastionDiskSize            = "bastion-disk-size"
	flagOpenStackBastionImage               = "bastion-image"
	flagOpenStackBastionMachineFlavor       = "bastion-machine-flavor"
	flagOpenStackControlPlaneBootFromVolume = "control-plane-boot-from-volume"
	flagOpenStackControlPlaneDiskSize       = "control-plane-disk-size"
	flagOpenStackControlPlaneImage          = "control-plane-image"
	flagOpenStackControlPlaneMachineFlavor  = "control-plane-machine-flavor"
	flagOpenStackWorkerBootFromVolume       = "worker-boot-from-volume"
	flagOpenStackWorkerDiskSize             = "worker-disk-size"
	flagOpenStackWorkerFailureDomain        = "worker-failure-domain"
	flagOpenStackWorkerImage                = "worker-image"
	flagOpenStackWorkerMachineFlavor        = "worker-machine-flavor"
	flagOpenStackWorkerReplicas             = "worker-replicas"

	// VSphere only.
	flagVSphereControlPlaneIP          = "vsphere-control-plane-ip"
	flagVSphereServiceLoadBalancerCIDR = "vsphere-service-load-balancer-cidr"
	flagVSphereNetworkName             = "vsphere-network-name"
	flagVSphereSvcLbIpPool             = "vsphere-service-lb-pool"
	flagVSphereControlPlaneDiskGiB     = "vsphere-control-plane-disk-gib"
	flagVSphereControlPlaneIpPool      = "vsphere-control-plane-ip-pool"
	flagVSphereControlPlaneMemoryMiB   = "vsphere-control-plane-memory-mib"
	flagVSphereControlPlaneNumCPUs     = "vsphere-control-plane-num-cpus"
	flagVSphereControlPlaneReplicas    = "vsphere-control-plane-replicas"
	flagVSphereWorkerDiskGiB           = "vsphere-worker-disk-gib"
	flagVSphereWorkerMemoryMiB         = "vsphere-worker-memory-mib"
	flagVSphereWorkerNumCPUs           = "vsphere-worker-num-cpus"
	flagVSphereWorkerReplicas          = "vsphere-worker-replicas"
	flagVSphereResourcePool            = "vsphere-resource-pool"
	flagVSphereImageTemplate           = "vsphere-image-template"
	flagVSphereCredentialsSecretName   = "vsphere-credentials-secret-name" // #nosec G101

	// Common.
	flagRegion                   = "region"
	flagBastionInstanceType      = "bastion-instance-type"
	flagBastionReplicas          = "bastion-replicas"
	flagControlPlaneInstanceType = "control-plane-instance-type"
	flagControlPlaneAZ           = "control-plane-az"
	flagDescription              = "description"
	flagGenerateName             = "generate-name"
	flagKubernetesVersion        = "kubernetes-version"
	flagName                     = "name"
	flagOIDCIssuerURL            = "oidc-issuer-url"
	flagOIDCCAFile               = "oidc-ca-file"
	flagOIDCClientID             = "oidc-client-id"
	flagOIDCUsernameClaim        = "oidc-username-claim"
	flagOIDCGroupsClaim          = "oidc-groups-claim"
	flagOutput                   = "output"
	flagOrganization             = "organization"
	flagPodsCIDR                 = "pods-cidr"
	flagRelease                  = "release"
	flagLabel                    = "label"
	flagServicePriority          = "service-priority"

	// defaults
	defaultKubernetesVersion        = "v1.20.9"
	defaultVSphereKubernetesVersion = "v1.24.12"
)

type flag struct {
	EnableLongNames   bool
	Provider          string
	ManagementCluster string

	// Common.
	ControlPlaneAZ           []string
	Description              string
	GenerateName             bool
	KubernetesVersion        string
	Name                     string
	Output                   string
	Organization             string
	PodsCIDR                 string
	Release                  string
	Label                    []string
	Region                   string
	BastionInstanceType      string
	BastionReplicas          int
	ControlPlaneInstanceType string
	ServicePriority          string

	// Provider-specific
	AWS       provider.AWSConfig
	Azure     provider.AzureConfig
	GCP       provider.GCPConfig
	OpenStack provider.OpenStackConfig
	VSphere   provider.VSphereConfig
	App       provider.AppConfig
	OIDC      provider.OIDC

	print *genericclioptions.PrintFlags
}

func (f *flag) Init(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&f.EnableLongNames, flagEnableLongNames, true, "Allow long names.")
	cmd.Flags().StringVar(&f.Provider, flagProvider, "", "Installation infrastructure provider.")
	cmd.Flags().StringVar(&f.ManagementCluster, flagManagementCluster, "", "Name of the management cluster. Only required in combination with certain parameters.")

	// AWS only.
	cmd.Flags().StringVar(&f.AWS.AWSClusterRoleIdentityName, flagAWSClusterRoleIdentityName, "", "Name of the AWSClusterRoleIdentity that will be used for cluster creation.")
	cmd.Flags().IntVar(&f.AWS.NetworkAZUsageLimit, flagNetworkAZUsageLimit, 3, "Amount of AZs that will be used for VPC.")
	cmd.Flags().StringVar(&f.AWS.NetworkVPCCIDR, flagNetworkVPCCidr, "", "CIDR for the VPC.")
	cmd.Flags().BoolVar(&f.AWS.ExternalSNAT, flagAWSExternalSNAT, false, "AWS CNI configuration.")
	cmd.Flags().StringVar(&f.AWS.ClusterType, flagAWSClusterType, "public", "Cluster type to be created (public,proxy-private)")
	cmd.Flags().StringVar(&f.AWS.HttpsProxy, flagAWSHttpsProxy, "", "'HTTPS_PROXY' env value configuration for the cluster (required if cluster-type is set to proxy-private)")
	cmd.Flags().StringVar(&f.AWS.HttpProxy, flagAWSHttpProxy, "", "'HTTP_PROXY' env value configuration for the cluster, if not set, --https-proxy value will be used instead")
	cmd.Flags().StringVar(&f.AWS.NoProxy, flagAWSNoProxy, "", "'NO_PROXY' env value configuration for the cluster")
	cmd.Flags().StringVar(&f.AWS.APIMode, flagAWSAPIMode, "", "API mode of the network (public,private)")
	cmd.Flags().StringVar(&f.AWS.VPCMode, flagAWSVPCMode, "", "VPC mode of the network (public,private)")
	cmd.Flags().StringVar(&f.AWS.TopologyMode, flagAWSTopologyMode, "", "Topology mode of the network (UserManaged,GiantSwarmManaged,None)")
	cmd.Flags().StringVar(&f.AWS.PrefixListID, flagAWSPrefixListID, "", "Prefix list ID to manage. Workload cluster will be able to reach the destinations in the prefix list via the transit gateway. If not specified, it will be looked up by name/namespace of the management cluster (ends with `-tgw-prefixlist`). Only applies to proxy-private clusters.")
	cmd.Flags().StringVar(&f.AWS.TransitGatewayID, flagAWSTransitGatewayID, "", "ID of the transit gateway to attach the cluster VPC to. If not specified for workload clusters, the management cluster's transit gateway will be used. Only applies to proxy-private clusters.")
	cmd.Flags().IntVar(&f.AWS.PublicSubnetMask, flagAWSPublicSubnetMask, 20, "Subnet mask of the public subnets. Minimum is 25 (128 IPs), default is 20.")
	cmd.Flags().IntVar(&f.AWS.PrivateSubnetMask, flagAWSPrivateSubnetMask, 18, "Subnet mask of the private subnets. Minimum size is 25 (128 IPs), default is 18.")

	// aws control plane
	cmd.Flags().StringVar(&f.AWS.ControlPlaneSubnet, flagAWSControlPlaneSubnet, "", "Subnet used for the Control Plane.")
	cmd.Flags().StringArrayVar(&f.AWS.ControlPlaneLoadBalancerIngressAllowCIDRBlocks, flagAWSControlPlaneLoadBalancerIngressAllowCIDRBlock, nil, fmt.Sprintf("IPv4 address ranges that are allowed to connect to the control plane load balancer, in CIDR notation. When setting this flag, kubectl-gs automatically adds the NAT Gateway IPs of the management cluster so that the workload cluster can still be managed. If only the management cluster's IP ranges should be allowed, specify one empty value instead of an IP range ('--%s \"\"'). Supported for CAPA. You also need to specify --%s.", flagAWSControlPlaneLoadBalancerIngressAllowCIDRBlock, flagManagementCluster))
	// aws machine pool
	cmd.Flags().StringVar(&f.AWS.MachinePool.Name, flagAWSMachinePoolName, "nodepool0", "AWS Machine pool name")
	cmd.Flags().StringVar(&f.AWS.MachinePool.InstanceType, flagAWSMachinePoolInstanceType, "m5.xlarge", "AWS Machine pool instance type")
	cmd.Flags().IntVar(&f.AWS.MachinePool.MinSize, flagAWSMachinePoolMinSize, 3, "AWS Machine pool min size")
	cmd.Flags().IntVar(&f.AWS.MachinePool.MaxSize, flagAWSMachinePoolMaxSize, 10, "AWS Machine pool max size")
	cmd.Flags().IntVar(&f.AWS.MachinePool.RootVolumeSizeGB, flagAWSMachinePoolRootVolumeSizeGB, 300, "AWS Machine pool disk size")
	cmd.Flags().StringSliceVar(&f.AWS.MachinePool.AZs, flagAWSMachinePoolAZs, []string{}, "AWS Machine pool availability zones")
	cmd.Flags().StringSliceVar(&f.AWS.MachinePool.CustomNodeLabels, flagAWSMachinePoolCustomNodeLabels, []string{}, "AWS Machine pool custom node labels")

	// Azure only
	cmd.Flags().StringVar(&f.Azure.SubscriptionID, flagAzureSubscriptionID, "", "Azure subscription ID")

	// GCP only.
	cmd.Flags().StringVar(&f.GCP.Project, flagGCPProject, "", "Google Cloud Platform project name")
	cmd.Flags().StringSliceVar(&f.GCP.FailureDomains, flagGCPFailureDomains, nil, "Google Cloud Platform cluster failure domains")

	cmd.Flags().StringVar(&f.GCP.ControlPlane.ServiceAccount.Email, flagGCPControlPlaneServiceAccountEmail, "default", "Google Cloud Platform Service Account used by the control plane")
	cmd.Flags().StringSliceVar(&f.GCP.ControlPlane.ServiceAccount.Scopes, flagGCPControlPlaneServiceAccountScopes, []string{"https://www.googleapis.com/auth/compute"}, "Scope of the control plane's Google Cloud Platform Service Account")

	cmd.Flags().StringVar(&f.GCP.MachineDeployment.Name, flagGCPMachineDeploymentName, "worker0", "Google Cloud Platform project name")
	cmd.Flags().StringVar(&f.GCP.MachineDeployment.InstanceType, flagGCPMachineDeploymentInstanceType, "n1-standard-4", "Google Cloud Platform worker instance type")
	cmd.Flags().IntVar(&f.GCP.MachineDeployment.Replicas, flagGCPMachineDeploymentReplicas, 3, "Google Cloud Platform worker replicas")
	cmd.Flags().StringVar(&f.GCP.MachineDeployment.FailureDomain, flagGCPMachineDeploymentFailureDomain, "europe-west6-a", "Google Cloud Platform worker failure domain")
	cmd.Flags().IntVar(&f.GCP.MachineDeployment.RootVolumeSizeGB, flagGCPMachineDeploymentRootDiskSize, 100, "Google Cloud Platform worker root disk size")
	cmd.Flags().StringVar(&f.GCP.MachineDeployment.ServiceAccount.Email, flagGCPMachineDeploymentServiceAccountEmail, "default", "Google Cloud Platform Service Account used by the worker nodes")
	cmd.Flags().StringSliceVar(&f.GCP.MachineDeployment.ServiceAccount.Scopes, flagGCPMachineDeploymentServiceAccountScopes, []string{"https://www.googleapis.com/auth/compute"}, "Scope of the worker nodes' Google Cloud Platform Service Account")

	// OpenStack only.
	cmd.Flags().StringVar(&f.OpenStack.Cloud, flagOpenStackCloud, "", "Name of cloud (OpenStack only).")
	cmd.Flags().StringVar(&f.OpenStack.CloudConfig, flagOpenStackCloudConfig, "", "Name of cloud config (OpenStack only).")
	cmd.Flags().StringSliceVar(&f.OpenStack.DNSNameservers, flagOpenStackDNSNameservers, nil, "DNS nameservers (OpenStack only).")
	cmd.Flags().StringVar(&f.OpenStack.ExternalNetworkID, flagOpenStackExternalNetworkID, "", "External network ID (OpenStack only).")
	cmd.Flags().StringVar(&f.OpenStack.NodeCIDR, flagOpenStackNodeCIDR, "", "CIDR used for the nodes (OpenStack only).")
	cmd.Flags().StringVar(&f.OpenStack.NetworkName, flagOpenStackNetworkName, "", "Existing network name. Used when CIDR for nodes are not set (OpenStack only).")
	cmd.Flags().StringVar(&f.OpenStack.SubnetName, flagOpenStackSubnetName, "", "Existing subnet name. Used when CIDR for nodes are not set (OpenStack only).")
	// bastion
	cmd.Flags().BoolVar(&f.OpenStack.Bastion.BootFromVolume, flagOpenStackBastionBootFromVolume, false, "Bastion boot from volume (OpenStack only).")
	cmd.Flags().IntVar(&f.OpenStack.Bastion.DiskSize, flagOpenStackBastionDiskSize, 10, "Bastion machine root volume disk size (OpenStack only).")
	cmd.Flags().StringVar(&f.OpenStack.Bastion.Image, flagOpenStackBastionImage, "", "Bastion machine image (OpenStack only).")
	cmd.Flags().StringVar(&f.OpenStack.Bastion.Flavor, flagOpenStackBastionMachineFlavor, "", "Bastion machine flavor (OpenStack only).")
	// control plane
	cmd.Flags().BoolVar(&f.OpenStack.ControlPlane.BootFromVolume, flagOpenStackControlPlaneBootFromVolume, false, "Control plane boot from volume (OpenStack only).")
	cmd.Flags().IntVar(&f.OpenStack.ControlPlane.DiskSize, flagOpenStackControlPlaneDiskSize, 0, "Control plane machine root volume disk size (OpenStack only).")
	cmd.Flags().StringVar(&f.OpenStack.ControlPlane.Image, flagOpenStackControlPlaneImage, "", "Control plane machine image (OpenStack only).")
	cmd.Flags().StringVar(&f.OpenStack.ControlPlane.Flavor, flagOpenStackControlPlaneMachineFlavor, "", "Control plane machine flavor (OpenStack only).")
	// workers
	cmd.Flags().BoolVar(&f.OpenStack.Worker.BootFromVolume, flagOpenStackWorkerBootFromVolume, false, "Default worker node pool boot from volume (OpenStack only).")
	cmd.Flags().IntVar(&f.OpenStack.Worker.DiskSize, flagOpenStackWorkerDiskSize, 0, "Default worker node pool machine root volume disk size (OpenStack only).")
	cmd.Flags().StringVar(&f.OpenStack.WorkerFailureDomain, flagOpenStackWorkerFailureDomain, "", "Default worker node pool failure domain (OpenStack only).")
	cmd.Flags().StringVar(&f.OpenStack.Worker.Image, flagOpenStackWorkerImage, "", "Default worker node pool machine image name (OpenStack only).")
	cmd.Flags().StringVar(&f.OpenStack.Worker.Flavor, flagOpenStackWorkerMachineFlavor, "", "Default worker node pool machine flavor (OpenStack only).")
	cmd.Flags().IntVar(&f.OpenStack.WorkerReplicas, flagOpenStackWorkerReplicas, 0, "Default worker node pool replicas (OpenStack only).")

	// VSphere only
	cmd.Flags().StringVar(&f.VSphere.ControlPlane.Ip, flagVSphereControlPlaneIP, "", "Control plane IP, leave empty for auto allocation.")
	cmd.Flags().StringVar(&f.VSphere.ServiceLoadBalancerCIDR, flagVSphereServiceLoadBalancerCIDR, "", "CIDR for Service LB for new cluster")
	cmd.Flags().StringVar(&f.VSphere.NetworkName, flagVSphereNetworkName, "", "Network name in vcenter that should be used for the new VMs")
	cmd.Flags().StringVar(&f.VSphere.SvcLbIpPoolName, flagVSphereSvcLbIpPool, "svc-lb-ips", "Name of `GlobalInClusterIpPool` CR from which the IP for Service LB (kubevip) is taken")
	cmd.Flags().StringVar(&f.VSphere.ControlPlane.IpPoolName, flagVSphereControlPlaneIpPool, "wc-cp-ips", "Name of `GlobalInClusterIpPool` CR from which the IP for CP is taken")
	cmd.Flags().IntVar(&f.VSphere.ControlPlane.DiskGiB, flagVSphereControlPlaneDiskGiB, 50, "Disk size in GiB for control individual plane nodes")
	cmd.Flags().IntVar(&f.VSphere.ControlPlane.MemoryMiB, flagVSphereControlPlaneMemoryMiB, 8096, "Memory size in MiB for individual control plane nodes")
	cmd.Flags().IntVar(&f.VSphere.ControlPlane.NumCPUs, flagVSphereControlPlaneNumCPUs, 4, "Number of CPUs for individual control plane nodes")
	cmd.Flags().IntVar(&f.VSphere.ControlPlane.Replicas, flagVSphereControlPlaneReplicas, 3, "Number of control plane replicas (use odd number)")
	cmd.Flags().IntVar(&f.VSphere.Worker.DiskGiB, flagVSphereWorkerDiskGiB, 50, "Disk size in GiB for control individual worker nodes")
	cmd.Flags().IntVar(&f.VSphere.Worker.MemoryMiB, flagVSphereWorkerMemoryMiB, 14144, "Memory size in MiB for individual worker plane nodes")
	cmd.Flags().IntVar(&f.VSphere.Worker.NumCPUs, flagVSphereWorkerNumCPUs, 6, "Number of CPUs for individual worker plane nodes")
	cmd.Flags().IntVar(&f.VSphere.Worker.Replicas, flagVSphereWorkerReplicas, 3, "Number of worker plane replicas")
	cmd.Flags().StringVar(&f.VSphere.ResourcePool, flagVSphereResourcePool, "*/Resources", "What resource pool in vsphere should be used")
	cmd.Flags().StringVar(&f.VSphere.ImageTemplate, flagVSphereImageTemplate, "flatcar-stable-3602.2.1-kube-%s-gs", "OS images with Kubernetes that should be used for VMs. These template should be available in vCenter. The '%s' will be replaced with correct Kubernetes version. Example: 'ubuntu-2004-kube-%%s'")
	cmd.Flags().StringVar(&f.VSphere.CredentialsSecretName, flagVSphereCredentialsSecretName, "vsphere-credentials", "Name of the secret in K8s that should be associated to cluster app. It should exist in the organization's namesapce and should contain the credentials for vsphere.")

	// App-based clusters only.
	cmd.Flags().StringVar(&f.App.ClusterCatalog, flagClusterCatalog, "cluster", "Catalog for cluster app.")
	cmd.Flags().StringVar(&f.App.ClusterVersion, flagClusterVersion, "", "Version of cluster to be created.")
	cmd.Flags().StringVar(&f.App.DefaultAppsCatalog, flagDefaultAppsCatalog, "cluster", "Catalog for cluster default apps app.")
	cmd.Flags().StringVar(&f.App.DefaultAppsVersion, flagDefaultAppsVersion, "", "Version of default apps to be created.")

	// TODO: Make these flags visible once we have a better method for displaying provider-specific flags.
	_ = cmd.Flags().MarkHidden(flagOpenStackCloud)
	_ = cmd.Flags().MarkHidden(flagOpenStackCloudConfig)
	_ = cmd.Flags().MarkHidden(flagOpenStackDNSNameservers)
	_ = cmd.Flags().MarkHidden(flagOpenStackExternalNetworkID)
	_ = cmd.Flags().MarkHidden(flagOpenStackNodeCIDR)
	_ = cmd.Flags().MarkHidden(flagOpenStackNetworkName)
	_ = cmd.Flags().MarkHidden(flagOpenStackSubnetName)
	_ = cmd.Flags().MarkHidden(flagOpenStackBastionMachineFlavor)
	_ = cmd.Flags().MarkHidden(flagOpenStackBastionDiskSize)
	_ = cmd.Flags().MarkHidden(flagOpenStackBastionImage)
	_ = cmd.Flags().MarkHidden(flagOpenStackControlPlaneDiskSize)
	_ = cmd.Flags().MarkHidden(flagOpenStackControlPlaneImage)
	_ = cmd.Flags().MarkHidden(flagOpenStackControlPlaneMachineFlavor)
	_ = cmd.Flags().MarkHidden(flagOpenStackWorkerDiskSize)
	_ = cmd.Flags().MarkHidden(flagOpenStackWorkerFailureDomain)
	_ = cmd.Flags().MarkHidden(flagOpenStackWorkerImage)
	_ = cmd.Flags().MarkHidden(flagOpenStackWorkerMachineFlavor)
	_ = cmd.Flags().MarkHidden(flagOpenStackWorkerReplicas)

	_ = cmd.Flags().MarkHidden(flagRegion)
	_ = cmd.Flags().MarkHidden(flagAWSClusterRoleIdentityName)
	_ = cmd.Flags().MarkHidden(flagBastionInstanceType)
	_ = cmd.Flags().MarkHidden(flagBastionReplicas)
	_ = cmd.Flags().MarkHidden(flagNetworkVPCCidr)
	_ = cmd.Flags().MarkHidden(flagNetworkAZUsageLimit)
	_ = cmd.Flags().MarkHidden(flagControlPlaneInstanceType)
	_ = cmd.Flags().MarkHidden(flagAWSMachinePoolName)
	_ = cmd.Flags().MarkHidden(flagAWSMachinePoolInstanceType)
	_ = cmd.Flags().MarkHidden(flagAWSMachinePoolAZs)
	_ = cmd.Flags().MarkHidden(flagAWSMachinePoolCustomNodeLabels)
	_ = cmd.Flags().MarkHidden(flagAWSMachinePoolRootVolumeSizeGB)
	_ = cmd.Flags().MarkHidden(flagAWSMachinePoolMinSize)
	_ = cmd.Flags().MarkHidden(flagAWSMachinePoolMaxSize)
	_ = cmd.Flags().MarkHidden(flagAWSClusterType)
	_ = cmd.Flags().MarkHidden(flagAWSHttpsProxy)
	_ = cmd.Flags().MarkHidden(flagAWSHttpProxy)
	_ = cmd.Flags().MarkHidden(flagAWSNoProxy)
	_ = cmd.Flags().MarkHidden(flagAWSAPIMode)
	_ = cmd.Flags().MarkHidden(flagAWSVPCMode)
	_ = cmd.Flags().MarkHidden(flagAWSTopologyMode)
	_ = cmd.Flags().MarkHidden(flagAWSPrefixListID)
	_ = cmd.Flags().MarkHidden(flagAWSTransitGatewayID)

	_ = cmd.Flags().MarkHidden(flagGCPProject)
	_ = cmd.Flags().MarkHidden(flagGCPFailureDomains)
	_ = cmd.Flags().MarkHidden(flagGCPMachineDeploymentName)
	_ = cmd.Flags().MarkHidden(flagGCPMachineDeploymentFailureDomain)
	_ = cmd.Flags().MarkHidden(flagGCPMachineDeploymentRootDiskSize)
	_ = cmd.Flags().MarkHidden(flagGCPMachineDeploymentReplicas)
	_ = cmd.Flags().MarkHidden(flagGCPMachineDeploymentInstanceType)

	_ = cmd.Flags().MarkHidden(flagVSphereImageTemplate)

	_ = cmd.Flags().MarkHidden(flagClusterCatalog)
	_ = cmd.Flags().MarkHidden(flagClusterVersion)
	_ = cmd.Flags().MarkHidden(flagDefaultAppsCatalog)
	_ = cmd.Flags().MarkHidden(flagDefaultAppsVersion)

	// Common.
	cmd.Flags().StringSliceVar(&f.ControlPlaneAZ, flagControlPlaneAZ, nil, "Availability zone(s) to use by control plane nodes. Azure only supports one.")
	cmd.Flags().StringVar(&f.ControlPlaneInstanceType, flagControlPlaneInstanceType, "", "Instance type used for Control plane nodes")
	cmd.Flags().StringVar(&f.Description, flagDescription, "", "User-friendly description of the cluster's purpose (formerly called name).")
	cmd.Flags().StringVar(&f.KubernetesVersion, flagKubernetesVersion, defaultKubernetesVersion, "Cluster Kubernetes version.")
	cmd.Flags().StringVar(&f.Name, flagName, "", fmt.Sprintf("Unique identifier of the cluster. You must specify either --%s or --%s (except for vintage where, for CLI compatibility, we kept the old default of randomly generating a name if you do not specify one of these flags).", flagName, flagGenerateName))
	cmd.Flags().BoolVar(&f.GenerateName, flagGenerateName, false, fmt.Sprintf("Generate a random identifier of the cluster. We recommend to instead choose a name explicitly using --%s. You must specify either --%s or --%s (except for vintage where, for CLI compatibility, we kept the old default of randomly generating a name if you do not specify one of these flags).", flagName, flagName, flagGenerateName))
	cmd.Flags().StringVar(&f.OIDC.IssuerURL, flagOIDCIssuerURL, "", "OIDC issuer URL.")
	cmd.Flags().StringVar(&f.OIDC.CAFile, flagOIDCCAFile, "", "Path to CA file used to verify OIDC issuer (optional, OpenStack only).")
	cmd.Flags().StringVar(&f.OIDC.ClientID, flagOIDCClientID, "", "OIDC client ID.")
	cmd.Flags().StringVar(&f.OIDC.UsernameClaim, flagOIDCUsernameClaim, "email", "OIDC username claim.")
	cmd.Flags().StringVar(&f.OIDC.GroupsClaim, flagOIDCGroupsClaim, "groups", "OIDC groups claim.")
	cmd.Flags().StringVar(&f.Output, flagOutput, "", "File path for storing CRs.")
	cmd.Flags().StringVar(&f.Organization, flagOrganization, "", "Workload cluster organization.")
	cmd.Flags().StringVar(&f.PodsCIDR, flagPodsCIDR, "", "CIDR used for the pods.")
	cmd.Flags().StringVar(&f.Release, flagRelease, "", "Workload cluster release.")
	cmd.Flags().StringSliceVar(&f.Label, flagLabel, nil, "Workload cluster label.")
	cmd.Flags().StringVar(&f.ServicePriority, flagServicePriority, label.ServicePriorityHighest, fmt.Sprintf("Service priority of the cluster. Must be one of %v", getServicePriorities()))
	cmd.Flags().StringVar(&f.Region, flagRegion, "", "AWS/Azure/GCP region where cluster will be created")
	// bastion
	cmd.Flags().StringVar(&f.BastionInstanceType, flagBastionInstanceType, "", "Instance type used for the bastion node.")
	cmd.Flags().IntVar(&f.BastionReplicas, flagBastionReplicas, 1, "Replica count for the bastion node")

	_ = cmd.Flags().MarkHidden(flagEnableLongNames)
	_ = cmd.Flags().MarkDeprecated(flagEnableLongNames, "Long names are supported by default, so this flag is not needed anymore and will be removed in the next major version.")

	f.print = genericclioptions.NewPrintFlags("")
	f.print.OutputFormat = nil

	// Merging current command flags and config flags,
	// to be able to override kubectl-specific ones.
	f.print.AddFlags(cmd)
}

func (f *flag) Validate(cmd *cobra.Command) error {
	var err error
	validProviders := []string{
		key.ProviderAWS,
		key.ProviderAzure,
		key.ProviderCAPA,
		key.ProviderCAPZ,
		key.ProviderEKS,
		key.ProviderGCP,
		key.ProviderOpenStack,
		key.ProviderVSphere,
		key.ProviderCloudDirector,
	}
	isValidProvider := false
	for _, p := range validProviders {
		if f.Provider == p {
			isValidProvider = true
			break
		}
	}
	if !isValidProvider {
		return microerror.Maskf(invalidFlagError, "--%s must be one of: %s", flagProvider, strings.Join(validProviders, ", "))
	}

	// For vintage, don't break CLI parameter compatibility. But for CAPI or newer implementations, we want to enforce
	// an explicit choice for a specified name (`--name`) or randomly generated name (`--generate-name`).
	requireEitherNameOrGenerateNameFlag := key.IsPureCAPIProvider(f.Provider)

	if f.Name != "" {
		if f.GenerateName {
			return microerror.Maskf(
				invalidFlagError,
				"--%s and --%s are mutually exclusive. We recommend choosing a name explicitly using --%s.",
				flagName, flagGenerateName, flagName)
		}

		valid, err := key.ValidateName(f.Name, true)
		if err != nil {
			return microerror.Mask(err)
		} else if !valid {
			message := fmt.Sprintf("--%s must only contain alphanumeric characters, start with a letter", flagName)
			maxLength := key.NameLengthShort
			if f.EnableLongNames {
				maxLength = key.NameLengthLong
			}
			message += fmt.Sprintf(", and be no longer than %d characters in length", maxLength)
			return microerror.Maskf(invalidFlagError, message)
		}
	} else if !f.GenerateName {
		if requireEitherNameOrGenerateNameFlag {
			return microerror.Maskf(
				invalidFlagError,
				"Either --%s or --%s must be specified. We recommend choosing a name explicitly using --%s.",
				flagName, flagGenerateName, flagName)
		} else {
			// Keep supporting this old default for vintage (= no naming parameter given means to randomly
			// generate a cluster name)
			f.GenerateName = true
		}
	}

	if f.PodsCIDR != "" {
		if !validateCIDR(f.PodsCIDR) {
			return microerror.Maskf(invalidFlagError, "--%s must be a valid CIDR", flagPodsCIDR)
		}
	}

	if f.Organization == "" {
		return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagOrganization)
	}

	{
		// Validate Master AZs.
		switch f.Provider {
		case key.ProviderAWS:
			if len(f.ControlPlaneAZ) != 0 && len(f.ControlPlaneAZ) != 1 && len(f.ControlPlaneAZ) != 3 {
				return microerror.Maskf(invalidFlagError, "--%s must be set to either one or three availability zone names", flagControlPlaneAZ)
			}
			if f.AWS.ControlPlaneSubnet != "" {
				matchedSubnet, err := regexp.MatchString("^20|21|22|23|24|25$", f.AWS.ControlPlaneSubnet)
				if err == nil && !matchedSubnet {
					return microerror.Maskf(invalidFlagError, "--%s must be a valid subnet size (20, 21, 22, 23, 24 or 25)", flagAWSControlPlaneSubnet)
				}
			}
		case key.ProviderGCP:
			if f.Region == "" {
				return microerror.Maskf(invalidFlagError, "--%s is required", flagRegion)
			}
			if f.GCP.Project == "" {
				return microerror.Maskf(invalidFlagError, "--%s is required", flagGCPProject)
			}
			if f.GCP.FailureDomains == nil {
				return microerror.Maskf(invalidFlagError, "--%s is required", flagGCPFailureDomains)
			}
		case key.ProviderAzure:
			if len(f.ControlPlaneAZ) > 1 {
				return microerror.Maskf(invalidFlagError, "--%s supports one availability zone only", flagControlPlaneAZ)
			}
		case key.ProviderVSphere:
			if f.VSphere.NetworkName == "" {
				return microerror.Maskf(invalidFlagError, "Provide the network name in vcenter (required) (--%s)", flagVSphereNetworkName)
			}
			if f.VSphere.ServiceLoadBalancerCIDR != "" && !validateCIDR(f.VSphere.ServiceLoadBalancerCIDR) {
				return microerror.Maskf(invalidFlagError, "--%s must be a valid CIDR", flagVSphereServiceLoadBalancerCIDR)
			}
			if !cmd.Flags().Changed(flagKubernetesVersion) {
				f.KubernetesVersion = defaultVSphereKubernetesVersion
			}

			placeholders := strings.Count(f.VSphere.ImageTemplate, "%s")
			if placeholders > 1 {
				return microerror.Maskf(invalidFlagError, "--%s must contain at most one occurrence of '%%s' where k8s version will be injected", flagVSphereImageTemplate)
			}
			if placeholders == 1 {
				f.VSphere.ImageTemplate = fmt.Sprintf(f.VSphere.ImageTemplate, f.KubernetesVersion)
			}
			if f.VSphere.Worker.Replicas < 1 {
				return microerror.Maskf(invalidFlagError, "--%s must be greater than 0", flagVSphereWorkerReplicas)
			}
			if f.VSphere.ControlPlane.Replicas < 1 {
				return microerror.Maskf(invalidFlagError, "--%s must be greater than 0", flagVSphereControlPlaneReplicas)
			}
		case key.ProviderOpenStack:
			if f.OpenStack.Cloud == "" {
				return microerror.Maskf(invalidFlagError, "--%s is required", flagOpenStackCloud)
			}
			if f.OpenStack.CloudConfig == "" {
				return microerror.Maskf(invalidFlagError, "--%s is required", flagOpenStackCloudConfig)
			}
			if f.OpenStack.ExternalNetworkID == "" {
				return microerror.Maskf(invalidFlagError, "--%s is required", flagOpenStackExternalNetworkID)
			}
			if f.OpenStack.NodeCIDR != "" {
				if f.OpenStack.NetworkName != "" || f.OpenStack.SubnetName != "" {
					return microerror.Maskf(invalidFlagError, "--%s and --%s must be empty when --%s is used",
						flagOpenStackNetworkName, flagOpenStackSubnetName, flagOpenStackNodeCIDR)
				}
				if !validateCIDR(f.OpenStack.NodeCIDR) {
					return microerror.Maskf(invalidFlagError, "--%s must be a valid CIDR", flagOpenStackNodeCIDR)
				}
			} else {
				if f.OpenStack.NetworkName == "" || f.OpenStack.SubnetName == "" {
					return microerror.Maskf(invalidFlagError, "--%s and --%s must be set when --%s is empty",
						flagOpenStackNetworkName, flagOpenStackSubnetName, flagOpenStackNodeCIDR)
				}
			}
			// bastion
			if f.OpenStack.Bastion.BootFromVolume && f.OpenStack.Bastion.DiskSize < 1 {
				return microerror.Maskf(invalidFlagError, "--%s must be greater than 0 when --%s is specified", flagOpenStackBastionDiskSize, flagOpenStackBastionBootFromVolume)
			}
			if f.OpenStack.Bastion.Flavor == "" {
				return microerror.Maskf(invalidFlagError, "--%s is required", flagOpenStackBastionMachineFlavor)
			}
			if f.OpenStack.Bastion.Image == "" {
				return microerror.Maskf(invalidFlagError, "--%s is required", flagOpenStackBastionImage)
			}
			// control plane
			if f.OpenStack.ControlPlane.BootFromVolume && f.OpenStack.ControlPlane.DiskSize < 1 {
				return microerror.Maskf(invalidFlagError, "--%s must be greater than 0 when --%s is specified", flagOpenStackControlPlaneDiskSize, flagOpenStackControlPlaneBootFromVolume)
			}
			if f.OpenStack.ControlPlane.Flavor == "" {
				return microerror.Maskf(invalidFlagError, "--%s is required", flagOpenStackControlPlaneMachineFlavor)
			}
			if f.OpenStack.ControlPlane.Image == "" {
				return microerror.Maskf(invalidFlagError, "--%s is required", flagOpenStackControlPlaneImage)
			}
			// worker
			if f.OpenStack.WorkerReplicas < 1 {
				return microerror.Maskf(invalidFlagError, "--%s must be greater than 0", flagOpenStackWorkerReplicas)
			}
			if f.OpenStack.WorkerFailureDomain == "" {
				return microerror.Maskf(invalidFlagError, "--%s is required", flagOpenStackWorkerFailureDomain)
			}
			if len(f.ControlPlaneAZ) != 0 {
				if len(f.ControlPlaneAZ)%2 != 1 {
					return microerror.Maskf(invalidFlagError, "--%s must be an odd number number of values (usually 1 or 3 for non-HA and HA respectively)", flagControlPlaneAZ)
				}

				var validFailureDomain bool
				for _, az := range f.ControlPlaneAZ {
					if f.OpenStack.WorkerFailureDomain == az {
						validFailureDomain = true
						break
					}
				}

				if !validFailureDomain {
					return microerror.Maskf(invalidFlagError, "--%s must be among the AZs specified with --%s", flagOpenStackWorkerFailureDomain, flagControlPlaneAZ)
				}
			}
			if f.OpenStack.Worker.BootFromVolume && f.OpenStack.Worker.DiskSize < 1 {
				return microerror.Maskf(invalidFlagError, "--%s must be greater than 0 when --%s is specified", flagOpenStackWorkerDiskSize, flagOpenStackWorkerBootFromVolume)
			}
			if f.OpenStack.Worker.Flavor == "" {
				return microerror.Maskf(invalidFlagError, "--%s is required", flagOpenStackWorkerMachineFlavor)
			}
			if f.OpenStack.Worker.Image == "" {
				return microerror.Maskf(invalidFlagError, "--%s is required", flagOpenStackWorkerImage)
			}
		case key.ProviderCAPA:
			if f.AWS.ClusterType == "proxy-private" && (f.AWS.HttpsProxy == "" || f.AWS.NetworkVPCCIDR == "") {
				return microerror.Maskf(invalidFlagError, "--%s and --%s are required when proxy-private is selected", flagAWSHttpsProxy, flagNetworkVPCCidr)
			}

			if len(f.AWS.ControlPlaneLoadBalancerIngressAllowCIDRBlocks) > 0 && f.ManagementCluster == "" {
				return microerror.Maskf(invalidFlagError, "--%s must not be empty when specifying --%s", flagManagementCluster, flagAWSControlPlaneLoadBalancerIngressAllowCIDRBlock)
			}
		}

		if f.Release == "" {
			if key.IsPureCAPIProvider(f.Provider) {
				// skip release validation
			} else {
				return microerror.Maskf(invalidFlagError, "--%s must not be empty", flagRelease)
			}
		}

		_, err = labels.Parse(f.Label)
		if err != nil {
			return microerror.Maskf(invalidFlagError, "--%s must contain valid label definitions (%s)", flagLabel, err)
		}

		if !isValidServicePriority(f.ServicePriority) {
			return microerror.Maskf(invalidFlagError, "--%s value %s is invalid. Must be one of %v.", flagServicePriority, f.ServicePriority, getServicePriorities())
		}

	}

	return nil
}

func validateCIDR(cidr string) bool {
	_, _, err := net.ParseCIDR(cidr)

	return err == nil
}

func isValidServicePriority(servicePriority string) bool {
	validServicePriorities := getServicePriorities()
	for _, p := range validServicePriorities {
		if servicePriority == p {
			return true
		}
	}
	return false
}

func getServicePriorities() []string {
	return []string{label.ServicePriorityHighest, label.ServicePriorityMedium, label.ServicePriorityLowest}
}
