package capi

import (
	"context"
	"io"
	"strings"

	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/printers"

	"github.com/giantswarm/kubectl-gs/pkg/commonconfig"
)

const (
	giantswarmNamespace = "giantswarm"

	providerAll       = "All"
	providerAWS       = "AWS"
	providerAzure     = "Azure"
	providerOpenStack = "OpenStack"
	providerVSphere   = "vSphere"
)

type runner struct {
	flag   *flag
	logger micrologger.Logger

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
			Name:        "vsphereclusters.infrastructure.cluster.x-k8s.io",
			Provider:    providerOpenStack,
		},
		{
			DisplayName: "OpenStack Machine",
			Name:        "vspheremachines.infrastructure.cluster.x-k8s.io",
			Provider:    providerOpenStack,
		},
		{
			DisplayName: "OpenStack Machine Template",
			Name:        "vspheremachinetemplates.infrastructure.cluster.x-k8s.io",
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
			LabelSelector: "app.kubernetes.io/name=cluster-api-provider-openstack",
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
	var client k8sclient.Interface
	{
		config := commonconfig.New(r.flag.config)
		c, err := config.GetClient(r.logger)

		if err != nil {
			return microerror.Mask(err)
		}

		client = c.K8sClient
	}

	table := &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Type", Type: "string"},
			{Name: "Provider", Type: "string"},
			{Name: "Name", Type: "string"},
			{Name: "Version", Type: "string"},
		},
	}

	for _, crd := range crds {
		foundCrd, err := client.ExtClient().ApiextensionsV1().CustomResourceDefinitions().Get(
			ctx,
			crd.Name,
			metav1.GetOptions{},
		)
		if err != nil && !errors.IsNotFound(err) {
			return microerror.Mask(err)
		}

		if errors.IsNotFound(err) {
			row := metav1.TableRow{Cells: []interface{}{"CRD", crd.Provider, crd.DisplayName, "NONE"}}
			table.Rows = append(table.Rows, row)
		}

		for _, version := range foundCrd.Spec.Versions {
			row := metav1.TableRow{Cells: []interface{}{"CRD", crd.Provider, crd.DisplayName, version.Name}}
			table.Rows = append(table.Rows, row)
		}
	}

	for _, controller := range controllers {
		version := "NONE"

		podList, err := client.K8sClient().CoreV1().Pods(giantswarmNamespace).List(
			ctx,
			metav1.ListOptions{
				LabelSelector: controller.LabelSelector,
			},
		)
		if err != nil {
			return microerror.Mask(err)
		}

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
	err := printer.PrintObj(table, r.stdout)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
