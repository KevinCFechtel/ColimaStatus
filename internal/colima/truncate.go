package colima

import "unicode/utf8"

// maximumDetailLength bounds command output that ends up in a menu tooltip.
const maximumDetailLength = 600

// shortCommandOutput cuts overlong command output on a rune boundary. Cutting
// by byte would split a multi-byte character and render as replacement
// characters in the menu, which Colima's own output makes likely: it is
// localized and quotes user paths.
func shortCommandOutput(output string) string {
	return truncateRunes(output, maximumDetailLength)
}

func truncateRunes(value string, maximumLength int) string {
	if len(value) <= maximumLength {
		return value
	}
	cut := maximumLength
	for cut > 0 && !utf8.RuneStart(value[cut]) {
		cut--
	}
	return value[:cut] + "\u2026"
}
