package provider

import (
	"fmt"
	"io"
	"text/template"

	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/reference"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/internal/key"
)

const (
	defaultMasterVMSize = "Standard_D4s_v3"
)

func WriteAzureTemplate(out io.Writer, config ClusterCRsConfig) error {
	var err error

	var azureClusterCRYaml, clusterCRYaml, azureMasterMachineCRYaml []byte
	{
		azureClusterCR := newAzureClusterCR(config)
		azureClusterCRYaml, err = yaml.Marshal(azureClusterCR)
		if err != nil {
			return microerror.Mask(err)
		}

		infrastructureRef := newCAPZClusterInfraRef(azureClusterCR)

		clusterCR := newCAPIV1Alpha3ClusterCR(config, infrastructureRef)
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

func newAzureClusterCR(config ClusterCRsConfig) *capzv1alpha3.AzureCluster {
	cr := &capzv1alpha3.AzureCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AzureCluster",
			APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha3",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.ClusterID,
			Namespace: config.Namespace,
			Labels: map[string]string{
				label.Cluster:                 config.ClusterID,
				capiv1alpha3.ClusterLabelName: config.ClusterID,
				label.Organization:            config.Owner,
				label.ReleaseVersion:          config.ReleaseVersion,
			},
		},
		Spec: capzv1alpha3.AzureClusterSpec{
			ResourceGroup: config.ClusterID,
			NetworkSpec: capzv1alpha3.NetworkSpec{
				APIServerLB: capzv1alpha3.LoadBalancerSpec{
					Name: "LB",
					SKU:  "Standard",
					Type: "Public",
					FrontendIPs: []capzv1alpha3.FrontendIP{
						{
							Name: "LB",
						},
					},
				},
			},
		},
	}

	return cr
}

func newAzureMasterMachineCR(config ClusterCRsConfig) *capzv1alpha3.AzureMachine {
	var failureDomain *string
	if len(config.MasterAZ) > 0 {
		failureDomain = &config.MasterAZ[0]
	}

	machine := &capzv1alpha3.AzureMachine{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AzureMachine",
			APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha3",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-master-%d", config.ClusterID, 0),
			Namespace: config.Namespace,
			Labels: map[string]string{
				label.Cluster:                             config.ClusterID,
				capiv1alpha3.ClusterLabelName:             config.ClusterID,
				capiv1alpha3.MachineControlPlaneLabelName: "true",
				label.Organization:                        config.Owner,
				label.ReleaseVersion:                      config.ReleaseVersion,
			},
		},
		Spec: capzv1alpha3.AzureMachineSpec{
			VMSize:        defaultMasterVMSize,
			FailureDomain: failureDomain,
			Image: &capzv1alpha3.Image{
				Marketplace: &capzv1alpha3.AzureMarketplaceImage{
					Publisher: "kinvolk",
					Offer:     "flatcar-container-linux-free",
					SKU:       "stable",
					Version:   "2345.3.1",
				},
			},
			OSDisk: capzv1alpha3.OSDisk{
				OSType:      "Linux",
				CachingType: "None",
				DiskSizeGB:  int32(50),
				ManagedDisk: capzv1alpha3.ManagedDisk{
					StorageAccountType: "Premium_LRS",
				},
			},
			SSHPublicKey: "",
		},
	}

	return machine
}

func newCAPZClusterInfraRef(obj runtime.Object) *corev1.ObjectReference {
	var err error
	var infrastructureCRRef *corev1.ObjectReference
	{
		s := runtime.NewScheme()
		err = capzv1alpha3.AddToScheme(s)
		if err != nil {
			panic(fmt.Sprintf("capzv1alpha3.AddToScheme: %+v", err))
		}

		infrastructureCRRef, err = reference.GetReference(s, obj)
		if err != nil {
			panic(fmt.Sprintf("cannot create reference to infrastructure CR: %q", err))
		}
	}

	return infrastructureCRRef
}
