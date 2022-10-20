package provider

import (
	"io"
	"text/template"

	"github.com/giantswarm/microerror"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/v2/cmd/template/cluster/util"
	"github.com/giantswarm/kubectl-gs/v2/internal/key"
)

type NetworkPoolCRsConfig struct {
	CIDRBlock       string
	NetworkPoolName string
	Organization    string
	FileName        string
}

func WriteTemplate(out io.Writer, config NetworkPoolCRsConfig) error {
	var err error

	crsConfig := util.NetworkPoolCRsConfig{
		CIDRBlock:     config.CIDRBlock,
		NetworkPoolID: config.NetworkPoolName,
		Owner:         config.Organization,
	}

	crs, err := util.NewNetworkPoolCRs(crsConfig)
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
