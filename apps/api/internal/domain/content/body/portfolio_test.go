package body

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
)

func testUUID(n int) string {
	return fmt.Sprintf("0190a5b2-7c4d-7e8f-9a0b-%012x", n)
}

func mustNormalize(t *testing.T, contentType ContentType, raw string) (Result, []apperr.FieldError) {
	t.Helper()
	return Normalize(contentType, json.RawMessage(raw))
}

func mustSubmit(t *testing.T, contentType ContentType, raw string) (Result, []apperr.FieldError) {
	t.Helper()
	return ValidateForSubmit(contentType, json.RawMessage(raw))
}

func hasFieldError(errs []apperr.FieldError, field, code string) bool {
	for _, e := range errs {
		if e.Field == field && e.Code == code {
			return true
		}
	}
	return false
}

func hasCode(errs []apperr.FieldError, code string) bool {
	for _, e := range errs {
		if e.Code == code {
			return true
		}
	}
	return false
}

func requireNoErrors(t *testing.T, errs []apperr.FieldError) {
	t.Helper()
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %+v", errs)
	}
}

func requireFieldError(t *testing.T, errs []apperr.FieldError, field, code string) {
	t.Helper()
	if !hasFieldError(errs, field, code) {
		t.Fatalf("expected field error %s/%s, got %+v", field, code, errs)
	}
}

func decodeBody(t *testing.T, res Result) map[string]any {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal(res.Body, &v); err != nil {
		t.Fatalf("body is not valid JSON: %v (%s)", err, res.Body)
	}
	return v
}

func blocksOf(t *testing.T, res Result) []any {
	t.Helper()
	body := decodeBody(t, res)
	blocks, ok := body["blocks"].([]any)
	if !ok {
		t.Fatalf("body.blocks is not an array: %s", res.Body)
	}
	return blocks
}

func TestEmptyBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   ContentType
		want string
	}{
		{"portfolio", Portfolio, `{"blocks":[]}`},
		{"blog", Blog, `{"type":"doc","content":[]}`},
		{"unknown", ContentType("other"), ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := string(EmptyBody(tc.in))
			if got != tc.want {
				t.Fatalf("EmptyBody(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestNormalize_UnknownContentType(t *testing.T) {
	t.Parallel()
	_, errs := mustNormalize(t, ContentType("podcast"), `{}`)
	if len(errs) != 1 {
		t.Fatalf("expected one error, got %+v", errs)
	}
	if errs[0].Code != "unknown_type" || errs[0].Field != "type" {
		t.Fatalf("unexpected error: %+v", errs[0])
	}
}

func TestNormalize_InvalidJSON(t *testing.T) {
	t.Parallel()
	_, errs := mustNormalize(t, Blog, `{"type":`)
	if len(errs) != 1 || errs[0].Code != "invalid_json" {
		t.Fatalf("unexpected errors: %+v", errs)
	}

	_, errs = mustNormalize(t, Blog, `{"type":"doc","content":[]} {}`)
	if len(errs) != 1 || errs[0].Code != "invalid_json" {
		t.Fatalf("expected trailing data to be rejected, got %+v", errs)
	}
}

func TestNormalize_BodyTooLarge(t *testing.T) {
	t.Parallel()
	raw := `{"blocks":[{"id":"` + testUUID(1) + `","type":"divider","data":{},"x":"` +
		strings.Repeat("a", MaxBodyBytes) + `"}]}`
	_, errs := mustNormalize(t, Portfolio, raw)
	if len(errs) != 1 {
		t.Fatalf("expected exactly one error, got %d: %+v", len(errs), errs)
	}
	if errs[0].Code != "too_large" || errs[0].Field != "body" {
		t.Fatalf("unexpected error: %+v", errs[0])
	}
}

func TestNormalize_PortfolioMinimal(t *testing.T) {
	t.Parallel()
	raw := fmt.Sprintf(`{"blocks":[{"id":%q,"type":"image","data":{"media_id":%q,"alt":"Poster","caption":"Final","fit":"wide"}}]}`,
		testUUID(1), testUUID(100))
	res, errs := mustNormalize(t, Portfolio, raw)
	requireNoErrors(t, errs)
	if len(res.MediaIDs) != 1 || res.MediaIDs[0] != uuid.MustParse(testUUID(100)) {
		t.Fatalf("unexpected media ids: %v", res.MediaIDs)
	}
	if res.PlainText != "Final" {
		t.Fatalf("PlainText = %q, want %q", res.PlainText, "Final")
	}
	if res.WordCount != 1 {
		t.Fatalf("WordCount = %d, want 1", res.WordCount)
	}
	if res.ReadingMinutes != 0 {
		t.Fatalf("portfolio ReadingMinutes = %d, want 0", res.ReadingMinutes)
	}
	blocks := blocksOf(t, res)
	if len(blocks) != 1 {
		t.Fatalf("blocks = %d, want 1", len(blocks))
	}
}

func TestNormalize_PortfolioRootShape(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		raw   string
		field string
		code  string
	}{
		{"null", `null`, "body", "invalid_type"},
		{"array", `[]`, "body", "invalid_type"},
		{"missing blocks", `{}`, "body.blocks", "required"},
		{"blocks not array", `{"blocks":{}}`, "body.blocks", "invalid_type"},
		{"block not object", `{"blocks":[1]}`, "body.blocks[0]", "invalid_type"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, errs := mustNormalize(t, Portfolio, tc.raw)
			requireFieldError(t, errs, tc.field, tc.code)
		})
	}
}

func TestNormalize_PortfolioBlockIDs(t *testing.T) {
	t.Parallel()

	t.Run("missing id", func(t *testing.T) {
		t.Parallel()
		_, errs := mustNormalize(t, Portfolio, `{"blocks":[{"type":"divider","data":{}}]}`)
		requireFieldError(t, errs, "body.blocks[0].id", "invalid_id")
	})

	t.Run("invalid id", func(t *testing.T) {
		t.Parallel()
		_, errs := mustNormalize(t, Portfolio, `{"blocks":[{"id":"nope","type":"divider","data":{}}]}`)
		requireFieldError(t, errs, "body.blocks[0].id", "invalid_id")
	})

	t.Run("non-string id", func(t *testing.T) {
		t.Parallel()
		_, errs := mustNormalize(t, Portfolio, `{"blocks":[{"id":12,"type":"divider","data":{}}]}`)
		requireFieldError(t, errs, "body.blocks[0].id", "invalid_id")
	})

	t.Run("duplicate id", func(t *testing.T) {
		t.Parallel()
		raw := fmt.Sprintf(`{"blocks":[{"id":%q,"type":"divider","data":{}},{"id":%q,"type":"divider","data":{}}]}`,
			testUUID(1), testUUID(1))
		_, errs := mustNormalize(t, Portfolio, raw)
		requireFieldError(t, errs, "body.blocks[1].id", "duplicate_id")
	})

	t.Run("duplicate id different case", func(t *testing.T) {
		t.Parallel()
		raw := fmt.Sprintf(`{"blocks":[{"id":%q,"type":"divider","data":{}},{"id":%q,"type":"divider","data":{}}]}`,
			testUUID(1), strings.ToUpper(testUUID(1)))
		_, errs := mustNormalize(t, Portfolio, raw)
		requireFieldError(t, errs, "body.blocks[1].id", "duplicate_id")
	})

	t.Run("unknown type", func(t *testing.T) {
		t.Parallel()
		raw := fmt.Sprintf(`{"blocks":[{"id":%q,"type":"carousel","data":{}}]}`, testUUID(1))
		_, errs := mustNormalize(t, Portfolio, raw)
		requireFieldError(t, errs, "body.blocks[0].type", "unknown_type")
	})

	t.Run("missing type", func(t *testing.T) {
		t.Parallel()
		raw := fmt.Sprintf(`{"blocks":[{"id":%q,"data":{}}]}`, testUUID(1))
		_, errs := mustNormalize(t, Portfolio, raw)
		requireFieldError(t, errs, "body.blocks[0].type", "unknown_type")
	})
}

func TestNormalize_PortfolioBlockLimit(t *testing.T) {
	t.Parallel()
	var b strings.Builder
	b.WriteString(`{"blocks":[`)
	for i := 0; i < 61; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `{"id":%q,"type":"divider","data":{}}`, testUUID(i+1))
	}
	b.WriteString(`]}`)
	res, errs := mustNormalize(t, Portfolio, b.String())
	requireFieldError(t, errs, "body.blocks", "too_many")
	if got := len(blocksOf(t, res)); got != 60 {
		t.Fatalf("normalized blocks = %d, want 60", got)
	}
}

