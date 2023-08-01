package provider

import (
	"context"
	"fmt"
	"io"
	"text/template"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/reference"
	"k8s.io/utils/ptr"
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/v2/internal/key"
	"github.com/giantswarm/kubectl-gs/v2/pkg/scheme"
)

const (
	defaultMasterVMSize = "Standard_D4s_v3"
)

func WriteAzureTemplate(ctx context.Context, client k8sclient.Interface, out io.Writer, config ClusterConfig) error {
	err := WriteGSAzureTemplate(ctx, client, out, config)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func WriteGSAzureTemplate(ctx context.Context, client k8sclient.Interface, out io.Writer, config ClusterConfig) error {
	var err error

	config.ReleaseComponents, err = key.GetReleaseComponents(ctx, client.CtrlClient(), config.ReleaseVersion)
	if err != nil {
		return microerror.Mask(err)
	}

	var azureClusterCRYaml, clusterCRYaml, azureMasterMachineCRYaml []byte
	{
		azureClusterCR := newAzureClusterCR(config)
		azureClusterCRYaml, err = yaml.Marshal(azureClusterCR)
		if err != nil {
			return microerror.Mask(err)
		}

		infrastructureRef := newCAPZClusterInfraRef(azureClusterCR)

		clusterCR := newcapiClusterCR(config, infrastructureRef)
		clusterCRYaml, err = yaml.Marshal(clusterCR)
		if err != nil {
			return microerror.Mask(err)
		}

		azureMasterMachineCR := newAzureMasterMachineCR(config)
		azureMasterMachineCRYaml, err = yaml.Marshal(azureMasterMachineCR)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	data := struct {
		ProviderClusterCR string
		MasterMachineCR   string
		ClusterCR         string
	}{
		ProviderClusterCR: string(azureClusterCRYaml),
		MasterMachineCR:   string(azureMasterMachineCRYaml),
		ClusterCR:         string(clusterCRYaml),
	}

	t := template.Must(template.New(config.FileName).Parse(key.ClusterAzureCRsTemplate))
	err = t.Execute(out, data)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func newAzureClusterCR(config ClusterConfig) *capz.AzureCluster {
	cr := &capz.AzureCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AzureCluster",
			APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Name,
			Namespace: config.Namespace,
			Labels: map[string]string{
				label.Cluster:              config.Name,
				capi.ClusterLabelName:      config.Name,
				label.Organization:         config.Organization,
				label.ReleaseVersion:       config.ReleaseVersion,
				label.AzureOperatorVersion: config.ReleaseComponents["azure-operator"],
			},
		},
		Spec: capz.AzureClusterSpec{
			ResourceGroup: config.Name,
			NetworkSpec: capz.NetworkSpec{
				APIServerLB: capz.LoadBalancerSpec{
					Name: fmt.Sprintf("%s-%s-%s", config.Name, "API", "PublicLoadBalancer"),
					LoadBalancerClassSpec: capz.LoadBalancerClassSpec{
						SKU:  "Standard",
						Type: "Public",
					},
					FrontendIPs: []capz.FrontendIP{
						{
							Name: fmt.Sprintf("%s-%s-%s-%s", config.Name, "API", "PublicLoadBalancer", "Frontend"),
						},
					},
				},
			},
		},
	}

	return cr
}

func newAzureMasterMachineCR(config ClusterConfig) *capz.AzureMachine {
	var failureDomain *string
	if len(config.ControlPlaneAZ) > 0 {
		failureDomain = &config.ControlPlaneAZ[0]
	}

	machine := &capz.AzureMachine{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AzureMachine",
			APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-master-%d", config.Name, 0),
			Namespace: config.Namespace,
			Labels: map[string]string{
				label.Cluster:                     config.Name,
				capi.ClusterLabelName:             config.Name,
				capi.MachineControlPlaneLabelName: "true",
				label.Organization:                config.Organization,
				label.ReleaseVersion:              config.ReleaseVersion,
				label.AzureOperatorVersion:        config.ReleaseComponents["azure-operator"],
			},
		},
		Spec: capz.AzureMachineSpec{
			VMSize:        defaultMasterVMSize,
			FailureDomain: failureDomain,
			Image: &capz.Image{
				Marketplace: &capz.AzureMarketplaceImage{
					Publisher: "kinvolk",
					Offer:     "flatcar-container-linux-free",
					SKU:       "stable",
					Version:   config.ReleaseComponents["containerlinux"],
				},
			},
			OSDisk: capz.OSDisk{
				OSType:      "Linux",
				CachingType: "ReadWrite",
				DiskSizeGB:  ptr.To[int32](50),
				ManagedDisk: &capz.ManagedDiskParameters{
					StorageAccountType: "Premium_LRS",
				},
			},
			SSHPublicKey: "",
		},
	}

	return machine
}

func newCAPZClusterInfraRef(obj client.Object) *corev1.ObjectReference {
	var infrastructureCRRef *corev1.ObjectReference
	{
		s, err := scheme.NewScheme()
		if err != nil {
			panic(microerror.Pretty(err, true))
		}

		infrastructureCRRef, err = reference.GetReference(s, obj)
		if err != nil {
			panic(fmt.Sprintf("cannot create reference to infrastructure CR: %q", err))
		}
	}

	return infrastructureCRRef
}
