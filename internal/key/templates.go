package key

const AppCRTemplate = `
{{- .UserConfigConfigMap -}}
---
{{ .UserConfigSecret -}}
---
{{ .AppCR -}}
`

const AppCatalogCRTemplate = `
{{- .ConfigMap -}}
---
{{ .Secret -}}
---
{{ .AppCatalogCR -}}
`

const ClusterAWSCRsTemplate = `
{{- .ClusterCR -}}
---
{{ .AWSClusterCR -}}
---
{{ .G8sControlPlaneCR -}}
---
{{ .AWSControlPlaneCR -}}
`

const ClusterAzureCRsTemplate = `
{{- .ProviderClusterCR -}}
---
{{ .ClusterCR -}}
---
{{ .MasterMachineCR -}}
`

const ClusterCAPACRsTemplate = `
{{- .ClusterCR -}}
---
{{ .AWSClusterCR -}}
---
{{ .KubeadmControlPlaneCR -}}
---
{{ .AWSMachineTemplateCR -}}
---
{{ .AWSClusterRoleIdentityCR -}}
`

const MachineDeploymentCRsTemplate = `
{{- .MachineDeploymentCR -}}
---
{{ .AWSMachineDeploymentCR -}}
`

const NetworkPoolCRsTemplate = `
{{- .NetworkPoolCR -}}
`

const MachinePoolAWSCRsTemplate = `
{{- .ProviderMachinePoolCR -}}
---
{{ .MachinePoolCR -}}
---
{{ .KubeadmConfigCR -}}
`

const MachinePoolAzureCRsTemplate = `
{{- .ProviderMachinePoolCR -}}
---
{{ .MachinePoolCR -}}
---
{{ .SparkCR -}}
`