func TestNormalize_ImageBlock(t *testing.T) {
	t.Parallel()

	base := func(data string) string {
		return fmt.Sprintf(`{"blocks":[{"id":%q,"type":"image","data":%s}]}`, testUUID(1), data)
	}

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		res, errs := mustNormalize(t, Portfolio, base(fmt.Sprintf(`{"media_id":%q,"alt":"Alt","caption":"Cap","fit":"contain"}`, testUUID(100))))
		requireNoErrors(t, errs)
		block := blocksOf(t, res)[0].(map[string]any)
		data := block["data"].(map[string]any)
		if data["fit"] != "contain" {
			t.Fatalf("fit = %v", data["fit"])
		}
	})

	t.Run("defaults fit", func(t *testing.T) {
		t.Parallel()
		res, errs := mustNormalize(t, Portfolio, base(`{}`))
		requireNoErrors(t, errs)
		data := blocksOf(t, res)[0].(map[string]any)["data"].(map[string]any)
		if data["fit"] != "contain" {
			t.Fatalf("fit default = %v", data["fit"])
		}
		if data["media_id"] != nil {
			t.Fatalf("media_id = %v, want null", data["media_id"])
		}
	})

	tests := []struct {
		name  string
		data  string
		field string
		code  string
	}{
		{"alt too long", fmt.Sprintf(`{"alt":%q}`, strings.Repeat("a", 201)), "body.blocks[0].data.alt", "too_long"},
		{"caption too long", fmt.Sprintf(`{"caption":%q}`, strings.Repeat("a", 301)), "body.blocks[0].data.caption", "too_long"},
		{"invalid fit", `{"fit":"cover"}`, "body.blocks[0].data.fit", "not_allowed"},
		{"invalid media id", `{"media_id":"nope"}`, "body.blocks[0].data.media_id", "invalid_id"},
		{"wrong alt type", `{"alt":3}`, "body.blocks[0].data.alt", "invalid_type"},
		{"wrong caption type", `{"caption":[]}`, "body.blocks[0].data.caption", "invalid_type"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, errs := mustNormalize(t, Portfolio, base(tc.data))
			requireFieldError(t, errs, tc.field, tc.code)
		})
	}

	t.Run("submit requires media", func(t *testing.T) {
		t.Parallel()
		_, errs := mustSubmit(t, Portfolio, base(`{}`))
		requireFieldError(t, errs, "body.blocks[0].data.media_id", "required")
	})
}

