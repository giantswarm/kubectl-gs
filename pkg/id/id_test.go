package id

import (
	"regexp"
	"strconv"
	"testing"
	"unicode"
)

func TestGenerate(t *testing.T) {
	attempts := 1000

	validCharsRegexp := regexp.MustCompile("^[02-9a-km-z]")
	lettersOnlyRegexp := regexp.MustCompile("^[a-z]+$")
	digitsOnlyRegexp := regexp.MustCompile("^[0-9]+$")

	for i := 0; i < attempts; i++ {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			id := Generate()
			idChars := []rune(id)

			if !validCharsRegexp.MatchString(id) {
				t.Fatalf("generated id contains invalid characters: %q", id)
			}

			if !unicode.IsLetter(idChars[0]) {
				t.Fatalf("generated id does not start with a letter: %q", id)
			}

			if lettersOnlyRegexp.MatchString(id) {
				t.Fatalf("generated id only contains letters: %q", id)
			}

			if digitsOnlyRegexp.MatchString(id) {
				t.Fatalf("generated id only contains digits: %q", id)
			}

			if len(idChars) > IDLength {
				t.Fatalf("generated id is too long: %d > %d", len(idChars), IDLength)
			}
		})
	}
}
