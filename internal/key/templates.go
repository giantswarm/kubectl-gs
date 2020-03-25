package key

const AppCatalogCRTemplate = `
{{ .AppCatalogCR -}}
`

const ClusterCRsTemplate = `
{{ .ClusterCR -}}
---
{{ .AWSClusterCR -}}
`