func TestNormalize_GalleryBlock(t *testing.T) {
	t.Parallel()

	base := func(data string) string {
		return fmt.Sprintf(`{"blocks":[{"id":%q,"type":"gallery","data":%s}]}`, testUUID(1), data)
	}
	item := func(n int, caption string) string {
		return fmt.Sprintf(`{"media_id":%q,"alt":"a","caption":%q}`, testUUID(100+n), caption)
	}

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		raw := base(fmt.Sprintf(`{"items":[%s,%s],"layout":"carousel"}`, item(1, "one"), item(2, "two")))
		res, errs := mustNormalize(t, Portfolio, raw)
		requireNoErrors(t, errs)
		if len(res.MediaIDs) != 2 {
			t.Fatalf("media ids = %v", res.MediaIDs)
		}
		if res.PlainText != "one two" {
			t.Fatalf("PlainText = %q", res.PlainText)
		}
	})

	t.Run("defaults layout", func(t *testing.T) {
		t.Parallel()
		res, errs := mustNormalize(t, Portfolio, base(fmt.Sprintf(`{"items":[%s,%s]}`, item(1, ""), item(2, ""))))
		requireNoErrors(t, errs)
		data := blocksOf(t, res)[0].(map[string]any)["data"].(map[string]any)
		if data["layout"] != "grid" {
			t.Fatalf("layout default = %v", data["layout"])
		}
	})

	t.Run("too many items", func(t *testing.T) {
		t.Parallel()
		items := make([]string, 21)
		for i := range items {
			items[i] = item(i+1, "")
		}
		_, errs := mustNormalize(t, Portfolio, base(`{"items":[`+strings.Join(items, ",")+`]}`))
		requireFieldError(t, errs, "body.blocks[0].data.items", "too_many")
	})

	t.Run("submit too few items", func(t *testing.T) {
		t.Parallel()
		_, errs := mustSubmit(t, Portfolio, base(fmt.Sprintf(`{"items":[%s]}`, item(1, ""))))
		requireFieldError(t, errs, "body.blocks[0].data.items", "too_few")
	})

	t.Run("submit requires item media", func(t *testing.T) {
		t.Parallel()
		_, errs := mustSubmit(t, Portfolio, base(`{"items":[{"media_id":null},{"media_id":null}]}`))
		requireFieldError(t, errs, "body.blocks[0].data.items[0].media_id", "required")
	})

	t.Run("invalid layout", func(t *testing.T) {
		t.Parallel()
		_, errs := mustNormalize(t, Portfolio, base(fmt.Sprintf(`{"items":[%s,%s],"layout":"masonry"}`, item(1, ""), item(2, ""))))
		requireFieldError(t, errs, "body.blocks[0].data.layout", "not_allowed")
	})

	t.Run("item not object", func(t *testing.T) {
		t.Parallel()
		_, errs := mustNormalize(t, Portfolio, base(`{"items":[1,2]}`))
		requireFieldError(t, errs, "body.blocks[0].data.items[0]", "invalid_type")
	})

	t.Run("item caption too long", func(t *testing.T) {
		t.Parallel()
		_, errs := mustNormalize(t, Portfolio, base(fmt.Sprintf(`{"items":[%s,%s]}`,
			item(1, strings.Repeat("c", 301)), item(2, ""))))
		requireFieldError(t, errs, "body.blocks[0].data.items[0].caption", "too_long")
	})

	t.Run("layout wrong type", func(t *testing.T) {
		t.Parallel()
		_, errs := mustNormalize(t, Portfolio, base(fmt.Sprintf(`{"items":[%s,%s],"layout":1}`, item(1, ""), item(2, ""))))
		requireFieldError(t, errs, "body.blocks[0].data.layout", "invalid_type")
	})

	t.Run("items wrong type", func(t *testing.T) {
		t.Parallel()
		_, errs := mustNormalize(t, Portfolio, base(`{"items":{}}`))
		requireFieldError(t, errs, "body.blocks[0].data.items", "invalid_type")
	})
}

func TestNormalize_VideoBlock(t *testing.T) {
	t.Parallel()

	base := func(data string) string {
		return fmt.Sprintf(`{"blocks":[{"id":%q,"type":"video","data":%s}]}`, testUUID(1), data)
	}

	t.Run("youtube", func(t *testing.T) {
		t.Parallel()
		res, errs := mustNormalize(t, Portfolio, base(`{"provider":"youtube","video_id":"dQw4w9WgXcQ","start":12,"title":"  Demo  "}`))
		requireNoErrors(t, errs)
		data := blocksOf(t, res)[0].(map[string]any)["data"].(map[string]any)
		if data["title"] != "Demo" {
			t.Fatalf("title not trimmed: %v", data["title"])
		}
	})

	t.Run("vimeo", func(t *testing.T) {
		t.Parallel()
		_, errs := mustNormalize(t, Portfolio, base(`{"provider":"vimeo","video_id":"123456789"}`))
		requireNoErrors(t, errs)
	})

	tests := []struct {
		name  string
		data  string
		field string
		code  string
	}{
		{"missing provider", `{"video_id":"dQw4w9WgXcQ"}`, "body.blocks[0].data.provider", "required"},
		{"invalid provider", `{"provider":"tiktok","video_id":"dQw4w9WgXcQ"}`, "body.blocks[0].data.provider", "not_allowed"},
		{"provider wrong type", `{"provider":1,"video_id":"dQw4w9WgXcQ"}`, "body.blocks[0].data.provider", "invalid_type"},
		{"missing video id", `{"provider":"youtube"}`, "body.blocks[0].data.video_id", "required"},
		{"short youtube id", `{"provider":"youtube","video_id":"short"}`, "body.blocks[0].data.video_id", "invalid_id"},
		{"non numeric vimeo id", `{"provider":"vimeo","video_id":"abc"}`, "body.blocks[0].data.video_id", "invalid_id"},
		{"negative start", `{"provider":"youtube","video_id":"dQw4w9WgXcQ","start":-1}`, "body.blocks[0].data.start", "invalid_value"},
		{"fractional start", `{"provider":"youtube","video_id":"dQw4w9WgXcQ","start":1.5}`, "body.blocks[0].data.start", "invalid_type"},
		{"start wrong type", `{"provider":"youtube","video_id":"dQw4w9WgXcQ","start":"1"}`, "body.blocks[0].data.start", "invalid_type"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, errs := mustNormalize(t, Portfolio, base(tc.data))
			requireFieldError(t, errs, tc.field, tc.code)
		})
	}
}

