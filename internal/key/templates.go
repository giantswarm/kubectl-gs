package key

const AppCRTemplate = `
{{ .UserConfigConfigMapCR -}}
---
{{ .UserConfigSecretCR -}}
---
{{ .AppCR -}}
`

const AppCatalogCRTemplate = `
{{ .ConfigmapCR -}}
---
{{ .SecretCR -}}
---
{{ .AppCatalogCR -}}
`

const ClusterCRsTemplate = `
{{ .ClusterCR -}}
---
{{ .AWSClusterCR -}}
---
{{ .G8sControlPlaneCR -}}
---
{{ .AWSControlPlaneCR -}}
`

const MachineDeploymentCRsTemplate = `
{{ .MachineDeploymentCR -}}
---
{{ .AWSMachineDeploymentCR -}}
`
