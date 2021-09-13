package provider

import (
	"bytes"
	"context"
	"io"
	"text/template"

	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/cmd/template/cluster/provider/templates/azure"
	"github.com/giantswarm/kubectl-gs/internal/key"
)

func WriteCAPZTemplate(client k8sclient.Interface, out io.Writer, config ClusterCRsConfig) error {
	var err error
	ctx := context.Background()

	data := struct {
		Description       string
		KubernetesVersion string
		Name              string
		Namespace         string
		Owner             string
		Version           string
		VMSize            string
	}{
		Description:       config.Description,
		KubernetesVersion: "v1.19.9",
		Name:              config.Name,
		Namespace:         key.OrganizationNamespaceFromName(config.Owner),
		Owner:             config.Owner,
		Version:           config.ReleaseVersion,
		VMSize:            "Standard_D4s_v3",
	}

	for _, t := range azure.GetTemplates() {
		te := template.Must(template.New(config.FileName).Parse(t))
		buf := new(bytes.Buffer)
		// Template from our inputs.
		err = te.Execute(buf, data)
		if err != nil {
			return microerror.Mask(err)
		}
		// DryRun the CR against the management cluster.
		tYaml, err := runMutation(ctx, client, buf.Bytes())
		if err != nil {
			return microerror.Mask(err)
		}
		// Write the yaml to our file.
		_, err = out.Write(tYaml)
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
