package slugify_test

import (
	"testing"

	"github.com/zerok/pyvideosearch/slugify"
)

func TestSlugify(t *testing.T) {
	testcases := []struct {
		input    string
		expected string
	}{
		{
			input:    "Carl Meyer about Django @ Instagram at Django: Under The Hood 2016",
			expected: "carl-meyer-about-django-instagram-at-django-under-the-hood-2016",
		},
	}

	for _, test := range testcases {
		output := slugify.Slugify(test.input)
		if output != test.expected {
			t.Fatalf("\nExpected: `%s`\nActual:   `%s`", test.expected, output)
		}
	}
}
