package key

const ClusterCRsTemplate = `
{{ .ClusterCR -}}
---
{{ .AWSClusterCR -}}
{{ if .TemplateDefaultNodepool}}
---
{{ .MachineDeploymentCR -}}
---
{{ .AWSMachineDeploymentCR -}}
{{ end }}
`

const MachineDeploymentCRsTemplate = `
{{ .MachineDeploymentCR -}}
---
{{ .AWSMachineDeploymentCR -}}
`
