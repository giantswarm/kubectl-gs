package template

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestGetSuccessHTMLTemplateReader(t *testing.T) {
	reader, err := GetSuccessHTMLTemplateReader()
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	out := new(bytes.Buffer)
	_, err = io.Copy(out, reader)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	if !strings.Contains(out.String(), "SSO authentication complete") {
		t.Fatalf("file contents do not contain desired substring")
	}
}

func TestGetFailedHTMLTemplateReader(t *testing.T) {
	reader, err := GetFailedHTMLTemplateReader()
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	out := new(bytes.Buffer)
	_, err = io.Copy(out, reader)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	if !strings.Contains(out.String(), "SSO authentication failed") {
		t.Fatalf("file contents do not contain desired substring")
	}
}
