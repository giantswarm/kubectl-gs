package ocirepo

import (
	"bytes"
	"io"

	"github.com/giantswarm/microerror"
	"go.yaml.in/yaml/v4"
)

// SemverModifier makes an OCIRepository follow a semver range, so Flux
// pulls the newest chart version in the range on its own. The range
// replaces the tag or digest the reference held. A semverFilter is kept, as
// it narrows the tags the range applies to.
type SemverModifier struct {
	Range string
}

func (m SemverModifier) Execute(rawYaml []byte) ([]byte, error) {
	var doc yaml.Node
	decoder := yaml.NewDecoder(bytes.NewReader(rawYaml))
	err := decoder.Decode(&doc)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	// Only one document is written back, so a file holding more is refused
	// rather than losing the rest.
	var next yaml.Node
	if decoder.Decode(&next) != io.EOF {
		return nil, microerror.Maskf(invalidConfigError, "the file holds more than the OCIRepository")
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode || value(doc.Content[0], "kind") == nil || value(doc.Content[0], "kind").Value != "OCIRepository" {
		return nil, microerror.Maskf(invalidConfigError, "not an OCIRepository")
	}

	spec := value(doc.Content[0], "spec")
	if spec == nil || spec.Kind != yaml.MappingNode {
		return nil, microerror.Maskf(invalidConfigError, "the OCIRepository has no spec")
	}

	// The !!str tag quotes a range that looks like a number, like `1.2`,
	// which Flux would reject.
	semver := scalar(m.Range)
	semver.Tag = "!!str"
	ref := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{scalar("semver"), semver}}
	if old := value(spec, "ref"); old != nil {
		if filter := value(old, "semverFilter"); filter != nil {
			ref.Content = append(ref.Content, scalar("semverFilter"), filter)
		}
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
