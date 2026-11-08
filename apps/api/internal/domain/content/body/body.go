// Package body validates, normalizes, and derives metadata from content bodies
// (portfolio blocks and blog ProseMirror documents).
//
// The package is pure: it has no database or HTTP dependencies. Normalize is
// used on every autosave and only enforces structural rules, while
// ValidateForSubmit adds the completeness rules required before moderation.
package body

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
)

// ContentType identifies the body format owned by a content type.
type ContentType string

const (
	// Portfolio bodies are block lists ({"blocks": [...]}).
	Portfolio ContentType = "portfolio"
	// Blog bodies are ProseMirror documents ({"type":"doc","content":[...]}).
	Blog ContentType = "blog"

	// TypePortfolio and TypeBlog are aliases kept for call sites that prefer a
	// Type prefix.
	TypePortfolio = Portfolio
	TypeBlog      = Blog
)

// Limits enforced by the package.
const (
	// MaxBodyBytes is the largest accepted raw body (2 MB).
	MaxBodyBytes = 2 << 20

	maxPortfolioBlocks = 60
	maxPortfolioMedia  = 50
	maxGalleryItems    = 20
	minGalleryItems    = 2
	maxAltRunes        = 200
	maxCaptionRunes    = 300
	maxTextBlockRunes  = 5000
	maxCodeRunes       = 10000
	maxTableColumns    = 10
	maxTableRows       = 50
	maxTableCellRunes  = 200
	maxLinkLabelRunes  = 60
	maxFileTitleRunes  = 120
	maxBlogWords       = 20000
	minBlogWords       = 150
	maxBlogImages      = 30
	maxBlogEmbeds      = 15
	maxListDepth       = 3
	maxExcerptRunes    = 300
	wordsPerMinute     = 200
)

// Heading is one entry of a blog table of contents.
type Heading struct {
	ID    string
	Level int
	Text  string
}

// Result is the normalized body plus derived values.
type Result struct {
	Body           json.RawMessage
	MediaIDs       []uuid.UUID
	PlainText      string
	WordCount      int
	ReadingMinutes int
	Headings       []Heading
	AutoExcerpt    string
}

// Embed is a recognized, allowlisted embed URL.
type Embed struct {
	Provider string
	URL      string
	VideoID  string
}

// Normalize validates the structure of raw and returns its canonical form.
// It is safe to call on every autosave: incomplete drafts (for example an image
// block whose media is still uploading) are accepted.
func Normalize(t ContentType, raw json.RawMessage) (Result, []apperr.FieldError) {
	return normalize(t, raw, false)
}

// ValidateForSubmit runs Normalize and then enforces the completeness rules
// required before a content can be sent to moderation.
func ValidateForSubmit(t ContentType, raw json.RawMessage) (Result, []apperr.FieldError) {
	return normalize(t, raw, true)
}

func normalize(t ContentType, raw json.RawMessage, submit bool) (Result, []apperr.FieldError) {
	if t != Portfolio && t != Blog {
		return Result{}, []apperr.FieldError{{
			Field:   "type",
			Code:    "unknown_type",
			Message: "Tipe konten tidak dikenal.",
		}}
	}
	if len(raw) > MaxBodyBytes {
		return Result{}, []apperr.FieldError{{
			Field:   "body",
			Code:    "too_large",
			Message: "Body konten terlalu besar (maksimal 2 MB).",
		}}
	}

	root, err := decodeJSON(raw)
	if err != nil {
		return Result{}, []apperr.FieldError{{
			Field:   "body",
			Code:    "invalid_json",
			Message: "Body konten bukan JSON yang valid.",
		}}
	}

	v := &validator{
		submit: submit,
		media:  make(map[uuid.UUID]struct{}),
	}

	var canonical map[string]any
	switch t {
	case Portfolio:
		canonical = v.portfolio(root)
	case Blog:
		canonical = v.blog(root)
	}

	plain := v.plainText()
	words := wordCount(plain)
	res := Result{
		MediaIDs:       v.sortedMediaIDs(),
		PlainText:      plain,
		Headings:       v.headings,
		WordCount:      words,
		ReadingMinutes: readingMinutes(t, words),
		AutoExcerpt:    autoExcerpt(plain),
	}
	if canonical != nil {
		if encoded, err := marshalCanonical(canonical); err == nil {
			res.Body = encoded
		} else {
			v.add("body", "", "invalid_json", "Body konten tidak dapat dinormalisasi.")
		}
	}

	if submit && t == Blog {
		if res.WordCount < minBlogWords {
			v.add("body", "", "too_few_words", fmt.Sprintf("Blog minimal %d kata (saat ini %d kata).", minBlogWords, res.WordCount))
		}
		if res.WordCount > maxBlogWords {
			v.add("body", "", "too_long", fmt.Sprintf("Blog maksimal %d kata (saat ini %d kata).", maxBlogWords, res.WordCount))
		}
	}

	return res, v.errs
}

