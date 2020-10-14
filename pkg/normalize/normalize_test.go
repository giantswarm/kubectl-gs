package normalize

import (
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_AsDNSLabelName(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "case 0: Uppercase input",
			input:    "Foobar",
			expected: "foobar",
		},
		{
			name:     "case 1: Uppercase input",
			input:    "FooBar",
			expected: "foobar",
		},
		{
			name:     "case 2: Uppercase input",
			input:    "FOOBAR",
			expected: "foobar",
		},
		{
			name:     "case 3: Lower input",
			input:    "foobar",
			expected: "foobar",
		},
		{
			name:     "case 4: Input with dot",
			input:    "foo.bar",
			expected: "foo-bar",
		},
		{
			name:     "case 5: Input with consequtive dots",
			input:    "foo..bar",
			expected: "foo-bar",
		},
		{
			name:     "case 6: Input with different punctuations",
			input:    "f\"\"oo!;ba_r",
			expected: "f-oo-ba-r",
		},
		{
			name:     "case 7: Input with dot in the beginning",
			input:    ".foobar",
			expected: "foobar",
		},
		{
			name:     "case 8: Input with dot in the end",
			input:    "foobar.",
			expected: "foobar",
		},
		{
			name:     "case 9: Input with dot all over the place",
			input:    ".foo-bar.",
			expected: "foo-bar",
		},
		{
			name:     "case 10: Input with dot all over the place",
			input:    "...foo-bar.",
			expected: "foo-bar",
		},
		{
			name:     "case 11: Input with dot all over the place",
			input:    "...foo-bar.....",
			expected: "foo-bar",
		},
		{
			name:     "case 12: Input without whitelisted characters",
			input:    ".*/!_*()[]%^\\.",
			expected: "",
		},
		{
			name:     "case 13: Way too long input",
			input:    "my.input.string.is.way.too.long.to.be.an.individual.dns.label.name.given.that.there.are.all.these.letters.and.other.punctuations.",
			expected: "my-input-string-is-way-too-long-to-be-an-individual-dns-label-n",
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Log(tc.name)

			output := AsDNSLabelName(tc.input)

			if !cmp.Equal(output, tc.expected) {
				t.Fatalf("\n\n%s\n", cmp.Diff(tc.expected, output))
			}
		})
	}
}