func TestNormalize_EmbedBlock(t *testing.T) {
	t.Parallel()

	base := func(data string) string {
		return fmt.Sprintf(`{"blocks":[{"id":%q,"type":"embed","data":%s}]}`, testUUID(1), data)
	}

	t.Run("figma canonicalized", func(t *testing.T) {
		t.Parallel()
		res, errs := mustNormalize(t, Portfolio, base(`{"provider":"figma","url":"https://www.figma.com/file/AbC123/My-Design?node-id=1-2","aspect":"4:3"}`))
		requireNoErrors(t, errs)
		data := blocksOf(t, res)[0].(map[string]any)["data"].(map[string]any)
		if data["url"] != "https://www.figma.com/file/AbC123" {
			t.Fatalf("canonical url = %v", data["url"])
		}
		if data["aspect"] != "4:3" {
			t.Fatalf("aspect = %v", data["aspect"])
		}
	})

	t.Run("default aspect", func(t *testing.T) {
		t.Parallel()
		res, errs := mustNormalize(t, Portfolio, base(`{"provider":"spotify","url":"https://open.spotify.com/track/abc123"} `))
		requireNoErrors(t, errs)
		data := blocksOf(t, res)[0].(map[string]any)["data"].(map[string]any)
		if data["aspect"] != "16:9" {
			t.Fatalf("aspect default = %v", data["aspect"])
		}
		if data["url"] != "https://open.spotify.com/track/abc123" {
			t.Fatalf("url = %v", data["url"])
		}
	})

	tests := []struct {
		name  string
		data  string
		field string
		code  string
	}{
		{"missing provider", `{"url":"https://open.spotify.com/track/abc123"}`, "body.blocks[0].data.provider", "required"},
		{"unknown provider", `{"provider":"vimeo","url":"https://vimeo.com/123"}`, "body.blocks[0].data.provider", "not_allowed"},
		{"missing url", `{"provider":"figma"}`, "body.blocks[0].data.url", "required"},
		{"host not allowed", `{"provider":"figma","url":"https://evil.com/file/x"}`, "body.blocks[0].data.url", "invalid_url"},
		{"provider mismatch", `{"provider":"figma","url":"https://open.spotify.com/track/abc123"}`, "body.blocks[0].data.provider", "not_allowed"},
		{"invalid aspect", `{"provider":"figma","url":"https://www.figma.com/file/AbC123","aspect":"21:9"}`, "body.blocks[0].data.aspect", "not_allowed"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, errs := mustNormalize(t, Portfolio, base(tc.data))
			requireFieldError(t, errs, tc.field, tc.code)
		})
	}
}

func TestNormalize_FileBlock(t *testing.T) {
	t.Parallel()
	base := func(data string) string {
		return fmt.Sprintf(`{"blocks":[{"id":%q,"type":"file","data":%s}]}`, testUUID(1), data)
	}

	res, errs := mustNormalize(t, Portfolio, base(fmt.Sprintf(`{"media_id":%q,"title":"  Laporan  "}`, testUUID(100))))
	requireNoErrors(t, errs)
	data := blocksOf(t, res)[0].(map[string]any)["data"].(map[string]any)
	if data["title"] != "Laporan" {
		t.Fatalf("title = %v", data["title"])
	}
	if len(res.MediaIDs) != 1 {
		t.Fatalf("media ids = %v", res.MediaIDs)
	}

	_, errs = mustNormalize(t, Portfolio, base(fmt.Sprintf(`{"title":%q}`, strings.Repeat("t", 121))))
	requireFieldError(t, errs, "body.blocks[0].data.title", "too_long")

	_, errs = mustSubmit(t, Portfolio, base(`{}`))
	requireFieldError(t, errs, "body.blocks[0].data.media_id", "required")
}

func TestNormalize_Model3DBlock(t *testing.T) {
	t.Parallel()
	raw := fmt.Sprintf(`{"blocks":[{"id":%q,"type":"model3d","data":{"media_id":%q,"poster_media_id":%q,"auto_rotate":true}}]}`,
		testUUID(1), testUUID(100), testUUID(101))
	res, errs := mustNormalize(t, Portfolio, raw)
	requireNoErrors(t, errs)
	if len(res.MediaIDs) != 2 {
		t.Fatalf("media ids = %v", res.MediaIDs)
	}
	data := blocksOf(t, res)[0].(map[string]any)["data"].(map[string]any)
	if data["auto_rotate"] != true {
		t.Fatalf("auto_rotate = %v", data["auto_rotate"])
	}

	_, errs = mustNormalize(t, Portfolio, fmt.Sprintf(`{"blocks":[{"id":%q,"type":"model3d","data":{"media_id":%q,"poster_media_id":"bad"}}]}`,
		testUUID(1), testUUID(100)))
	requireFieldError(t, errs, "body.blocks[0].data.poster_media_id", "invalid_id")

	_, errs = mustNormalize(t, Portfolio, fmt.Sprintf(`{"blocks":[{"id":%q,"type":"model3d","data":{"auto_rotate":"yes"}}]}`,
		testUUID(1)))
	requireFieldError(t, errs, "body.blocks[0].data.auto_rotate", "invalid_type")

	_, errs = mustSubmit(t, Portfolio, fmt.Sprintf(`{"blocks":[{"id":%q,"type":"model3d","data":{}}]}`,
		testUUID(1)))
	requireFieldError(t, errs, "body.blocks[0].data.media_id", "required")
}

