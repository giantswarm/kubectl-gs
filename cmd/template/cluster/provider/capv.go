package provider

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/cmd/template/cluster/provider/templates/vsphere"
	"github.com/giantswarm/kubectl-gs/internal/key"
)

func WriteCAPVTemplate(ctx context.Context, client k8sclient.Interface, out io.Writer, config ClusterCRsConfig) error {
	var err error

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return microerror.Mask(err)
	}

	idRsaPubBytes, err := os.ReadFile(filepath.Join(homeDir, ".ssh/id_rsa.pub"))
	if err != nil {
		return microerror.Mask(err)
	}
	idRsaPubBytes = bytes.TrimSpace(idRsaPubBytes)

	data := struct {
		Description       string
		KubernetesVersion string
		Name              string
		Namespace         string
		Organization      string
		ReleaseVersion    string
		SSHPublicKey      string
	}{
		Description:       config.Description,
		KubernetesVersion: "v1.20.1",
		Name:              config.Name,
		Namespace:         key.OrganizationNamespaceFromName(config.Organization),
		Organization:      config.Organization,
		ReleaseVersion:    config.ReleaseVersion,
		SSHPublicKey:      string(idRsaPubBytes),
	}

	var templates []templateConfig
	for _, t := range vsphere.GetTemplates() {
		templates = append(templates, templateConfig(t))
	}

	err = runMutation(ctx, client, data, templates, out)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
