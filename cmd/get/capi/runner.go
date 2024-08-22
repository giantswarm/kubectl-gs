package capi

import (
	"context"
	"io"
	"strings"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/cli-runtime/pkg/printers"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/kubectl-gs/v5/pkg/commonconfig"
)

const (
	providerAll       = "All"
	providerAWS       = "AWS"
	providerAzure     = "Azure"
	providerOpenStack = "OpenStack"
	providerVSphere   = "vSphere"
)

type runner struct {
	commonConfig *commonconfig.CommonConfig
	flag         *flag
	logger       micrologger.Logger

	stdout io.Writer
}

type crd struct {
	DisplayName string
	Name        string
	Provider    string
}

type controller struct {
	DisplayName   string
	LabelSelector string
	ContainerName string
	Provider      string
}

var (
	crds = []crd{
		// All
		{
			DisplayName: "Cluster",
			Name:        "clusters.cluster.x-k8s.io",
			Provider:    providerAll,
		},
		{
			DisplayName: "ClusterResourceSet",
			Name:        "clusterresourcesets.addons.cluster.x-k8s.io",
			Provider:    providerAll,
		},
		{
			DisplayName: "ClusterResourceSetBinding",
			Name:        "clusterresourcesetbindings.addons.cluster.x-k8s.io",
			Provider:    providerAll,
		},
		{
			DisplayName: "Machine Pool",
			Name:        "machinepools.exp.cluster.x-k8s.io",
			Provider:    providerAll,
		},
		{
			DisplayName: "Machine Deployment",
			Name:        "machinedeployments.cluster.x-k8s.io",
			Provider:    providerAll,
		},
		{
			DisplayName: "Machine Set",
			Name:        "machinesets.cluster.x-k8s.io",
			Provider:    providerAll,
		},
		{
			DisplayName: "Machine",
			Name:        "machines.cluster.x-k8s.io",
			Provider:    providerAll,
		},
		{
			DisplayName: "Machine Health Check",
			Name:        "machinehealthchecks.cluster.x-k8s.io",
			Provider:    providerAll,
		},
		{
			DisplayName: "Kubeadm Control Plane",
			Name:        "kubeadmcontrolplanes.controlplane.cluster.x-k8s.io",
			Provider:    providerAll,
		},
		{
			DisplayName: "Kubeadm Config",
			Name:        "kubeadmconfigs.bootstrap.cluster.x-k8s.io",
			Provider:    providerAll,
		},
		{
			DisplayName: "Kubeadm Config Template",
			Name:        "kubeadmconfigtemplates.bootstrap.cluster.x-k8s.io",
			Provider:    providerAll,
		},
		// AWS
		{
			DisplayName: "AWS Cluster",
			Name:        "awsclusters.infrastructure.cluster.x-k8s.io",
			Provider:    providerAWS,
		},
		{
			DisplayName: "AWS Machine Pools",
			Name:        "awsmachinepools.infrastructure.cluster.x-k8s.io",
			Provider:    providerAWS,
		},
		{
			DisplayName: "AWS Machine",
			Name:        "awsmachines.infrastructure.cluster.x-k8s.io",
			Provider:    providerAWS,
		},
		{
			DisplayName: "AWS Machine Template",
			Name:        "awsmachinetemplates.infrastructure.cluster.x-k8s.io",
			Provider:    providerAWS,
		},
		// Azure
		{
			DisplayName: "Azure Cluster",
			Name:        "azureclusters.infrastructure.cluster.x-k8s.io",
			Provider:    providerAzure,
		},
		{
			DisplayName: "Azure Machine Pools",
			Name:        "azuremachinepools.exp.infrastructure.cluster.x-k8s.io",
			Provider:    providerAzure,
		},
		{
			DisplayName: "Azure Machine",
			Name:        "azuremachines.infrastructure.cluster.x-k8s.io",
			Provider:    providerAzure,
		},
		{
			DisplayName: "Azure Machine Template",
			Name:        "azuremachinetemplates.infrastructure.cluster.x-k8s.io",
			Provider:    providerAzure,
		},
		// Open Stack
		{
			DisplayName: "OpenStack Cluster",
			Name:        "openstackclusters.infrastructure.cluster.x-k8s.io",
			Provider:    providerOpenStack,
		},
		{
			DisplayName: "OpenStack Machine",
			Name:        "openstackmachines.infrastructure.cluster.x-k8s.io",
			Provider:    providerOpenStack,
		},
		{
			DisplayName: "OpenStack Machine Template",
			Name:        "openstackmachinetemplates.infrastructure.cluster.x-k8s.io",
			Provider:    providerOpenStack,
		},
		// vSphere
		{
			DisplayName: "vSphere Cluster",
			Name:        "vsphereclusters.infrastructure.cluster.x-k8s.io",
			Provider:    providerVSphere,
		},
		{
			DisplayName: "vSphere Machine",
			Name:        "vspheremachines.infrastructure.cluster.x-k8s.io",
			Provider:    providerVSphere,
		},
		{
			DisplayName: "vSphere Machine Template",
			Name:        "vspheremachinetemplates.infrastructure.cluster.x-k8s.io",
			Provider:    providerVSphere,
		},
		{
			DisplayName: "vSphere VM",
			Name:        "vspherevms.infrastructure.cluster.x-k8s.io",
			Provider:    providerVSphere,
		},
		{
			DisplayName: "vSphere HAProxy",
			Name:        "haproxyloadbalancers.infrastructure.cluster.x-k8s.io",
			Provider:    providerVSphere,
		},
	}

	controllers = []controller{
		{
			DisplayName:   "Core",
			LabelSelector: "app.kubernetes.io/name=cluster-api-core",
			ContainerName: "manager",
			Provider:      providerAll,
		},
		{
			DisplayName:   "Kubeadm Bootstrap",
			LabelSelector: "app.kubernetes.io/name=cluster-api-bootstrap-provider-kubeadm",
			ContainerName: "manager",
			Provider:      providerAll,
		},
		{
			DisplayName:   "Kubeadm Control Plane",
			LabelSelector: "app.kubernetes.io/name=cluster-api-control-plane",
			ContainerName: "manager",
			Provider:      providerAll,
		},
		{
			DisplayName:   "AWS Provider",
			LabelSelector: "app.kubernetes.io/name=cluster-api-provider-aws,control-plane=capa-controller-manager",
			ContainerName: "manager",
			Provider:      providerAWS,
		},
		{
			DisplayName:   "Azure Provider",
			LabelSelector: "app.kubernetes.io/name=cluster-api-provider-azure",
			ContainerName: "manager",
			Provider:      providerAzure,
		},
		{
			DisplayName:   "OpenStack Provider",
			LabelSelector: "cluster.x-k8s.io/provider=infrastructure-openstack,control-plane=capo-controller-manager",
			ContainerName: "manager",
			Provider:      providerOpenStack,
		},
		{
			DisplayName:   "vSphere Provider",
			LabelSelector: "app.kubernetes.io/name=cluster-api-provider-vsphere",
			ContainerName: "manager",
			Provider:      providerVSphere,
		},
	}
)