func TestNormalize_TextBlock(t *testing.T) {
	t.Parallel()

	doc := func(content string) string {
		return fmt.Sprintf(`{"blocks":[{"id":%q,"type":"text","data":{"doc":{"type":"doc","content":[%s]}}}]}`, testUUID(1), content)
	}

	t.Run("valid subset", func(t *testing.T) {
		t.Parallel()
		raw := doc(`{"type":"paragraph","content":[
			{"type":"text","text":"Hello "},
			{"type":"text","text":"world","marks":[{"type":"bold"},{"type":"italic"},{"type":"code"}]},
			{"type":"hardBreak"},
			{"type":"text","text":"link","marks":[{"type":"link","attrs":{"href":"https://example.com/x"}}]},
			{"type":"text","text":"internal","marks":[{"type":"link","attrs":{"href":"/karya/abc"}}]}
		]},
		{"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Judul"}]},
		{"type":"bulletList","content":[{"type":"listItem","content":[
			{"type":"paragraph","content":[{"type":"text","text":"Satu"}]},
			{"type":"orderedList","content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Nested"}]}]}]}
		]}]}`)
		res, errs := mustNormalize(t, Portfolio, raw)
		requireNoErrors(t, errs)
		if !strings.Contains(res.PlainText, "Hello world") || !strings.Contains(res.PlainText, "Judul") {
			t.Fatalf("PlainText = %q", res.PlainText)
		}
	})

	tests := []struct {
		name  string
		raw   string
		field string
		code  string
	}{
		{"missing doc", fmt.Sprintf(`{"blocks":[{"id":%q,"type":"text","data":{}}]}`, testUUID(1)), "body.blocks[0].data.doc", "required"},
		{"doc not object", fmt.Sprintf(`{"blocks":[{"id":%q,"type":"text","data":{"doc":1}}]}`, testUUID(1)), "body.blocks[0].data.doc", "invalid_type"},
		{"doc wrong type", fmt.Sprintf(`{"blocks":[{"id":%q,"type":"text","data":{"doc":{"type":"paragraph"}}}]}`, testUUID(1)), "body.blocks[0].data.doc.type", "invalid_type"},
		{"unknown node", doc(`{"type":"image"}`), "body.blocks[0].data.doc.content[0]", "not_allowed"},
		{"truly unknown node", doc(`{"type":"mystery"}`), "body.blocks[0].data.doc.content[0].type", "unknown_type"},
		{"heading level 4", doc(`{"type":"heading","attrs":{"level":4},"content":[{"type":"text","text":"x"}]}`), "body.blocks[0].data.doc.content[0].attrs.level", "invalid_value"},
		{"heading missing level", doc(`{"type":"heading","attrs":{}}`), "body.blocks[0].data.doc.content[0].attrs.level", "required"},
		{"unknown mark", doc(`{"type":"paragraph","content":[{"type":"text","text":"x","marks":[{"type":"underline"}]}]}`), "body.blocks[0].data.doc.content[0].content[0].marks[0].type", "unknown_type"},
		{"strike mark not allowed", doc(`{"type":"paragraph","content":[{"type":"text","text":"x","marks":[{"type":"strike"}]}]}`), "body.blocks[0].data.doc.content[0].content[0].marks[0].type", "not_allowed"},
		{"link http", doc(`{"type":"paragraph","content":[{"type":"text","text":"x","marks":[{"type":"link","attrs":{"href":"http://example.com"}}]}]}`), "body.blocks[0].data.doc.content[0].content[0].marks[0].attrs.href", "invalid_url"},
		{"link protocol relative", doc(`{"type":"paragraph","content":[{"type":"text","text":"x","marks":[{"type":"link","attrs":{"href":"//evil.com"}}]}]}`), "body.blocks[0].data.doc.content[0].content[0].marks[0].attrs.href", "invalid_url"},
		{"link missing href", doc(`{"type":"paragraph","content":[{"type":"text","text":"x","marks":[{"type":"link","attrs":{}}]}]}`), "body.blocks[0].data.doc.content[0].content[0].marks[0].attrs.href", "required"},
		{"list wrong child", doc(`{"type":"bulletList","content":[{"type":"paragraph","content":[{"type":"text","text":"x"}]}]}`), "body.blocks[0].data.doc.content[0].content[0]", "not_allowed"},
		{"inline image", doc(`{"type":"paragraph","content":[{"type":"image"}]}`), "body.blocks[0].data.doc.content[0].content[0]", "not_allowed"},
		{"top level text", doc(`{"type":"text","text":"x"}`), "body.blocks[0].data.doc.content[0]", "not_allowed"},
		{"text node missing text", doc(`{"type":"paragraph","content":[{"type":"text"}]}`), "body.blocks[0].data.doc.content[0].content[0].text", "invalid_type"},
		{"marks wrong type", doc(`{"type":"paragraph","content":[{"type":"text","text":"x","marks":{}}]}`), "body.blocks[0].data.doc.content[0].content[0].marks", "invalid_type"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, errs := mustNormalize(t, Portfolio, tc.raw)
			requireFieldError(t, errs, tc.field, tc.code)
		})
	}

	t.Run("text too long", func(t *testing.T) {
		t.Parallel()
		raw := doc(fmt.Sprintf(`{"type":"paragraph","content":[{"type":"text","text":%q}]}`, strings.Repeat("a", 5001)))
		_, errs := mustNormalize(t, Portfolio, raw)
		requireFieldError(t, errs, "body.blocks[0].data.doc.content", "too_long")
	})

	t.Run("text exactly at limit", func(t *testing.T) {
		t.Parallel()
		raw := doc(fmt.Sprintf(`{"type":"paragraph","content":[{"type":"text","text":%q}]}`, strings.Repeat("a", 5000)))
		_, errs := mustNormalize(t, Portfolio, raw)
		requireNoErrors(t, errs)
	})

	t.Run("doc content wrong type", func(t *testing.T) {
		t.Parallel()
		raw := fmt.Sprintf(`{"blocks":[{"id":%q,"type":"text","data":{"doc":{"type":"doc","content":{}}}}]}`, testUUID(1))
		_, errs := mustNormalize(t, Portfolio, raw)
		requireFieldError(t, errs, "body.blocks[0].data.doc.content", "invalid_type")
	})

	t.Run("list item non object", func(t *testing.T) {
		t.Parallel()
		raw := doc(`{"type":"bulletList","content":[1]}`)
		_, errs := mustNormalize(t, Portfolio, raw)
		requireFieldError(t, errs, "body.blocks[0].data.doc.content[0].content[0]", "invalid_type")
	})

	t.Run("list item wrong type", func(t *testing.T) {
		t.Parallel()
		raw := doc(`{"type":"bulletList","content":[{"type":"heading","content":[]}]}`)
		_, errs := mustNormalize(t, Portfolio, raw)
		requireFieldError(t, errs, "body.blocks[0].data.doc.content[0].content[0]", "not_allowed")
	})

	t.Run("list content wrong type", func(t *testing.T) {
		t.Parallel()
		raw := doc(`{"type":"bulletList","content":{}}`)
		_, errs := mustNormalize(t, Portfolio, raw)
		requireFieldError(t, errs, "body.blocks[0].data.doc.content[0].content", "invalid_type")
	})

	t.Run("list item content wrong type", func(t *testing.T) {
		t.Parallel()
		raw := doc(`{"type":"bulletList","content":[{"type":"listItem","content":{}}]}`)
		_, errs := mustNormalize(t, Portfolio, raw)
		requireFieldError(t, errs, "body.blocks[0].data.doc.content[0].content[0].content", "invalid_type")
	})

	t.Run("list text counted toward limit", func(t *testing.T) {
		t.Parallel()
		raw := doc(fmt.Sprintf(`{"type":"bulletList","content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":%q}]}]}]}`, strings.Repeat("a", 5001)))
		_, errs := mustNormalize(t, Portfolio, raw)
		requireFieldError(t, errs, "body.blocks[0].data.doc.content", "too_long")
	})
}

