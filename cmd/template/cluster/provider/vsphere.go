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
	capvv1alpha4 "sigs.k8s.io/cluster-api-provider-vsphere/api/v1alpha4"
	capiv1alpha4 "sigs.k8s.io/cluster-api/api/v1alpha4"

	"github.com/giantswarm/kubectl-gs/cmd/template/cluster/provider/templates/vsphere"
	"github.com/giantswarm/kubectl-gs/internal/key"
)

func WriteVSphereTemplate(out io.Writer, config ClusterCRsConfig) error {
	var err error

	data := struct {
		Description       string
		KubernetesVersion string
		Name              string
		Namespace         string
		Owner             string
		Version           string
		VMSize            string
	}{
		Description:       config.Description,
		KubernetesVersion: "v1.19.9",
		Name:              config.Name,
		Namespace:         key.OrganizationNamespaceFromName(config.Owner),
		Owner:             config.Owner,
		Version:           config.ReleaseVersion,
		VMSize:            "Standard_D4s_v3",
	}

	t := template.Must(template.New(config.FileName).Parse(azure.GetTemplate()))
	err = t.Execute(out, data)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func newVSphereClusterCR(config ClusterCRsConfig) *capvv1alpha4.VSphereCluster {
	cr := &capvv1alpha4.VSphereCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AzureCluster",
			APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha3",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Name,
			Namespace: config.Namespace,
			Labels: map[string]string{
				label.Cluster:                 config.Name,
				capiv1alpha4.ClusterLabelName: config.Name,
				label.Organization:            config.Owner,
				label.ReleaseVersion:          config.ReleaseVersion,
			},
		},
		Spec: capvv1alpha4.VSphereClusterSpec{
			Server:               "",
			Thumbprint:           "",
			ControlPlaneEndpoint: capvv1alpha4.APIEndpoint{},
			IdentityRef:          nil,
		},
	}

	return cr
}

func newVSphereMasterMachineCR(config ClusterCRsConfig) *capvv1alpha4.VSphereMachine {
	var failureDomain *string
	if len(config.ControlPlaneAZ) > 0 {
		failureDomain = &config.ControlPlaneAZ[0]
	}

	machine := &capvv1alpha4.VSphereMachine{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VSphereMachine",
			APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-master-%d", config.Name, 0),
			Namespace: config.Namespace,
			Labels: map[string]string{
				label.Cluster:                             config.Name,
				capiv1alpha4.ClusterLabelName:             config.Name,
				capiv1alpha4.MachineControlPlaneLabelName: "true",
				label.Organization:                        config.Owner,
				label.ReleaseVersion:                      config.ReleaseVersion,
			},
		},
		Spec: capvv1alpha4.VSphereMachineSpec{
			VirtualMachineCloneSpec: capvv1alpha4.VirtualMachineCloneSpec{},
			ProviderID:              nil,
			FailureDomain:           failureDomain,
		},
	}

	return machine
}

func newCAPVClusterInfraRef(obj runtime.Object) *corev1.ObjectReference {
	var err error
	var infrastructureCRRef *corev1.ObjectReference
	{
		s := runtime.NewScheme()
		err = capvv1alpha4.AddToScheme(s)
		if err != nil {
			panic(fmt.Sprintf("capvv1alpha4.AddToScheme: %+v", err))
		}

		infrastructureCRRef, err = reference.GetReference(s, obj)
		if err != nil {
			panic(fmt.Sprintf("cannot create reference to infrastructure CR: %q", err))
		}
	}

	return infrastructureCRRef
}
