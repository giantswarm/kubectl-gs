package modifier

import (
	"errors"
	"testing"
)

type suffix string

func (s suffix) Execute(data []byte) ([]byte, error) {
	if s == "" {
		return nil, errors.New("failed")
	}
	return append(data, s...), nil
}

func Test_Chain(t *testing.T) {
	out, err := Chain{suffix("a"), suffix("b")}.Execute([]byte("-"))
	if err != nil || string(out) != "-ab" {
		t.Fatalf("expected -ab, got %q, %v", out, err)
	}

	_, err = Chain{suffix("a"), suffix("")}.Execute([]byte("-"))
	if err == nil {
		t.Fatalf("expected the error of the failing modifier")
	}
}