func TestNormalize_CodeBlock(t *testing.T) {
	t.Parallel()
	base := func(data string) string {
		return fmt.Sprintf(`{"blocks":[{"id":%q,"type":"code","data":%s}]}`, testUUID(1), data)
	}

	res, errs := mustNormalize(t, Portfolio, base(`{"language":"go","code":"fmt.Println(1)","filename":" main.go "}`))
	requireNoErrors(t, errs)
	data := blocksOf(t, res)[0].(map[string]any)["data"].(map[string]any)
	if data["filename"] != "main.go" {
		t.Fatalf("filename = %v", data["filename"])
	}

	tests := []struct {
		name  string
		data  string
		field string
		code  string
	}{
		{"missing language", `{"code":"x"}`, "body.blocks[0].data.language", "required"},
		{"unknown language", `{"language":"brainfuck","code":"x"}`, "body.blocks[0].data.language", "not_allowed"},
		{"language wrong type", `{"language":1,"code":"x"}`, "body.blocks[0].data.language", "invalid_type"},
		{"missing code", `{"language":"go"}`, "body.blocks[0].data.code", "required"},
		{"code wrong type", `{"language":"go","code":1}`, "body.blocks[0].data.code", "invalid_type"},
		{"code too long", fmt.Sprintf(`{"language":"go","code":%q}`, strings.Repeat("x", 10001)), "body.blocks[0].data.code", "too_long"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, errs := mustNormalize(t, Portfolio, base(tc.data))
			requireFieldError(t, errs, tc.field, tc.code)
		})
	}

	t.Run("all languages accepted", func(t *testing.T) {
		t.Parallel()
		for lang := range codeLanguages {
			raw := base(fmt.Sprintf(`{"language":%q,"code":"x"}`, lang))
			if _, errs := mustNormalize(t, Portfolio, raw); len(errs) != 0 {
				t.Fatalf("language %q rejected: %+v", lang, errs)
			}
		}
	})
}