// EmptyBody returns the initial body for a new content.
func EmptyBody(t ContentType) json.RawMessage {
	switch t {
	case Portfolio:
		return json.RawMessage(`{"blocks":[]}`)
	case Blog:
		return json.RawMessage(`{"type":"doc","content":[]}`)
	default:
		return nil
	}
}

type validator struct {
	submit bool
	errs   []apperr.FieldError

	texts    []string
	headings []Heading
	media    map[uuid.UUID]struct{}

	hasMediaBlock bool
	imageCount    int
	embedCount    int
}

func (v *validator) add(field, ref, code, message string) {
	v.errs = append(v.errs, apperr.FieldError{
		Field:   field,
		Ref:     ref,
		Code:    code,
		Message: message,
	})
}

func (v *validator) addText(s string) {
	if s = collapseSpace(s); s != "" {
		v.texts = append(v.texts, s)
	}
}

func (v *validator) plainText() string {
	return collapseSpace(strings.Join(v.texts, " "))
}

func (v *validator) sortedMediaIDs() []uuid.UUID {
	if len(v.media) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(v.media))
	for id := range v.media {
		ids = append(ids, id)
	}
	slicesSortUUIDs(ids)
	return ids
}

func (v *validator) addHeading(id string, level int, text string) {
	if id == "" || text == "" {
		return
	}
	v.headings = append(v.headings, Heading{ID: id, Level: level, Text: text})
}

// ---------------------------------------------------------------------------
// Generic decoding and field helpers.
// ---------------------------------------------------------------------------

func decodeJSON(raw []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var value any
	if err := dec.Decode(&value); err != nil {
		return nil, err
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, errors.New("body contains trailing data")
	}
	return value, nil
}

func asObject(value any) (map[string]any, bool) {
	obj, ok := value.(map[string]any)
	return obj, ok
}

func asArray(value any) ([]any, bool) {
	arr, ok := value.([]any)
	return arr, ok
}

func asString(value any) (string, bool) {
	s, ok := value.(string)
	return s, ok
}

// intValue accepts JSON numbers without a fractional part.
func intValue(value any) (int64, bool) {
	number, ok := value.(json.Number)
	if !ok {
		return 0, false
	}
	if i, err := strconv.ParseInt(number.String(), 10, 64); err == nil {
		return i, true
	}
	f, err := strconv.ParseFloat(number.String(), 64)
	if err != nil || f != float64(int64(f)) {
		return 0, false
	}
	return int64(f), true
}

// stringField returns a sanitized string. When the key is absent or null it
// returns false; required fields produce an error. maxRunes <= 0 disables the
// length check. trim removes surrounding whitespace.
func (v *validator) stringField(obj map[string]any, key, path, ref string, maxRunes int, trim, required bool) (string, bool) {
	raw, exists := obj[key]
	if !exists || raw == nil {
		if required {
			v.add(path, ref, "required", "Isian ini wajib diisi.")
		}
		return "", false
	}
	s, ok := asString(raw)
	if !ok {
		v.add(path, ref, "invalid_type", "Nilai harus berupa teks.")
		return "", false
	}
	s = sanitizeText(s)
	if trim {
		s = strings.TrimSpace(s)
	}
	if maxRunes > 0 && utf8.RuneCountInString(s) > maxRunes {
		v.add(path, ref, "too_long", fmt.Sprintf("Maksimal %d karakter.", maxRunes))
	}
	return s, true
}

func (v *validator) enumField(obj map[string]any, key, path, ref string, allowed map[string]struct{}, fallback string, required bool) (string, bool) {
	raw, exists := obj[key]
	if !exists || raw == nil {
		if required {
			v.add(path, ref, "required", "Isian ini wajib diisi.")
			return "", false
		}
		if fallback != "" {
			return fallback, true
		}
		return "", false
	}
	s, ok := asString(raw)
	if !ok {
		v.add(path, ref, "invalid_type", "Nilai harus berupa teks.")
		return "", false
	}
	if _, ok := allowed[s]; !ok {
		v.add(path, ref, "not_allowed", "Pilihan tidak diizinkan.")
		return "", false
	}
	return s, true
}

func (v *validator) boolField(obj map[string]any, key, path, ref string, fallback bool) bool {
	raw, exists := obj[key]
	if !exists || raw == nil {
		return fallback
	}
	b, ok := raw.(bool)
	if !ok {
		v.add(path, ref, "invalid_type", "Nilai harus berupa boolean.")
		return fallback
	}
	return b
}

