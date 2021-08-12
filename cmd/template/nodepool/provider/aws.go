package provider

import (
	"io"
	"os"
	"strconv"
	"text/template"

	"github.com/giantswarm/apiextensions/v3/pkg/annotation"
	"github.com/giantswarm/apiextensions/v3/pkg/apis/infrastructure/v1alpha3"
	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	capav1alpha3 "sigs.k8s.io/cluster-api-provider-aws/exp/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	bootstrap "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1alpha3"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client"
	capiv1alpha3exp "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/internal/key"
)

func WriteAWSTemplate(out io.Writer, config NodePoolCRsConfig) error {
	var err error

	if key.IsCAPAVersion(config.ReleaseVersion) {
		err = WriteCAPATemplate(out, config)
		if err != nil {
			return microerror.Mask(err)
		}
	} else {
		err = WriteGSAWSTemplate(out, config)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func WriteCAPATemplate(out io.Writer, config NodePoolCRsConfig) error {
	var err error

	if config.UseAlikeInstanceTypes {
		return microerror.Maskf(invalidFlagError, "--use-alike-instance-types setting is not available for release %v", config.ReleaseVersion)
	}

	var sshSSOPublicKey string
	{
		sshSSOPublicKey, err = key.SSHSSOPublicKey()
		if err != nil {
			return microerror.Mask(err)
		}
	}

	nodepoolTemplate, err := getCAPANodepoolTemplate(config)
	if err != nil {
		return err
	}

	data := struct {
		ProviderMachinePoolCR string
		MachinePoolCR         string
		KubeadmConfigCR       string
	}{}

	objects := nodepoolTemplate.Objs()
	for _, o := range objects {
		o.SetName(config.NodePoolID)
		o.SetLabels(map[string]string{
			label.ReleaseVersion:            config.ReleaseVersion,
			label.Cluster:                   config.ClusterName,
			label.MachinePool:               config.NodePoolID,
			capiv1alpha3.ClusterLabelName:   config.ClusterName,
			label.Organization:              config.Owner,
			"cluster.x-k8s.io/watch-filter": "capi",
		})
		switch o.GetKind() {
		case "AWSMachinePool":
			awsmachinepool, err := newAWSMachinePoolFromUnstructured(config, o)
			if err != nil {
				return microerror.Mask(err)
			}
			awsMachinePoolCRYaml, err := yaml.Marshal(awsmachinepool)
			if err != nil {
				return microerror.Mask(err)
			}
			data.ProviderMachinePoolCR = string(awsMachinePoolCRYaml)
		case "MachinePool":
			machinepool, err := newMachinePoolFromUnstructured(config, o)
			if err != nil {
				return microerror.Mask(err)
			}
			machinePoolCRYaml, err := yaml.Marshal(machinepool)
			if err != nil {
				return microerror.Mask(err)
			}
			data.MachinePoolCR = string(machinePoolCRYaml)
		case "KubeadmConfig":
			kubeadmConfig, err := newKubeadmConfigFromUnstructured(sshSSOPublicKey, key.NodeSSHDConfigEncoded(), o)
			if err != nil {
				return microerror.Mask(err)
			}
			kubeadmConfigCRYaml, err := yaml.Marshal(kubeadmConfig)
			if err != nil {
				return microerror.Mask(err)
			}
			data.KubeadmConfigCR = string(kubeadmConfigCRYaml)
		}
	}

	t := template.Must(template.New(config.FileName).Parse(key.MachinePoolAWSCRsTemplate))
	err = t.Execute(out, data)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func WriteGSAWSTemplate(out io.Writer, config NodePoolCRsConfig) error {
	var err error

	crsConfig := v1alpha3.NodePoolCRsConfig{
		AvailabilityZones:                   config.AvailabilityZones,
		AWSInstanceType:                     config.AWSInstanceType,
		ClusterID:                           config.ClusterName,
		Description:                         config.Description,
		MachineDeploymentID:                 config.NodePoolID,
		NodesMax:                            config.NodesMax,
		NodesMin:                            config.NodesMin,
		OnDemandBaseCapacity:                config.OnDemandBaseCapacity,
		OnDemandPercentageAboveBaseCapacity: config.OnDemandPercentageAboveBaseCapacity,
		Owner:                               config.Owner,
		UseAlikeInstanceTypes:               config.UseAlikeInstanceTypes,
	}

	crs, err := v1alpha3.NewNodePoolCRs(crsConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	if config.MachineDeploymentSubnet != "" {
		crs.AWSMachineDeployment.Annotations[annotation.AWSSubnetSize] = config.MachineDeploymentSubnet
	}

	if key.IsOrgNamespaceVersion(config.ReleaseVersion) {
		crs = moveCRsToOrgNamespace(crs, config.Owner)
	}

	mdCRYaml, err := yaml.Marshal(crs.MachineDeployment)
	if err != nil {
		return microerror.Mask(err)
	}

	awsMDCRYaml, err := yaml.Marshal(crs.AWSMachineDeployment)
	if err != nil {
		return microerror.Mask(err)
	}

	data := struct {
		AWSMachineDeploymentCR string
		MachineDeploymentCR    string
	}{
		AWSMachineDeploymentCR: string(awsMDCRYaml),
		MachineDeploymentCR:    string(mdCRYaml),
	}

	t := template.Must(template.New(config.FileName).Parse(key.MachineDeploymentCRsTemplate))
	err = t.Execute(out, data)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func getCAPANodepoolTemplate(config NodePoolCRsConfig) (client.Template, error) {
	var err error

	c, err := client.New("")
	if err != nil {
		return nil, err
	}

	templateOptions := client.GetClusterTemplateOptions{
		ClusterName:       config.ClusterName,
		TargetNamespace:   key.OrganizationNamespaceFromName(config.Owner),
		KubernetesVersion: "v1.19.9",
		ProviderRepositorySource: &client.ProviderRepositorySourceOptions{
			InfrastructureProvider: "aws:v0.6.8",
			Flavor:                 "machinepool",
		},
	}

	// Set all environment variables expected by the upstream client to empty strings. These values are defaulted later
	// Make sure that the values are reset.
	for _, envVar := range key.GetCAPAEnvVars() {
		if os.Getenv(envVar) != "" {
			prevEnv := os.Getenv(envVar)
			os.Setenv(envVar, "")
			defer os.Setenv(envVar, prevEnv)
		} else {
			os.Setenv(envVar, "")
			defer os.Unsetenv(envVar)
		}
	}

	if replicas := int64(config.NodesMin); replicas > 0 {
		templateOptions.WorkerMachineCount = &replicas
	}

	nodepoolTemplate, err := c.GetClusterTemplate(templateOptions)
	if err != nil {
		return nil, err
	}
	return nodepoolTemplate, nil
}

func newAWSMachinePoolFromUnstructured(config NodePoolCRsConfig, o unstructured.Unstructured) (*capav1alpha3.AWSMachinePool, error) {
	var awsmachinepool capav1alpha3.AWSMachinePool
	{
		err := runtime.DefaultUnstructuredConverter.
			FromUnstructured(o.Object, &awsmachinepool)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		awsmachinepool.Spec.AvailabilityZones = config.AvailabilityZones
		awsmachinepool.Spec.AWSLaunchTemplate.InstanceType = config.AWSInstanceType
		awsmachinepool.Spec.AWSLaunchTemplate.IamInstanceProfile = key.GetNodeInstanceProfile(config.NodePoolID, config.ClusterName)
		awsmachinepool.Spec.MinSize = int32(config.NodesMin)
		awsmachinepool.Spec.MaxSize = int32(config.NodesMax)
		onDemandBaseCapacity := int64(config.OnDemandBaseCapacity)
		awsmachinepool.Spec.Subnets = nil // for now we dont allow spec of subnets ID and  this needs to be ommited
		onDemandPercentageAboveBaseCapacity := int64(config.OnDemandPercentageAboveBaseCapacity)
		awsmachinepool.Spec.MixedInstancesPolicy = &capav1alpha3.MixedInstancesPolicy{
			InstancesDistribution: &capav1alpha3.InstancesDistribution{
				OnDemandBaseCapacity:                &onDemandBaseCapacity,
				OnDemandPercentageAboveBaseCapacity: &onDemandPercentageAboveBaseCapacity,
			},
		}
		if config.MachineDeploymentSubnet != "" {
			awsmachinepool.SetAnnotations(map[string]string{annotation.AWSSubnetSize: config.MachineDeploymentSubnet})
		}
	}
	return &awsmachinepool, nil
}

func newMachinePoolFromUnstructured(config NodePoolCRsConfig, o unstructured.Unstructured) (*capiv1alpha3exp.MachinePool, error) {
	var machinepool capiv1alpha3exp.MachinePool
	{
		err := runtime.DefaultUnstructuredConverter.
			FromUnstructured(o.Object, &machinepool)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		machinepool.SetAnnotations(map[string]string{
			annotation.MachinePoolName: config.Description,
			annotation.NodePoolMinSize: strconv.Itoa(config.NodesMin),
			annotation.NodePoolMaxSize: strconv.Itoa(config.NodesMax)})

		machinepool.Spec.Template.Spec.Bootstrap.ConfigRef.Name = config.NodePoolID
		machinepool.Spec.Template.Spec.InfrastructureRef.Name = config.NodePoolID

	}
	return &machinepool, nil
}

func newKubeadmConfigFromUnstructured(sshSSOPubKey string, sshdConfig string, o unstructured.Unstructured) (*bootstrap.KubeadmConfig, error) {
	var kubeadmConfig bootstrap.KubeadmConfig
	{
		groups := "sudo"
		shell := "/bin/bash"

		err := runtime.DefaultUnstructuredConverter.
			FromUnstructured(o.Object, &kubeadmConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		kubeadmConfig.Spec.Files = []bootstrap.File{
			{
				Owner:       "root",
				Permissions: "640",
				Path:        "/etc/ssh/sshd_config",
				Content:     sshdConfig,
				Encoding:    bootstrap.Base64,
			},
			{
				Owner:       "root",
				Permissions: "600",
				Path:        "/etc/ssh/trusted-user-ca-keys.pem",
				Content:     sshSSOPubKey,
				Encoding:    bootstrap.Base64,
			},
			{
				Owner:       "root",
				Permissions: "600",
				Path:        "/etc/sudoers.d/giantswarm",
				Content:     key.UbuntuSudoersConfigEncoded(),
				Encoding:    bootstrap.Base64,
			},
		}
		kubeadmConfig.Spec.PostKubeadmCommands = []string{
			"service ssh restart",
		}
		kubeadmConfig.Spec.Users = []bootstrap.User{
			{
				Name:   "giantswarm",
				Groups: &groups,
				Shell:  &shell,
			},
		}
	}

	return &kubeadmConfig, nil
}

func moveCRsToOrgNamespace(crs v1alpha3.NodePoolCRs, organization string) v1alpha3.NodePoolCRs {
	for _, cr := range []interface{}{crs.AWSMachineDeployment, crs.MachineDeployment} {
		cr = key.MoveNamespace(cr.(metav1.Object), key.OrganizationNamespaceFromName(organization))
	}
	crs.MachineDeployment.Spec.Template.Spec.InfrastructureRef.Namespace = key.OrganizationNamespaceFromName(organization)
	return crs
}