func TestNormalize_TableBlock(t *testing.T) {
	t.Parallel()
	base := func(data string) string {
		return fmt.Sprintf(`{"blocks":[{"id":%q,"type":"table","data":%s}]}`, testUUID(1), data)
	}

	res, errs := mustNormalize(t, Portfolio, base(`{"headers":["A","B"],"rows":[["1","2"],["3","4"]]}`))
	requireNoErrors(t, errs)
	if res.PlainText != "A B 1 2 3 4" {
		t.Fatalf("PlainText = %q", res.PlainText)
	}

	tests := []struct {
		name  string
		data  string
		field string
		code  string
	}{
		{"missing headers", `{"rows":[]}`, "body.blocks[0].data.headers", "required"},
		{"headers wrong type", `{"headers":{},"rows":[]}`, "body.blocks[0].data.headers", "invalid_type"},
		{"too many headers", fmt.Sprintf(`{"headers":%s,"rows":[]}`, jsonArrayOfStrings(11, 1)), "body.blocks[0].data.headers", "too_many"},
		{"header too long", fmt.Sprintf(`{"headers":[%q],"rows":[]}`, strings.Repeat("h", 201)), "body.blocks[0].data.headers[0]", "too_long"},
		{"header wrong type", `{"headers":[1],"rows":[]}`, "body.blocks[0].data.headers[0]", "invalid_type"},
		{"missing rows", `{"headers":[]}`, "body.blocks[0].data.rows", "required"},
		{"rows wrong type", `{"headers":[],"rows":{}}`, "body.blocks[0].data.rows", "invalid_type"},
		{"row not array", `{"headers":[],"rows":[1]}`, "body.blocks[0].data.rows[0]", "invalid_type"},
		{"too many rows", fmt.Sprintf(`{"headers":[],"rows":%s}`, jsonRows(51, 1, 1)), "body.blocks[0].data.rows", "too_many"},
		{"too many columns", fmt.Sprintf(`{"headers":[],"rows":[%s]}`, jsonArrayOfStrings(11, 1)), "body.blocks[0].data.rows[0]", "too_many"},
		{"cell too long", fmt.Sprintf(`{"headers":[],"rows":[[%q]]}`, strings.Repeat("c", 201)), "body.blocks[0].data.rows[0][0]", "too_long"},
		{"cell wrong type", `{"headers":[],"rows":[[1]]}`, "body.blocks[0].data.rows[0][0]", "invalid_type"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, errs := mustNormalize(t, Portfolio, base(tc.data))
			requireFieldError(t, errs, tc.field, tc.code)
		})
	}

	t.Run("limits accepted exactly", func(t *testing.T) {
		t.Parallel()
		raw := base(fmt.Sprintf(`{"headers":%s,"rows":%s}`, jsonArrayOfStrings(10, 1), jsonRows(50, 10, 1)))
		_, errs := mustNormalize(t, Portfolio, raw)
		requireNoErrors(t, errs)
	})
}

func jsonArrayOfStrings(n, seed int) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = fmt.Sprintf(`"v%d-%d"`, seed, i)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func jsonRows(rows, cols, seed int) string {
	parts := make([]string, rows)
	for i := range parts {
		parts[i] = jsonArrayOfStrings(cols, seed+i)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func TestNormalize_LinkBlock(t *testing.T) {
	t.Parallel()
	base := func(data string) string {
		return fmt.Sprintf(`{"blocks":[{"id":%q,"type":"link","data":%s}]}`, testUUID(1), data)
	}

	res, errs := mustNormalize(t, Portfolio, base(`{"label":"  Karya  ","url":"https://example.com/a","style":"card"}`))
	requireNoErrors(t, errs)
	data := blocksOf(t, res)[0].(map[string]any)["data"].(map[string]any)
	if data["label"] != "Karya" {
		t.Fatalf("label = %v", data["label"])
	}
	if res.PlainText != "Karya" {
		t.Fatalf("PlainText = %q", res.PlainText)
	}

	tests := []struct {
		name  string
		data  string
		field string
		code  string
	}{
		{"missing label", `{"url":"https://example.com","style":"button"}`, "body.blocks[0].data.label", "required"},
		{"label too long", fmt.Sprintf(`{"label":%q,"url":"https://example.com","style":"button"}`, strings.Repeat("l", 61)), "body.blocks[0].data.label", "too_long"},
		{"missing url", `{"label":"x","style":"button"}`, "body.blocks[0].data.url", "required"},
		{"empty url", `{"label":"x","url":"","style":"button"}`, "body.blocks[0].data.url", "invalid_url"},
		{"http url", `{"label":"x","url":"http://example.com","style":"button"}`, "body.blocks[0].data.url", "invalid_url"},
		{"protocol relative url", `{"label":"x","url":"//evil.com","style":"button"}`, "body.blocks[0].data.url", "invalid_url"},
		{"javascript url", `{"label":"x","url":"javascript:alert(1)","style":"button"}`, "body.blocks[0].data.url", "invalid_url"},
		{"missing style", `{"label":"x","url":"https://example.com"}`, "body.blocks[0].data.style", "required"},
		{"invalid style", `{"label":"x","url":"https://example.com","style":"banner"}`, "body.blocks[0].data.style", "not_allowed"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, errs := mustNormalize(t, Portfolio, base(tc.data))
			requireFieldError(t, errs, tc.field, tc.code)
		})
	}
}

func TestNormalize_DividerBlock(t *testing.T) {
	t.Parallel()
	raw := fmt.Sprintf(`{"blocks":[{"id":%q,"type":"divider","data":{"evil":"dropped"},"unknown":true}]}`, testUUID(1))
	res, errs := mustNormalize(t, Portfolio, raw)
	requireNoErrors(t, errs)
	if got := string(res.Body); !strings.Contains(got, `"type":"divider"`) || strings.Contains(got, "evil") || strings.Contains(got, "unknown") {
		t.Fatalf("unexpected canonical body: %s", got)
	}
}

func TestNormalize_UnknownKeysDropped(t *testing.T) {
	t.Parallel()
	raw := fmt.Sprintf(`{"blocks":[{"id":%q,"type":"image","evil":1,"data":{"media_id":%q,"nope":"x"}}],"extra":true}`, testUUID(1), testUUID(100))
	res, errs := mustNormalize(t, Portfolio, raw)
	requireNoErrors(t, errs)
	body := string(res.Body)
	for _, unwanted := range []string{"evil", "nope", "extra"} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("body still contains %q: %s", unwanted, body)
		}
	}
}