func (r *runner) Run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	err := r.flag.Validate()
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.run(ctx, cmd, args)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *runner) run(ctx context.Context, cmd *cobra.Command, args []string) error {
	var k8sClients k8sclient.Interface
	{
		var err error
		k8sClients, err = r.commonConfig.GetClient(r.logger)

		if err != nil {
			return microerror.Mask(err)
		}
	}

	table := &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Type", Type: "string"},
			{Name: "Provider", Type: "string"},
			{Name: "Name", Type: "string"},
			{Name: "Version", Type: "string"},
		},
	}

	crdList, err := k8sClients.ExtClient().ApiextensionsV1().CustomResourceDefinitions().List(ctx, metav1.ListOptions{})
	if err != nil {
		return microerror.Mask(err)
	}

	crdMap := map[string]apiextensionsv1.CustomResourceDefinition{}
	for _, crd := range crdList.Items {
		crdMap[crd.Name] = crd
	}

	for _, crd := range crds {
		foundCrd, ok := crdMap[crd.Name]
		if !ok {
			row := metav1.TableRow{Cells: []interface{}{"CRD", crd.Provider, crd.DisplayName, "NONE"}}
			table.Rows = append(table.Rows, row)
			continue
		}

		for _, version := range foundCrd.Spec.Versions {
			row := metav1.TableRow{Cells: []interface{}{"CRD", crd.Provider, crd.DisplayName, version.Name}}
			table.Rows = append(table.Rows, row)
		}
	}

	for _, controller := range controllers {
		selector, err := labels.Parse(controller.LabelSelector)
		if err != nil {
			return microerror.Mask(err)
		}

		var podList v1.PodList
		err = k8sClients.CtrlClient().List(ctx, &podList, &client.ListOptions{LabelSelector: selector})
		if err != nil {
			return microerror.Mask(err)
		}

		version := "NONE"
		for _, pod := range podList.Items {
			for _, container := range pod.Spec.Containers {
				if container.Name == "manager" {
					version = strings.Split(container.Image, ":")[1]
				}
			}
		}

		row := metav1.TableRow{Cells: []interface{}{"Controller", controller.Provider, controller.DisplayName, version}}
		table.Rows = append(table.Rows, row)
	}

	printer := printers.NewTablePrinter(printers.PrintOptions{})
	err = printer.PrintObj(table, r.stdout)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