// mediaIDField validates an optional/required media reference. required is
// only enforced on submit, which lets drafts reference a still-uploading media.
func (v *validator) mediaIDField(obj map[string]any, key, path, ref string, required bool) (string, bool) {
	raw, exists := obj[key]
	if !exists || raw == nil {
		if required && v.submit {
			v.add(path, ref, "required", "Media wajib dipilih.")
		}
		return "", false
	}
	s, ok := asString(raw)
	if !ok {
		v.add(path, ref, "invalid_type", "ID media harus berupa teks.")
		return "", false
	}
	id, err := uuid.Parse(s)
	if err != nil {
		v.add(path, ref, "invalid_id", "ID media tidak valid.")
		return "", false
	}
	v.media[id] = struct{}{}
	return id.String(), true
}

func (v *validator) blockIDField(obj map[string]any, path, ref string, required bool, seen map[string]bool) (string, bool) {
	raw, exists := obj["id"]
	if !exists || raw == nil {
		if required {
			v.add(path, ref, "invalid_id", "ID blok wajib berupa UUID.")
		}
		return "", false
	}
	s, ok := asString(raw)
	if !ok {
		v.add(path, ref, "invalid_id", "ID blok tidak valid.")
		return "", false
	}
	id, err := uuid.Parse(s)
	if err != nil {
		v.add(path, ref, "invalid_id", "ID blok tidak valid.")
		return "", false
	}
	key := id.String()
	if seen != nil {
		if seen[key] {
			v.add(path, s, "duplicate_id", "ID blok tidak boleh duplikat.")
			return s, false
		}
		seen[key] = true
	}
	return s, true
}

// nodeIDField reads an id stored in a ProseMirror attrs object.
func (v *validator) nodeIDField(attrs map[string]any, path, ref string, required bool, seen map[string]bool) (string, bool) {
	raw, exists := attrs["id"]
	if !exists || raw == nil {
		if required {
			v.add(path, ref, "invalid_id", "ID node wajib berupa UUID.")
		}
		return "", false
	}
	s, ok := asString(raw)
	if !ok {
		v.add(path, ref, "invalid_id", "ID node tidak valid.")
		return "", false
	}
	id, err := uuid.Parse(s)
	if err != nil {
		v.add(path, ref, "invalid_id", "ID node tidak valid.")
		return "", false
	}
	key := id.String()
	if seen != nil {
		if seen[key] {
			v.add(path, s, "duplicate_id", "ID node tidak boleh duplikat.")
			return s, false
		}
		seen[key] = true
	}
	return s, true
}

func (v *validator) objectField(obj map[string]any, key, path, ref string, required bool) (map[string]any, bool) {
	raw, exists := obj[key]
	if !exists || raw == nil {
		if required {
			v.add(path, ref, "required", "Isian ini wajib diisi.")
		}
		return map[string]any{}, false
	}
	m, ok := asObject(raw)
	if !ok {
		v.add(path, ref, "invalid_type", "Nilai harus berupa objek JSON.")
		return map[string]any{}, false
	}
	return m, true
}

func (v *validator) arrayField(obj map[string]any, key, path, ref string, required bool) ([]any, bool) {
	raw, exists := obj[key]
	if !exists || raw == nil {
		if required {
			v.add(path, ref, "required", "Isian ini wajib diisi.")
		}
		return nil, false
	}
	arr, ok := asArray(raw)
	if !ok {
		v.add(path, ref, "invalid_type", "Nilai harus berupa array.")
		return nil, false
	}
	return arr, true
}

// ---------------------------------------------------------------------------
// Shared string helpers.
// ---------------------------------------------------------------------------

// sanitizeText removes NUL and control characters except newline and tab.
func sanitizeText(s string) string {
	if !strings.ContainsFunc(s, func(r rune) bool {
		return unicode.IsControl(r) && r != '\n' && r != '\t'
	}) {
		return s
	}
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, s)
}

// collapseSpace trims and collapses Unicode whitespace runs to one space.
func collapseSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func readingMinutes(t ContentType, words int) int {
	if t != Blog {
		return 0
	}
	minutes := (words + wordsPerMinute - 1) / wordsPerMinute
	if minutes < 1 {
		return 1
	}
	return minutes
}

func wordCount(plain string) int {
	return len(strings.Fields(plain))
}

func autoExcerpt(plain string) string {
	runes := []rune(plain)
	if len(runes) <= maxExcerptRunes {
		return plain
	}
	prefix := runes[:maxExcerptRunes]
	cut := len(prefix)
	if !unicode.IsSpace(runes[maxExcerptRunes]) {
		for i := len(prefix) - 1; i > 0; i-- {
			if unicode.IsSpace(prefix[i]) {
				cut = i
				break
			}
		}
	}
	return strings.TrimRightFunc(string(prefix[:cut]), unicode.IsSpace) + "…"
}
