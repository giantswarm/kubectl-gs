package util

import (
	"github.com/giantswarm/apiextensions/v6/pkg/apis/infrastructure/v1alpha3"
	"github.com/giantswarm/apiextensions/v6/pkg/id"
	"github.com/giantswarm/k8smetadata/pkg/annotation"
	"github.com/giantswarm/k8smetadata/pkg/label"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen=false

type NetworkPoolCRsConfig struct {
	CIDRBlock     string
	NetworkPoolID string
	Namespace     string
	Owner         string
}

// +k8s:deepcopy-gen=false

type NetworkPoolCRs struct {
	NetworkPool *v1alpha3.NetworkPool
}

func NewNetworkPoolCRs(config NetworkPoolCRsConfig) (NetworkPoolCRs, error) {
	// Default some essentials in case certain information are not given. E.g.
	// the Tenant NetworkPoolID may be provided by the user.
	{
		if config.NetworkPoolID == "" {
			config.NetworkPoolID = id.Generate()
		}
		if config.Namespace == "" {
			config.Namespace = metav1.NamespaceDefault
		}
	}

	networkPoolCR := newNetworkPoolCR(config)

	crs := NetworkPoolCRs{
		NetworkPool: networkPoolCR,
	}

	return crs, nil
}

func newNetworkPoolCR(c NetworkPoolCRsConfig) *v1alpha3.NetworkPool {
	return &v1alpha3.NetworkPool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.NetworkPoolID,
			Namespace: c.Namespace,
			Annotations: map[string]string{
				annotation.Docs: "https://docs.giantswarm.io/ui-api/management-api/crd/networkpools.infrastructure.giantswarm.io/",
			},
			Labels: map[string]string{
				label.Organization: c.Owner,
			},
		},
		Spec: v1alpha3.NetworkPoolSpec{
			CIDRBlock: c.CIDRBlock,
		},
	}
}
