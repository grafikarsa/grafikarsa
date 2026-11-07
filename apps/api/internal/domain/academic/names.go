package academic

import (
	"fmt"
	"strings"
)

// gradeRoman returns the Roman numeral used for a grade level.
func gradeRoman(grade int16) string {
	switch grade {
	case 10:
		return "X"
	case 11:
		return "XI"
	case 12:
		return "XII"
	default:
		return fmt.Sprintf("%d", grade)
	}
}

// ClassName builds the display name of a class, e.g. "XI RPL B".
func ClassName(grade int16, majorShortName, section string) string {
	return fmt.Sprintf("%s %s %s",
		gradeRoman(grade),
		normalizeShortName(majorShortName),
		normalizeSection(section),
	)
}

// ClassSlug builds the URL slug of a class, e.g. "xi-rpl-b".
func ClassSlug(grade int16, majorShortName, section string) string {
	raw := strings.ToLower(strings.Join([]string{
		gradeRoman(grade),
		normalizeShortName(majorShortName),
		normalizeSection(section),
	}, " "))
	return strings.Join(strings.Fields(raw), "-")
}

// YearLabel builds the human label of an academic year, e.g. "2026/2027".
func YearLabel(startYear int16) string {
	return fmt.Sprintf("%d/%d", startYear, startYear+1)
}

// YearSlug builds the URL slug of an academic year, e.g. "2026-2027".
func YearSlug(startYear int16) string {
	return fmt.Sprintf("%d-%d", startYear, startYear+1)
}
