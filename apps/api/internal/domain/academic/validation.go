package academic

import (
	"regexp"
	"time"
	"unicode/utf8"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
)

var (
	codePattern      = regexp.MustCompile(`^[a-z0-9-]{2,20}$`)
	shortNamePattern = regexp.MustCompile(`^[A-Z0-9]{2,20}$`)
	sectionPattern   = regexp.MustCompile(`^[A-Z]{1,2}$`)
)

func validateCreateMajor(code, name, shortName string) error {
	var fields []apperr.FieldError
	if !codePattern.MatchString(code) {
		fields = append(fields, apperr.Field(
			"code", "format",
			"Kode jurusan hanya boleh berisi huruf kecil, angka, dan tanda hubung (2–20 karakter).",
		))
	}
	if !validLength(name, 2, 100) {
		fields = append(fields, apperr.Field("name", "length", "Nama jurusan harus 2–100 karakter."))
	}
	if !shortNamePattern.MatchString(shortName) {
		fields = append(fields, apperr.Field(
			"short_name", "format",
			"Nama singkat jurusan harus 2–20 karakter huruf kapital atau angka.",
		))
	}
	return validationOrNil(fields)
}

func validateUpdateMajor(name, shortName *string) error {
	var fields []apperr.FieldError
	if name != nil && !validLength(*name, 2, 100) {
		fields = append(fields, apperr.Field("name", "length", "Nama jurusan harus 2–100 karakter."))
	}
	if shortName != nil && !shortNamePattern.MatchString(*shortName) {
		fields = append(fields, apperr.Field(
			"short_name", "format",
			"Nama singkat jurusan harus 2–20 karakter huruf kapital atau angka.",
		))
	}
	return validationOrNil(fields)
}

func validateYear(startYear int16, startsOn, endsOn time.Time) error {
	var fields []apperr.FieldError

	if startYear < 2000 || startYear > 2100 {
		fields = append(fields, apperr.Field("start_year", "range", "Tahun mulai harus antara 2000 dan 2100."))
	}

	startsMissing := startsOn.IsZero()
	endsMissing := endsOn.IsZero()
	if startsMissing {
		fields = append(fields, apperr.Field("starts_on", "required", "Tanggal mulai wajib diisi."))
	}
	if endsMissing {
		fields = append(fields, apperr.Field("ends_on", "required", "Tanggal selesai wajib diisi."))
	}
	if !startsMissing && startsOn.Year() != int(startYear) {
		fields = append(fields, apperr.Field(
			"starts_on", "invalid",
			"Tanggal mulai harus berada di tahun ajaran yang dipilih.",
		))
	}
	if !startsMissing && !endsMissing && !endsOn.After(startsOn) {
		fields = append(fields, apperr.Field(
			"ends_on", "invalid",
			"Tanggal selesai harus setelah tanggal mulai.",
		))
	}

	return validationOrNil(fields)
}

func validateClass(grade int16, section string) error {
	var fields []apperr.FieldError
	if grade != 10 && grade != 11 && grade != 12 {
		fields = append(fields, apperr.Field("grade", "invalid", "Tingkat kelas harus 10, 11, atau 12."))
	}
	if !sectionPattern.MatchString(section) {
		fields = append(fields, apperr.Field("section", "format", "Rombel harus 1–2 huruf kapital."))
	}
	return validationOrNil(fields)
}

func validationOrNil(fields []apperr.FieldError) error {
	if len(fields) == 0 {
		return nil
	}
	return apperr.Validation(fields...)
}

func validLength(value string, minLen, maxLen int) bool {
	n := utf8.RuneCountInString(value)
	return n >= minLen && n <= maxLen
}
