package ocirepo

import (
	"bytes"

	"github.com/giantswarm/microerror"
	"go.yaml.in/yaml/v3"
)

// SemverModifier makes an OCIRepository follow a semver range, so Flux
// pulls the newest chart version in the range on its own. The range
// replaces the tag or digest the reference held.
type SemverModifier struct {
	Range string
}

func (m SemverModifier) Execute(rawYaml []byte) ([]byte, error) {
	var doc yaml.Node
	err := yaml.Unmarshal(rawYaml, &doc)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode || value(doc.Content[0], "kind") == nil || value(doc.Content[0], "kind").Value != "OCIRepository" {
		return nil, microerror.Maskf(invalidConfigError, "not an OCIRepository")
	}

	spec := value(doc.Content[0], "spec")
	if spec == nil || spec.Kind != yaml.MappingNode {
		return nil, microerror.Maskf(invalidConfigError, "the OCIRepository has no spec")
	}

	ref := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{scalar("semver"), scalar(m.Range)}}
	if old := value(spec, "ref"); old != nil {
		*old = *ref
	} else {
		spec.Content = append(spec.Content, scalar("ref"), ref)
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	err = encoder.Encode(&doc)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	err = encoder.Close()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return buf.Bytes(), nil
}

func value(node *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func scalar(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Value: value}
}
