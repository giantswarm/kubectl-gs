package migration

import (
	"bytes"

	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	"go.yaml.in/yaml/v4"
)

// readYAML returns the top level mapping of the file.
func readYAML(fs *afero.Afero, path string) (*yaml.Node, error) {
	data, err := fs.ReadFile(path)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return decodeMapping(data, path)
}

// decodeMapping returns the top level mapping of a YAML document.
func decodeMapping(data []byte, path string) (*yaml.Node, error) {
	var doc yaml.Node
	err := yaml.Unmarshal(data, &doc)
	if err != nil {
		return nil, microerror.Maskf(unexpectedFileError, "cannot parse %s: %s", path, err)
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, microerror.Maskf(unexpectedFileError, "%s is not a mapping", path)
	}

	return doc.Content[0], nil
}

func writeYAML(fs *afero.Afero, path string, node *yaml.Node) error {
	data, err := encodeYAML(node)
	if err != nil {
		return microerror.Mask(err)
	}

	return microerror.Mask(fs.WriteFile(path, data, 0644))
}

func encodeYAML(node *yaml.Node) ([]byte, error) {
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	err := encoder.Encode(node)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	err = encoder.Close()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return buf.Bytes(), nil
}

func mapValue(node *yaml.Node, key string) *yaml.Node {
	if node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}

	return nil
}

func deleteKey(node *yaml.Node, key string) {
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			node.Content = append(node.Content[:i], node.Content[i+2:]...)
			return
		}
	}
}

func scalar(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Value: value}
}
