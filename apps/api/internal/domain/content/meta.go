package content

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
)

var portfolioContexts = map[string]struct{}{
	"assignment":  {},
	"internship":  {},
	"competition": {},
	"ukk":         {},
	"personal":    {},
	"other":       {},
}

var monthPattern = regexp.MustCompile(`^[0-9]{4}-(0[1-9]|1[0-2])$`)

const (
	maxPortfolioLinks = 5
	maxLinkLabelRunes = 40
	maxLinkURLRunes   = 500
)

// validateMeta parses the type-specific meta object, dropping unknown keys.
// For blog it also returns the requested series id so the caller can check its
// ownership against the content owner.
func validateMeta(contentType string, raw map[string]any) (map[string]any, *uuid.UUID, []apperr.FieldError) {
	switch contentType {
	case TypePortfolio:
		meta, errs := validatePortfolioMeta(raw)
		return meta, nil, errs
	case TypeBlog:
		meta, seriesID, errs := validateBlogMeta(raw)
		return meta, seriesID, errs
	default:
		return map[string]any{}, nil, []apperr.FieldError{
			apperr.Field("type", "unknown_type", "Tipe konten tidak dikenal."),
		}
	}
}

func validatePortfolioMeta(raw map[string]any) (map[string]any, []apperr.FieldError) {
	var errs []apperr.FieldError
	meta := map[string]any{}

	context := "personal"
	if value, ok := raw["context"]; ok && value != nil {
		text, isString := value.(string)
		if !isString {
			errs = append(errs, apperr.Field("meta.context", "invalid_type", "Konteks karya tidak valid."))
		} else if _, allowed := portfolioContexts[text]; !allowed {
			errs = append(errs, apperr.Field("meta.context", "invalid_value", "Konteks karya tidak dikenal."))
		} else {
			context = text
		}
	}
	meta["context"] = context

	if value, ok := raw["period"]; ok && value != nil {
		period, isObject := value.(map[string]any)
		if !isObject {
			errs = append(errs, apperr.Field("meta.period", "invalid_type", "Periode karya tidak valid."))
		} else {
			canonical := map[string]any{}
			start, startOK := metaMonth(period["start"], "meta.period.start", &errs)
			end, endOK := metaMonth(period["end"], "meta.period.end", &errs)
			if startOK {
				canonical["start"] = start
			}
			if endOK {
				canonical["end"] = end
			}
			if startOK && endOK && end < start {
				errs = append(errs, apperr.Field("meta.period.end", "invalid_value", "Akhir periode tidak boleh sebelum awal periode."))
			}
			if len(canonical) > 0 {
				meta["period"] = canonical
			}
		}
	}

	links := []any{}
	if value, ok := raw["links"]; ok && value != nil {
		items, isSlice := value.([]any)
		if !isSlice {
			errs = append(errs, apperr.Field("meta.links", "invalid_type", "Daftar tautan tidak valid."))
		} else if len(items) > maxPortfolioLinks {
			errs = append(errs, apperr.Field("meta.links", "too_many", "Tautan maksimal 5."))
		} else {
			for i, item := range items {
				link, isObject := item.(map[string]any)
				if !isObject {
					errs = append(errs, apperr.Field(linkField(i, "label"), "invalid_type", "Tautan tidak valid."))
					continue
				}
				label, _ := link["label"].(string)
				label = strings.TrimSpace(label)
				if label == "" || utf8.RuneCountInString(label) > maxLinkLabelRunes {
					errs = append(errs, apperr.Field(linkField(i, "label"), "invalid_length", "Label tautan harus 1–40 karakter."))
				}
				rawURL, _ := link["url"].(string)
				rawURL = strings.TrimSpace(rawURL)
				if !validHTTPSURL(rawURL) {
					errs = append(errs, apperr.Field(linkField(i, "url"), "invalid_url", "URL harus diawali https:// dan maksimal 500 karakter."))
				}
				links = append(links, map[string]any{"label": label, "url": rawURL})
			}
		}
	}
	meta["links"] = links

	return meta, errs
}

func validateBlogMeta(raw map[string]any) (map[string]any, *uuid.UUID, []apperr.FieldError) {
	var errs []apperr.FieldError
	meta := map[string]any{}

	var seriesID *uuid.UUID
	if value, ok := raw["seriesId"]; ok && value != nil {
		text, isString := value.(string)
		if !isString {
			errs = append(errs, apperr.Field("meta.seriesId", "invalid_type", "ID seri tidak valid."))
		} else {
			parsed, err := uuid.Parse(text)
			if err != nil || parsed == uuid.Nil {
				errs = append(errs, apperr.Field("meta.seriesId", "invalid_value", "ID seri tidak valid."))
			} else {
				meta["seriesId"] = parsed.String()
				seriesID = &parsed
			}
		}
	}

	return meta, seriesID, errs
}

func metaMonth(value any, field string, errs *[]apperr.FieldError) (string, bool) {
	if value == nil {
		return "", false
	}
	text, ok := value.(string)
	if !ok || !monthPattern.MatchString(text) {
		*errs = append(*errs, apperr.Field(field, "invalid_value", "Format bulan harus YYYY-MM."))
		return "", false
	}
	return text, true
}

func linkField(index int, name string) string {
	return "meta.links[" + strconv.Itoa(index) + "]." + name
}

func validHTTPSURL(raw string) bool {
	if raw == "" || len(raw) > maxLinkURLRunes {
		return false
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return parsed.Scheme == "https" && parsed.Host != ""
}
