package migration

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"strings"

	"github.com/giantswarm/microerror"
	"go.yaml.in/yaml/v4"
)

// document is one YAML document of a file, as its bytes, so that a file can
// be put back together byte for byte.
type document struct {
	// start is the `---` line that starts the document, empty when it has
	// none.
	start []byte
	// body is what follows it, up to the next document marker.
	body []byte
	// end is the `...` line that ends the document, empty when it has none.
	end []byte
	// node is its content, nil for a document of only comments.
	node *yaml.Node
}

func (d document) bytes() []byte {
	return append(append(append([]byte{}, d.start...), d.body...), d.end...)
}

// splitDocuments splits a YAML stream into its documents. The YAML decoder
// says what the documents are; their bytes are found from the markers that
// start and end them, `---` and `...` at the start of a line, which YAML
// forbids anywhere else. When the two don't agree, or the documents don't
// add up to the file, it fails rather than guess.
func splitDocuments(data []byte, path string) ([]document, error) {
	documents := scanDocuments(data)

	var joined []byte
	for i := range documents {
		joined = append(joined, documents[i].bytes()...)

		node, err := decodeDocument(documents[i].bytes(), path)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		documents[i].node = node
	}

	decoded, err := decodeStream(data, path)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if !bytes.Equal(joined, data) || !sameDocuments(documents, decoded) {
		return nil, microerror.Maskf(unexpectedFileError, "can't tell the YAML documents of %s apart exactly, change it by hand", path)
	}

	return documents, nil
}

// scanDocuments splits the stream at its document markers.
func scanDocuments(data []byte) []document {
	var documents []document
	current := document{}
	push := func() {
		if len(current.start)+len(current.body)+len(current.end) > 0 {
			documents = append(documents, current)
		}
		current = document{}
	}

	for _, line := range bytes.SplitAfter(data, []byte("\n")) {
		switch {
		case len(line) == 0:
			continue
		case isMarker(line, "---"):
			push()
			current.start = line
		case isMarker(line, "..."):
			current.end = line
			push()
		default:
			current.body = append(current.body, line...)
		}
	}
	push()

	return documents
}

// isMarker reports whether the line is the document marker: the marker
// alone, or followed by a space or a tab.
func isMarker(line []byte, marker string) bool {
	text := strings.TrimRight(string(line), "\r\n")
	if !strings.HasPrefix(text, marker) {
		return false
	}
	rest := text[len(marker):]

	return rest == "" || rest[0] == ' ' || rest[0] == '\t'
}

// decodeDocument returns the content of a single document, nil when it has
// none.
func decodeDocument(data []byte, path string) (*yaml.Node, error) {
	nodes, err := decodeStream(data, path)
	if err != nil || len(nodes) == 0 {
		return nil, microerror.Mask(err)
	}
	if len(nodes) > 1 {
		return nil, microerror.Maskf(unexpectedFileError, "can't tell the YAML documents of %s apart exactly, change it by hand", path)
	}

	return nodes[0], nil
}

// decodeStream returns the content of every document of the stream that
// has any.
func decodeStream(data []byte, path string) ([]*yaml.Node, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(data))

	var nodes []*yaml.Node
	for {
		var doc yaml.Node
		err := decoder.Decode(&doc)
		if errors.Is(err, io.EOF) {
			return nodes, nil
		} else if err != nil {
			return nil, microerror.Maskf(unexpectedFileError, "cannot parse %s: %s", path, err)
		}
		if len(doc.Content) > 0 {
			nodes = append(nodes, doc.Content[0])
		}
	}
}

// sameDocuments reports whether the documents hold the same content as the
// decoder found in the stream, in the same order.
func sameDocuments(documents []document, decoded []*yaml.Node) bool {
	var nodes []*yaml.Node
	for _, d := range documents {
		if d.node != nil {
			nodes = append(nodes, d.node)
		}
	}
	if len(nodes) != len(decoded) {
		return false
	}

	for i := range nodes {
		var a, b any
		if nodes[i].Decode(&a) != nil || decoded[i].Decode(&b) != nil || !reflect.DeepEqual(a, b) {
			return false
		}
	}

	return true
}

// rewrite returns the document with its content encoded anew, keeping the
// markers around it and its line endings. It fails for a document whose
// start marker holds content, which re-encoding would duplicate.
func (d document) rewrite(path string) ([]byte, error) {
	if !isMarker(d.start, "---") && len(d.start) > 0 || hasContent(d.start) {
		return nil, microerror.Maskf(unexpectedFileError, "a document of %s starts on its `---` line, change it by hand", path)
	}

	encoded, err := encodeYAML(d.node)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if bytes.Contains(d.body, []byte("\r\n")) {
		encoded = bytes.ReplaceAll(encoded, []byte("\n"), []byte("\r\n"))
	}

	return append(append(append([]byte{}, d.start...), encoded...), d.end...), nil
}

// hasContent reports whether a `---` line holds more than a comment.
func hasContent(start []byte) bool {
	rest := strings.TrimSpace(strings.TrimPrefix(strings.TrimRight(string(start), "\r\n"), "---"))

	return rest != "" && !strings.HasPrefix(rest, "#")
}
