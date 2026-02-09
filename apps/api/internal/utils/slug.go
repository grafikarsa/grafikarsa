package utils

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

var (
	// slugRegexReplace matches characters that should be replaced
	nonAlphanumericRegex = regexp.MustCompile(`[^a-z0-9-]+`)
	// multiHyphenRegex matches multiple consecutive hyphens
	multiHyphenRegex = regexp.MustCompile(`-+`)
)

// GenerateSlug creates a URL-friendly slug from a title
func GenerateSlug(title string) string {
	// Normalize Unicode characters
	slug := norm.NFD.String(title)

	// Convert to lowercase
	slug = strings.ToLower(slug)

	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")

	// Remove non-ASCII characters (diacritics, etc.)
	slug = removeNonASCII(slug)

	// Replace non-alphanumeric characters (except hyphens) with empty string
	slug = nonAlphanumericRegex.ReplaceAllString(slug, "")

	// Replace multiple consecutive hyphens with a single hyphen
	slug = multiHyphenRegex.ReplaceAllString(slug, "-")

	// Trim hyphens from the beginning and end
	slug = strings.Trim(slug, "-")

	// Limit length to 200 characters (matching database constraint)
	if len(slug) > 200 {
		slug = slug[:200]
		// Ensure we don't cut in the middle of a word
		if idx := strings.LastIndex(slug, "-"); idx > 150 {
			slug = slug[:idx]
		}
	}

	// Default slug if empty
	if slug == "" {
		slug = "portfolio"
	}

	return slug
}

// removeNonASCII removes non-ASCII characters from a string
func removeNonASCII(s string) string {
	var result strings.Builder
	for _, r := range s {
		if r <= unicode.MaxASCII {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// UniqueSlug appends a suffix to make a slug unique
func UniqueSlug(baseSlug, suffix string) string {
	if suffix == "" {
		return baseSlug
	}

	slug := baseSlug + "-" + suffix

	// Ensure total length doesn't exceed 200
	if len(slug) > 200 {
		// Truncate base slug to make room for suffix
		maxBaseLen := 200 - len(suffix) - 1
		if maxBaseLen > 0 {
			slug = baseSlug[:maxBaseLen] + "-" + suffix
		} else {
			slug = suffix
		}
	}

	return slug
}
