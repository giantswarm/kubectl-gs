package key

const ClusterCRsTemplate = `---
{{ .ClusterCR }}
---
{{ .AWSClusterCR }}
`
