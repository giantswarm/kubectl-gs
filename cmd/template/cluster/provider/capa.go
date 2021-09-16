package provider

import (
	"bytes"
	"context"
	"io"
	"text/template"

	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/cmd/template/cluster/provider/templates/aws"
	"github.com/giantswarm/kubectl-gs/internal/key"
)

func WriteCAPATemplate(ctx context.Context, client k8sclient.Interface, out io.Writer, config ClusterCRsConfig) error {
	var err error

	data := struct {
		Description       string
		KubernetesVersion string
		Name              string
		Namespace         string
		Owner             string
		ReleaseVersion    string
	}{
		Description:       config.Description,
		KubernetesVersion: "v1.19.9",
		Name:              config.Name,
		Namespace:         key.OrganizationNamespaceFromName(config.Owner),
		Owner:             config.Owner,
		ReleaseVersion:    config.ReleaseVersion,
	}

	for _, t := range aws.GetTemplates() {
		te := template.Must(template.New(config.FileName).Parse(t))
		var buf bytes.Buffer
		// Template from our inputs.
		err = te.Execute(&buf, data)
		if err != nil {
			return microerror.Mask(err)
		}
		// DryRun the CR against the management cluster.
		mutated, err := runMutation(ctx, client, buf.Bytes())
		if err != nil {
			return microerror.Mask(err)
		}
		// Write the yaml to our file.
		_, err = out.Write(mutated)
		if err != nil {
			return microerror.Mask(err)
		}
		// Add separators to make the entire file valid yaml and allow easy appending.
		_, err = out.Write([]byte("---\n"))
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