func TestValidateForSubmit_Portfolio(t *testing.T) {
	t.Parallel()

	t.Run("requires a media block", func(t *testing.T) {
		t.Parallel()
		raw := fmt.Sprintf(`{"blocks":[{"id":%q,"type":"text","data":{"doc":{"type":"doc","content":[]}}}]}`, testUUID(1))
		_, errs := mustSubmit(t, Portfolio, raw)
		requireFieldError(t, errs, "body.blocks", "media_required")
	})

	t.Run("media block accepted", func(t *testing.T) {
		t.Parallel()
		raw := fmt.Sprintf(`{"blocks":[{"id":%q,"type":"image","data":{"media_id":%q}}]}`, testUUID(1), testUUID(100))
		_, errs := mustSubmit(t, Portfolio, raw)
		requireNoErrors(t, errs)
	})

	t.Run("too many media ids", func(t *testing.T) {
		t.Parallel()
		var b strings.Builder
		b.WriteString(`{"blocks":[`)
		for i := 0; i < 51; i++ {
			if i > 0 {
				b.WriteByte(',')
			}
			fmt.Fprintf(&b, `{"id":%q,"type":"image","data":{"media_id":%q}}`, testUUID(i+1), testUUID(1000+i))
		}
		b.WriteString(`]}`)
		_, errs := mustSubmit(t, Portfolio, b.String())
		requireFieldError(t, errs, "body.blocks", "too_many")
	})
}

func TestNormalize_MediaIDsDedupedAndSorted(t *testing.T) {
	t.Parallel()
	raw := fmt.Sprintf(`{"blocks":[
		{"id":%q,"type":"image","data":{"media_id":%q}},
		{"id":%q,"type":"gallery","data":{"items":[{"media_id":%q},{"media_id":%q}]}},
		{"id":%q,"type":"file","data":{"media_id":%q}},
		{"id":%q,"type":"model3d","data":{"media_id":%q,"poster_media_id":%q}}
	]}`,
		testUUID(1), testUUID(110),
		testUUID(2), testUUID(105), testUUID(101),
		testUUID(3), testUUID(101),
		testUUID(4), testUUID(110), testUUID(120))
	res, errs := mustNormalize(t, Portfolio, raw)
	requireNoErrors(t, errs)
	want := []uuid.UUID{
		uuid.MustParse(testUUID(101)),
		uuid.MustParse(testUUID(105)),
		uuid.MustParse(testUUID(110)),
		uuid.MustParse(testUUID(120)),
	}
	if fmt.Sprint(res.MediaIDs) != fmt.Sprint(want) {
		t.Fatalf("MediaIDs = %v, want %v", res.MediaIDs, want)
	}
}

func TestNormalize_PortfolioPlainText(t *testing.T) {
	t.Parallel()
	raw := fmt.Sprintf(`{"blocks":[
		{"id":%q,"type":"text","data":{"doc":{"type":"doc","content":[
			{"type":"paragraph","content":[{"type":"text","text":"Hello "},{"type":"text","text":"world"}]},
			{"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Judul"}]}
		]}}},
		{"id":%q,"type":"image","data":{"media_id":%q,"alt":"not included","caption":"Caption satu"}},
		{"id":%q,"type":"table","data":{"headers":["H1","H2"],"rows":[["A","B"]]}},
		{"id":%q,"type":"link","data":{"label":"Tautan","url":"https://example.com","style":"button"}}
	]}`,
		testUUID(1), testUUID(2), testUUID(100), testUUID(3), testUUID(4))
	res, errs := mustNormalize(t, Portfolio, raw)
	requireNoErrors(t, errs)
	want := "Hello world Judul Caption satu H1 H2 A B Tautan"
	if res.PlainText != want {
		t.Fatalf("PlainText = %q, want %q", res.PlainText, want)
	}
	if res.WordCount != 10 {
		t.Fatalf("WordCount = %d, want 10", res.WordCount)
	}
}

func TestNormalize_SanitizesControlCharacters(t *testing.T) {
	t.Parallel()
	raw := fmt.Sprintf(`{"blocks":[{"id":%q,"type":"image","data":{"media_id":%q,"caption":"a\u0000b\u0007c\u007f\nd\te"}}]}`, testUUID(1), testUUID(100))
	res, errs := mustNormalize(t, Portfolio, raw)
	requireNoErrors(t, errs)
	data := blocksOf(t, res)[0].(map[string]any)["data"].(map[string]any)
	if data["caption"] != "abc\nd\te" {
		t.Fatalf("caption = %q", data["caption"])
	}
	if res.PlainText != "abc d e" {
		t.Fatalf("PlainText = %q", res.PlainText)
	}
}

func TestNormalize_EmptyPortfolio(t *testing.T) {
	t.Parallel()
	res, errs := mustNormalize(t, Portfolio, `{"blocks":[]}`)
	requireNoErrors(t, errs)
	if string(res.Body) != `{"blocks":[]}` {
		t.Fatalf("Body = %s", res.Body)
	}
	if res.PlainText != "" || res.WordCount != 0 || res.ReadingMinutes != 0 || res.AutoExcerpt != "" {
		t.Fatalf("unexpected derived result: %+v", res)
	}
}
