package key

const AppCRTemplate = `
{{- if .UserConfigConfigMap -}}
---
{{ .UserConfigConfigMap -}}
{{- end -}}
{{- if .UserConfigSecret -}}
---
{{ .UserConfigSecret -}}
{{- end -}}
---
{{ .AppCR -}}
`

// ClusterFluxTemplate renders the user-values ConfigMap alongside the
// OCIRepository and HelmRelease that deploy a release-<provider> chart via
// Flux, in place of the App CR used by AppCRTemplate.
const ClusterFluxTemplate = `
{{- if .UserConfigConfigMap -}}
---
{{ .UserConfigConfigMap -}}
{{- end -}}
---
{{ .OCIRepository -}}
---
{{ .HelmRelease -}}
`

const CatalogCRTemplate = `
{{- .ConfigMap -}}
---
{{ .Secret -}}
---
{{ .CatalogCR -}}
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
---
{{ .BastionBootstrapSecret -}}
---
{{ .BastionMachineDeploymentCR -}}
---
{{ .BastionAWSMachineTemplateCR -}}
`

const ClusterEKSCRsTemplate = `
{{- .ClusterCR -}}
---
{{ .AWSManagedControlPlaneCR -}}
---
{{ .AWSClusterRoleIdentityCR -}}
`

const MachinePoolAWSCRsTemplate = `
{{- .ProviderMachinePoolCR -}}
---
{{ .MachinePoolCR -}}
---
{{ .KubeadmConfigCR -}}
`

const MachinePoolEKSCRsTemplate = `
{{- .ManagedMachinePoolCR -}}
---
{{ .MachinePoolCR -}}
`

const MachinePoolAzureCRsTemplate = `
{{- .ProviderMachinePoolCR -}}
---
{{ .MachinePoolCR -}}
---
{{ .SparkCR -}}
`
