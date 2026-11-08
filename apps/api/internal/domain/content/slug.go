package content

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Slugify converts a human title into a URL-safe slug: Unicode compatibility
// decomposition, combining marks dropped, lowercase, "&" read as "dan", any
// run of characters outside [a-z0-9] collapsed into one dash, then trimmed and
// cut at maxLen (maxLen <= 0 means no cut). An input with no usable characters
// becomes "".
func Slugify(s string, maxLen int) string {
	decomposed := norm.NFKD.String(s)
	decomposed = strings.ReplaceAll(decomposed, "&", " dan ")
	decomposed = strings.ToLower(decomposed)

	var b strings.Builder
	b.Grow(len(decomposed))
	lastDash := false
	for _, r := range decomposed {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && b.Len() > 0 {
			b.WriteByte('-')
			lastDash = true
		}
	}

	slug := strings.Trim(b.String(), "-")
	if maxLen > 0 && len(slug) > maxLen {
		slug = strings.TrimRight(slug[:maxLen], "-")
	}
	return slug
}
