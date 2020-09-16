package key

const AppCRTemplate = `
{{- .UserConfigConfigMapCR -}}
---
{{ .UserConfigSecretCR -}}
---
{{ .AppCR -}}
`

const AppCatalogCRTemplate = `
{{- .ConfigmapCR -}}
---
{{ .SecretCR -}}
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

const MachineDeploymentCRsTemplate = `
{{- .MachineDeploymentCR -}}
---
{{ .AWSMachineDeploymentCR -}}
`

const MachinePoolAzureCRsTemplate = `
{{- .MachinePoolCR -}}
---
{{ .ProviderMachinePoolCR -}}
`
