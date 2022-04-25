package provider

import (
	"io"
	"text/template"

	"github.com/giantswarm/apiextensions/v6/pkg/apis/infrastructure/v1alpha3"

	"github.com/giantswarm/microerror"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/internal/key"
)

type NetworkPoolCRsConfig struct {
	CIDRBlock       string
	NetworkPoolName string
	Organization    string
	FileName        string
}

func WriteTemplate(out io.Writer, config NetworkPoolCRsConfig) error {
	var err error

	crsConfig := v1alpha3.NetworkPoolCRsConfig{
		CIDRBlock:     config.CIDRBlock,
		NetworkPoolID: config.NetworkPoolName,
		Owner:         config.Organization,
	}

	crs, err := v1alpha3.NewNetworkPoolCRs(crsConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	npCRYaml, err := yaml.Marshal(crs.NetworkPool)
	if err != nil {
		return microerror.Mask(err)
	}

	data := struct {
		NetworkPoolCR string
	}{
		NetworkPoolCR: string(npCRYaml),
	}

	t := template.Must(template.New(config.FileName).Parse(key.NetworkPoolCRsTemplate))
	err = t.Execute(out, data)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
