package slugify

import (
	"regexp"
	"strings"

	"golang.org/x/text/unicode/norm"
)

var tagRegex = regexp.MustCompile("<[^>]*>")

// Slugify is a partial reimplementation of pelican.utils.slugify as used by
// the pyvideo project.
//
// https://github.com/getpelican/pelican/blob/b27153fe9b9362a3f7f87b90225c26975ba18f1d/pelican/utils.py#L266
func Slugify(data string) string {
	output := stripTags(data)
	output = toNormalizedLowercase(output)
	output = replaceSpecialCharacters(output)
	return output
}

func stripTags(input string) string {
	return tagRegex.ReplaceAllString(input, "")
}

func toNormalizedLowercase(input string) string {
	normalized := norm.NFKD.String(input)
	return strings.ToLower(normalized)
}

func replaceSpecialCharacters(input string) string {
	output := input
	output = strings.ReplaceAll(output, "©", "c")
	output = regexp.MustCompile("[^\\w\\s-]").ReplaceAllString(output, "")
	output = regexp.MustCompile("[-\\s]+").ReplaceAllString(output, "-")
	return output
}
