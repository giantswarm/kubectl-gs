package cluster

import (
	"encoding/base64"
	"fmt"
	"github.com/giantswarm/apiextensions/pkg/annotation"
	"github.com/giantswarm/apiextensions/pkg/id"
	"github.com/giantswarm/apiextensions/pkg/label"
	"github.com/giantswarm/kubectl-gs/internal/key"
	"github.com/giantswarm/kubectl-gs/pkg/clusterlabels"
	"github.com/giantswarm/kubectl-gs/pkg/release"
	"github.com/giantswarm/microerror"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/reference"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/yaml"
	"strings"
	"text/template"
)

const (
	serviceNetworkCIDR  = "172.31.0.0/16"
	defaultMasterVMSize = "Standard_D4_v3"
)

type clusterCRConfig struct {
	ClusterID         string
	Credential        string
	Domain            string
	MasterAZ          []string
	Description       string
	Owner             string
	PublicSSHKey      string
	Region            string
	ReleaseComponents map[string]string
	ReleaseVersion    string
	Labels            map[string]string
	Namespace         string
}

func writeAzureTemplate(out io.Writer, name string, flags *flag) error {
	var err error

	var releaseComponents map[string]string
	{
		c := release.Config{}

		releaseCollection, err := release.New(c)
		if err != nil {
			return microerror.Mask(err)
		}

		releaseComponents = releaseCollection.ReleaseComponents(flags.Release)
	}

	var userLabels map[string]string
	{
		userLabels, err = clusterlabels.Parse(flags.Label)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var config clusterCRConfig
	{
		config = clusterCRConfig{
			ClusterID:         flags.ClusterID,
			Credential:        flags.Credential,
			Domain:            flags.Domain,
			MasterAZ:          flags.MasterAZ,
			Description:       flags.Name,
			Owner:             flags.Owner,
			Region:            flags.Region,
			ReleaseComponents: releaseComponents,
			ReleaseVersion:    flags.Release,
			Labels:            userLabels,
			Namespace:         metav1.NamespaceDefault,
		}

		// Remove leading 'v' from release flag input.
		config.ReleaseVersion = strings.TrimLeft(config.ReleaseVersion, "v")

		if config.ClusterID == "" {
			config.ClusterID = id.Generate()
		}
	}

	var azureClusterCRYaml, clusterCRYaml, azureMasterMachineCRYaml []byte
	{
		azureClusterCR := newAzureClusterCR(config)
		azureClusterCRYaml, err = yaml.Marshal(azureClusterCR)
		if err != nil {
			return microerror.Mask(err)
		}

		var clusterCR *capiv1alpha3.Cluster
		clusterCR, err = newCAPIV1Alpha3ClusterCR(config, azureClusterCR)
		if err != nil {
			return microerror.Mask(err)
		}
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
		AzureClusterCR       string
		AzureMasterMachineCR string
		ClusterCR            string
	}{
		AzureClusterCR:       string(azureClusterCRYaml),
		AzureMasterMachineCR: string(azureMasterMachineCRYaml),
		ClusterCR:            string(clusterCRYaml),
	}

	t := template.Must(template.New(name).Parse(key.ClusterAzureCRsTemplate))
	err = t.Execute(out, data)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func newAzureClusterCR(config clusterCRConfig) *capzv1alpha3.AzureCluster {
	cr := &capzv1alpha3.AzureCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AzureCluster",
			APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha3",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.ClusterID,
			Namespace: config.Namespace,
			Labels: map[string]string{
				label.AzureOperatorVersion:    config.ReleaseComponents["azure-operator"],
				label.Cluster:                 config.ClusterID,
				capiv1alpha3.ClusterLabelName: config.ClusterID,
				label.Organization:            config.Owner,
				label.ReleaseVersion:          config.ReleaseVersion,
			},
		},
		Spec: capzv1alpha3.AzureClusterSpec{
			Location: config.Region,
			ControlPlaneEndpoint: capiv1alpha3.APIEndpoint{
				Host: fmt.Sprintf("api.%s.k8s.%s", config.ClusterID, config.Domain),
				Port: 443,
			},
		},
	}

	return cr
}

