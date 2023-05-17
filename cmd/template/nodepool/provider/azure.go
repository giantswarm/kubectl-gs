package provider

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"text/template"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1beta1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/reference"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/v2/internal/key"
	"github.com/giantswarm/kubectl-gs/v2/pkg/scheme"
)

func WriteAzureTemplate(ctx context.Context, client k8sclient.Interface, out io.Writer, config NodePoolCRsConfig) error {
	return WriteGSAzureTemplate(ctx, client, out, config)
}

func WriteGSAzureTemplate(ctx context.Context, client k8sclient.Interface, out io.Writer, config NodePoolCRsConfig) error {
	var err error
	config.ReleaseComponents, err = key.GetReleaseComponents(ctx, client.CtrlClient(), config.ReleaseVersion)
	if err != nil {
		return microerror.Mask(err)
	}

	var azureMachinePoolCRYaml, machinePoolCRYaml, sparkCRYaml []byte
	{
		azureMachinePoolCR := newAzureMachinePoolCR(config)
		azureMachinePoolCRYaml, err = yaml.Marshal(azureMachinePoolCR)
		if err != nil {
			return microerror.Mask(err)
		}

		sparkCR := newSparkCR(config)
		sparkCRYaml, err = yaml.Marshal(sparkCR)
		if err != nil {
			return microerror.Mask(err)
		}

		infrastructureRef := newCAPZMachinePoolInfraRef(azureMachinePoolCR)
		bootstrapConfigRef := newSparkCRRef(sparkCR)

		machinePoolCR := newcapiMachinePoolCR(config, infrastructureRef, bootstrapConfigRef)
		{
			machinePoolCR.GetAnnotations()[annotation.NodePoolMinSize] = strconv.Itoa(config.NodesMin)
			machinePoolCR.GetAnnotations()[annotation.NodePoolMaxSize] = strconv.Itoa(config.NodesMax)
		}
		machinePoolCRYaml, err = yaml.Marshal(machinePoolCR)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	data := struct {
		ProviderMachinePoolCR string
		MachinePoolCR         string
		SparkCR               string
	}{
		ProviderMachinePoolCR: string(azureMachinePoolCRYaml),
		MachinePoolCR:         string(machinePoolCRYaml),
		SparkCR:               string(sparkCRYaml),
	}

	t := template.Must(template.New(config.FileName).Parse(key.MachinePoolAzureCRsTemplate))
	err = t.Execute(out, data)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func newAzureMachinePoolCR(config NodePoolCRsConfig) *capzexp.AzureMachinePool {
	var spot *v1beta1.SpotVMOptions
	if config.AzureUseSpotVms {
		var maxPrice resource.Quantity
		if config.AzureSpotMaxPrice > 0 {
			maxPrice = resource.MustParse(fmt.Sprintf("%f", config.AzureSpotMaxPrice))

		} else {
			maxPrice = resource.MustParse("-1")
		}
		spot = &v1beta1.SpotVMOptions{
			MaxPrice: &maxPrice,
		}
	}

	azureMp := &capzexp.AzureMachinePool{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AzureMachinePool",
			APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.NodePoolName,
			Namespace: config.Namespace,
			Labels: map[string]string{
				label.Cluster:              config.ClusterName,
				capi.ClusterLabelName:      config.ClusterName,
				label.MachinePool:          config.NodePoolName,
				label.Organization:         config.Organization,
				label.AzureOperatorVersion: config.ReleaseComponents["azure-operator"],
				label.ReleaseVersion:       config.ReleaseVersion,
			},
		},
		Spec: capzexp.AzureMachinePoolSpec{
			Template: capzexp.AzureMachinePoolMachineTemplate{
				VMSize: config.VMSize,
				OSDisk: v1beta1.OSDisk{
					ManagedDisk: &v1beta1.ManagedDiskParameters{
						StorageAccountType: "",
					},
				},
				SSHPublicKey:  "",
				SpotVMOptions: spot,
			},
		},
	}

	return azureMp
}

func newCAPZMachinePoolInfraRef(obj client.Object) *corev1.ObjectReference {
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

func newSparkCRRef(obj client.Object) *corev1.ObjectReference {
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