func newCAPIV1Alpha3ClusterCR(config clusterCRConfig, infrastructureObj interface{}) (*capiv1alpha3.Cluster, error) {
	runtimeObj, ok := infrastructureObj.(runtime.Object)
	if !ok {
		panic(fmt.Sprintf("cannot alias %T as runtime.Object", infrastructureObj))
	}

	infraCR, err := meta.Accessor(infrastructureObj)
	if err != nil {
		return nil, microerror.Maskf(invalidObjectDefinitionError, fmt.Sprintf("cannot get metav1.Object from %T", infrastructureObj))
	}

	var infrastructureCRRef *corev1.ObjectReference
	{
		s := runtime.NewScheme()
		err = capzv1alpha3.AddToScheme(s)
		if err != nil {
			return nil, microerror.Maskf(invalidObjectDefinitionError, fmt.Sprintf("capzv1alph3.AddToScheme: %+v", err))
		}

		infrastructureCRRef, err = reference.GetReference(s, runtimeObj)
		if err != nil {
			return nil, microerror.Maskf(invalidObjectDefinitionError, fmt.Sprintf("cannot create reference to infrastructure CR: %q", err))
		}
	}

	httpsPort := int32(443)
	cluster := &capiv1alpha3.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "cluster.x-k8s.io/v1alpha3",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.ClusterID,
			Namespace: infraCR.GetNamespace(),
			Labels: map[string]string{
				// XXX: azure-operator reconciles Cluster & MachinePool to set OwnerReferences (for now).
				label.AzureOperatorVersion:    config.ReleaseComponents["azure-operator"],
				label.ClusterOperatorVersion:  config.ReleaseComponents["cluster-operator"],
				label.Cluster:                 config.ClusterID,
				capiv1alpha3.ClusterLabelName: config.ClusterID,
				label.Organization:            config.Owner,
				label.ReleaseVersion:          config.ReleaseVersion,
			},
			Annotations: map[string]string{
				annotation.ClusterDescription: config.Description,
			},
		},
		Spec: capiv1alpha3.ClusterSpec{
			ClusterNetwork: &capiv1alpha3.ClusterNetwork{
				APIServerPort: &httpsPort,
				Services: &capiv1alpha3.NetworkRanges{
					CIDRBlocks: []string{
						serviceNetworkCIDR,
					},
				},
				ServiceDomain: fmt.Sprintf("%s.k8s.%s", config.ClusterID, config.Domain),
			},
			ControlPlaneEndpoint: capiv1alpha3.APIEndpoint{
				Host: fmt.Sprintf("api.%s.k8s.%s", config.ClusterID, config.Domain),
				Port: 443,
			},
			InfrastructureRef: infrastructureCRRef,
		},
	}

	return cluster, nil
}

func newAzureMasterMachineCR(config clusterCRConfig) *capzv1alpha3.AzureMachine {
	publicSSHKeyEncoded := base64.StdEncoding.EncodeToString([]byte(config.PublicSSHKey))

	machine := &capzv1alpha3.AzureMachine{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AzureMachine",
			APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha3",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-master-%d", config.ClusterID, 0),
			Namespace: config.Namespace,
			Labels: map[string]string{
				label.AzureOperatorVersion:                config.ReleaseComponents["azure-operator"],
				label.Cluster:                             config.ClusterID,
				capiv1alpha3.ClusterLabelName:             config.ClusterID,
				capiv1alpha3.MachineControlPlaneLabelName: "true",
				label.Organization:                        config.Owner,
				label.ReleaseVersion:                      config.ReleaseVersion,
			},
		},
		Spec: capzv1alpha3.AzureMachineSpec{
			VMSize:        defaultMasterVMSize,
			FailureDomain: &config.MasterAZ[0],
			Image: &capzv1alpha3.Image{
				Marketplace: &capzv1alpha3.AzureMarketplaceImage{
					Publisher: "kinvolk",
					Offer:     "flatcar-container-linux-free",
					SKU:       "stable",
					Version:   "2345.3.1",
				},
			},
			OSDisk: capzv1alpha3.OSDisk{
				OSType:     "Linux",
				DiskSizeGB: int32(50),
				ManagedDisk: capzv1alpha3.ManagedDisk{
					StorageAccountType: "Premium_LRS",
				},
			},
			Location:     config.Region,
			SSHPublicKey: publicSSHKeyEncoded,
		},
	}

	return machine
}
