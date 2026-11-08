package body

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func blogDoc(nodes ...string) string {
	return `{"type":"doc","content":[` + strings.Join(nodes, ",") + `]}`
}

func para(id int, text string) string {
	return fmt.Sprintf(`{"type":"paragraph","attrs":{"id":%q},"content":[{"type":"text","text":%q}]}`, testUUID(id), text)
}

func heading(id, level int, text string) string {
	return fmt.Sprintf(`{"type":"heading","attrs":{"id":%q,"level":%d},"content":[{"type":"text","text":%q}]}`,
		testUUID(id), level, text)
}

func imageNode(id, media int) string {
	return fmt.Sprintf(`{"type":"image","attrs":{"id":%q,"media_id":%q}}`, testUUID(id), testUUID(media))
}

func listNode(depth, id int) string {
	inner := fmt.Sprintf(`{"type":"paragraph","content":[{"type":"text","text":"level %d"}]}`, depth)
	if depth > 1 {
		inner += "," + listNode(depth-1, id+1)
	}
	return fmt.Sprintf(`{"type":"bulletList","attrs":{"id":%q},"content":[{"type":"listItem","content":[%s]}]}`,
		testUUID(id), inner)
}

func TestNormalize_BlogMinimal(t *testing.T) {
	t.Parallel()

	t.Run("empty doc", func(t *testing.T) {
		t.Parallel()
		res, errs := mustNormalize(t, Blog, `{"type":"doc","content":[]}`)
		requireNoErrors(t, errs)
		if string(res.Body) != `{"content":[],"type":"doc"}` {
			t.Fatalf("Body = %s", res.Body)
		}
		if res.WordCount != 0 || res.ReadingMinutes != 1 || res.PlainText != "" {
			t.Fatalf("unexpected derived result: %+v", res)
		}
	})

	t.Run("paragraph", func(t *testing.T) {
		t.Parallel()
		res, errs := mustNormalize(t, Blog, blogDoc(para(1, "Halo dunia")))
		requireNoErrors(t, errs)
		if res.PlainText != "Halo dunia" || res.WordCount != 2 {
			t.Fatalf("plain=%q words=%d", res.PlainText, res.WordCount)
		}
	})
}

func TestNormalize_BlogRootShape(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		raw   string
		field string
		code  string
	}{
		{"null", `null`, "body", "invalid_type"},
		{"array", `[]`, "body", "invalid_type"},
		{"missing type", `{"content":[]}`, "body.type", "invalid_type"},
		{"wrong type", `{"type":"document","content":[]}`, "body.type", "invalid_type"},
		{"missing content", `{"type":"doc"}`, "body.content", "required"},
		{"content not array", `{"type":"doc","content":{}}`, "body.content", "invalid_type"},
		{"node not object", `{"type":"doc","content":[1]}`, "body.content[0]", "invalid_type"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, errs := mustNormalize(t, Blog, tc.raw)
			requireFieldError(t, errs, tc.field, tc.code)
		})
	}
}

func TestNormalize_BlogNodeIDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		node  string
		field string
		code  string
	}{
		{"missing id", `{"type":"paragraph","content":[]}`, "body.content[0].attrs.id", "invalid_id"},
		{"missing attrs", `{"type":"paragraph"}`, "body.content[0].attrs.id", "invalid_id"},
		{"invalid id", `{"type":"paragraph","attrs":{"id":"nope"}}`, "body.content[0].attrs.id", "invalid_id"},
		{"non-string id", `{"type":"paragraph","attrs":{"id":7}}`, "body.content[0].attrs.id", "invalid_id"},
		{"attrs not object", `{"type":"paragraph","attrs":[]}`, "body.content[0].attrs", "invalid_type"},
		{"duplicate", `{"type":"paragraph","attrs":{"id":"` + testUUID(1) + `"}},{"type":"paragraph","attrs":{"id":"` + testUUID(1) + `"}}`, "body.content[1].attrs.id", "duplicate_id"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, errs := mustNormalize(t, Blog, blogDoc(tc.node))
			requireFieldError(t, errs, tc.field, tc.code)
		})
	}
}

func TestNormalize_BlogNodeTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		node  string
		field string
		code  string
	}{
		{"unknown", `{"type":"mystery","attrs":{"id":"` + testUUID(1) + `"}}`, "body.content[0].type", "unknown_type"},
		{"missing type", `{"attrs":{"id":"` + testUUID(1) + `"}}`, "body.content[0].type", "unknown_type"},
		{"top-level text", `{"type":"text","attrs":{"id":"` + testUUID(1) + `"},"text":"x"}`, "body.content[0]", "not_allowed"},
		{"top-level hardBreak", `{"type":"hardBreak","attrs":{"id":"` + testUUID(1) + `"}}`, "body.content[0]", "not_allowed"},
		{"top-level listItem", `{"type":"listItem","attrs":{"id":"` + testUUID(1) + `"},"content":[]}`, "body.content[0]", "not_allowed"},
		{"top-level tableRow", `{"type":"tableRow","attrs":{"id":"` + testUUID(1) + `"},"content":[]}`, "body.content[0]", "not_allowed"},
		{"inline image", `{"type":"paragraph","attrs":{"id":"` + testUUID(1) + `"},"content":[{"type":"image"}]}`, "body.content[0].content[0]", "not_allowed"},
		{"unknown inline", `{"type":"paragraph","attrs":{"id":"` + testUUID(1) + `"},"content":[{"type":"mention"}]}`, "body.content[0].content[0].type", "unknown_type"},
		{"inline node not object", `{"type":"paragraph","attrs":{"id":"` + testUUID(1) + `"},"content":[1]}`, "body.content[0].content[0]", "invalid_type"},
		{"inline missing type", `{"type":"paragraph","attrs":{"id":"` + testUUID(1) + `"},"content":[{}]}`, "body.content[0].content[0].type", "unknown_type"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, errs := mustNormalize(t, Blog, blogDoc(tc.node))
			requireFieldError(t, errs, tc.field, tc.code)
		})
	}
}

func TestNormalize_BlogMarks(t *testing.T) {
	t.Parallel()

	withMarks := func(marks string) string {
		return blogDoc(fmt.Sprintf(`{"type":"paragraph","attrs":{"id":%q},"content":[{"type":"text","text":"x","marks":%s}]}`,
			testUUID(1), marks))
	}

	t.Run("all allowed", func(t *testing.T) {
		t.Parallel()
		marks := `[{"type":"bold"},{"type":"italic"},{"type":"strike"},{"type":"code"},
			{"type":"link","attrs":{"href":"https://example.com"}},
			{"type":"link","attrs":{"href":"/karya/abc"}}]`
		res, errs := mustNormalize(t, Blog, withMarks(marks))
		requireNoErrors(t, errs)
		if !strings.Contains(string(res.Body), `"type":"strike"`) {
			t.Fatalf("strike mark dropped: %s", res.Body)
		}
	})

	tests := []struct {
		name  string
		marks string
		field string
		code  string
	}{
		{"unknown", `[{"type":"underline"}]`, "body.content[0].content[0].marks[0].type", "unknown_type"},
		{"missing type", `[{}]`, "body.content[0].content[0].marks[0].type", "unknown_type"},
		{"mark not object", `[1]`, "body.content[0].content[0].marks[0]", "invalid_type"},
		{"http link", `[{"type":"link","attrs":{"href":"http://example.com"}}]`, "body.content[0].content[0].marks[0].attrs.href", "invalid_url"},
		{"protocol relative", `[{"type":"link","attrs":{"href":"//evil.com"}}]`, "body.content[0].content[0].marks[0].attrs.href", "invalid_url"},
		{"javascript link", `[{"type":"link","attrs":{"href":"javascript:alert(1)"}}]`, "body.content[0].content[0].marks[0].attrs.href", "invalid_url"},
		{"missing link attrs", `[{"type":"link"}]`, "body.content[0].content[0].marks[0].attrs", "required"},
		{"missing href", `[{"type":"link","attrs":{}}]`, "body.content[0].content[0].marks[0].attrs.href", "required"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, errs := mustNormalize(t, Blog, withMarks(tc.marks))
			requireFieldError(t, errs, tc.field, tc.code)
		})
	}

	t.Run("marks not array", func(t *testing.T) {
		t.Parallel()
		_, errs := mustNormalize(t, Blog, withMarks(`{}`))
		requireFieldError(t, errs, "body.content[0].content[0].marks", "invalid_type")
	})

	t.Run("text not string", func(t *testing.T) {
		t.Parallel()
		node := fmt.Sprintf(`{"type":"paragraph","attrs":{"id":%q},"content":[{"type":"text","text":3}]}`, testUUID(1))
		_, errs := mustNormalize(t, Blog, blogDoc(node))
		requireFieldError(t, errs, "body.content[0].content[0].text", "invalid_type")
	})

	t.Run("hard break preserved", func(t *testing.T) {
		t.Parallel()
		node := fmt.Sprintf(`{"type":"paragraph","attrs":{"id":%q},"content":[{"type":"text","text":"a"},{"type":"hardBreak"},{"type":"text","text":"b"}]}`, testUUID(1))
		res, errs := mustNormalize(t, Blog, blogDoc(node))
		requireNoErrors(t, errs)
		if res.PlainText != "a b" {
			t.Fatalf("PlainText = %q", res.PlainText)
		}
		if !strings.Contains(string(res.Body), `"type":"hardBreak"`) {
			t.Fatalf("hardBreak dropped: %s", res.Body)
		}
	})
}

func TestNormalize_BlogHeadings(t *testing.T) {
	t.Parallel()

	t.Run("valid levels and toc", func(t *testing.T) {
		t.Parallel()
		raw := blogDoc(
			heading(1, 2, "Persiapan"),
			heading(2, 3, "Bahan"),
			heading(3, 4, "Detail kecil"),
			para(4, "Isi"),
		)
		res, errs := mustNormalize(t, Blog, raw)
		requireNoErrors(t, errs)
		if len(res.Headings) != 2 {
			t.Fatalf("headings = %+v, want H2+H3 only", res.Headings)
		}
		if res.Headings[0].ID != testUUID(1) || res.Headings[0].Level != 2 || res.Headings[0].Text != "Persiapan" {
			t.Fatalf("heading[0] = %+v", res.Headings[0])
		}
		if res.Headings[1].ID != testUUID(2) || res.Headings[1].Level != 3 {
			t.Fatalf("heading[1] = %+v", res.Headings[1])
		}
	})

	tests := []struct {
		name  string
		node  string
		field string
		code  string
	}{
		{"level 1", fmt.Sprintf(`{"type":"heading","attrs":{"id":%q,"level":1},"content":[]}`, testUUID(1)), "body.content[0].attrs.level", "invalid_value"},
		{"level 5", fmt.Sprintf(`{"type":"heading","attrs":{"id":%q,"level":5},"content":[]}`, testUUID(1)), "body.content[0].attrs.level", "invalid_value"},
		{"missing level", fmt.Sprintf(`{"type":"heading","attrs":{"id":%q},"content":[]}`, testUUID(1)), "body.content[0].attrs.level", "required"},
		{"level not integer", fmt.Sprintf(`{"type":"heading","attrs":{"id":%q,"level":2.5},"content":[]}`, testUUID(1)), "body.content[0].attrs.level", "invalid_type"},
		{"level not number", fmt.Sprintf(`{"type":"heading","attrs":{"id":%q,"level":"2"},"content":[]}`, testUUID(1)), "body.content[0].attrs.level", "invalid_type"},
		{"missing attrs", fmt.Sprintf(`{"type":"heading","content":[]}`), "body.content[0].attrs.level", "required"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, errs := mustNormalize(t, Blog, blogDoc(tc.node))
			requireFieldError(t, errs, tc.field, tc.code)
		})
	}

	t.Run("integer float accepted", func(t *testing.T) {
		t.Parallel()
		node := fmt.Sprintf(`{"type":"heading","attrs":{"id":%q,"level":2.0},"content":[{"type":"text","text":"x"}]}`, testUUID(1))
		_, errs := mustNormalize(t, Blog, blogDoc(node))
		requireNoErrors(t, errs)
	})
}

func TestNormalize_BlogLists(t *testing.T) {
	t.Parallel()

	t.Run("nesting up to three levels", func(t *testing.T) {
		t.Parallel()
		res, errs := mustNormalize(t, Blog, blogDoc(listNode(3, 1)))
		requireNoErrors(t, errs)
		if !strings.Contains(res.PlainText, "level 1") || !strings.Contains(res.PlainText, "level 3") {
			t.Fatalf("PlainText = %q", res.PlainText)
		}
	})

	t.Run("nesting four levels rejected", func(t *testing.T) {
		t.Parallel()
		_, errs := mustNormalize(t, Blog, blogDoc(listNode(4, 1)))
		if !hasCode(errs, "too_many") {
			t.Fatalf("expected too_many, got %+v", errs)
		}
	})

	tests := []struct {
		name  string
		node  string
		field string
		code  string
	}{
		{
			"list child not listItem",
			fmt.Sprintf(`{"type":"bulletList","attrs":{"id":%q},"content":[{"type":"paragraph","content":[]}]}`, testUUID(1)),
			"body.content[0].content[0]", "not_allowed",
		},
		{
			"list child not object",
			fmt.Sprintf(`{"type":"bulletList","attrs":{"id":%q},"content":[1]}`, testUUID(1)),
			"body.content[0].content[0]", "invalid_type",
		},
		{
			"listItem child not allowed",
			fmt.Sprintf(`{"type":"bulletList","attrs":{"id":%q},"content":[{"type":"listItem","content":[{"type":"image"}]}]}`, testUUID(1)),
			"body.content[0].content[0].content[0]", "not_allowed",
		},
		{
			"listItem child not object",
			fmt.Sprintf(`{"type":"bulletList","attrs":{"id":%q},"content":[{"type":"listItem","content":[1]}]}`, testUUID(1)),
			"body.content[0].content[0].content[0]", "invalid_type",
		},
		{
			"list content not array",
			fmt.Sprintf(`{"type":"bulletList","attrs":{"id":%q},"content":{}}`, testUUID(1)),
			"body.content[0].content", "invalid_type",
		},
		{
			"listItem content not array",
			fmt.Sprintf(`{"type":"bulletList","attrs":{"id":%q},"content":[{"type":"listItem","content":{}}]}`, testUUID(1)),
			"body.content[0].content[0].content", "invalid_type",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, errs := mustNormalize(t, Blog, blogDoc(tc.node))
			requireFieldError(t, errs, tc.field, tc.code)
		})
	}
}

func TestNormalize_BlogCodeBlock(t *testing.T) {
	t.Parallel()

	codeBlock := func(attrs, content string) string {
		return blogDoc(fmt.Sprintf(`{"type":"codeBlock","attrs":%s,"content":%s}`, attrs, content))
	}

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		res, errs := mustNormalize(t, Blog, codeBlock(
			fmt.Sprintf(`{"id":%q,"language":"go"}`, testUUID(1)),
			`[{"type":"text","text":"func main() {}"}]`))
		requireNoErrors(t, errs)
		if !strings.Contains(res.PlainText, "func main()") {
			t.Fatalf("PlainText = %q", res.PlainText)
		}
	})

	t.Run("default language", func(t *testing.T) {
		t.Parallel()
		res, errs := mustNormalize(t, Blog, codeBlock(fmt.Sprintf(`{"id":%q}`, testUUID(1)), `[{"type":"text","text":"x"}]`))
		requireNoErrors(t, errs)
		if !strings.Contains(string(res.Body), `"language":"plaintext"`) {
			t.Fatalf("Body = %s", res.Body)
		}
	})

	t.Run("any language label accepted", func(t *testing.T) {
		t.Parallel()
		res, errs := mustNormalize(t, Blog, codeBlock(
			fmt.Sprintf(`{"id":%q,"language":"brainfuck"}`, testUUID(1)),
			`[{"type":"text","text":"x"}]`))
		requireNoErrors(t, errs)
		if !strings.Contains(string(res.Body), `"language":"brainfuck"`) {
			t.Fatalf("Body = %s", res.Body)
		}
	})

	tests := []struct {
		name    string
		attrs   string
		content string
		field   string
		code    string
	}{
		{"too long", fmt.Sprintf(`{"id":%q,"language":"go"}`, testUUID(1)), fmt.Sprintf(`[{"type":"text","text":%q}]`, strings.Repeat("x", 10001)), "body.content[0].content", "too_long"},
		{"non text child", fmt.Sprintf(`{"id":%q,"language":"go"}`, testUUID(1)), `[{"type":"hardBreak"}]`, "body.content[0].content[0]", "not_allowed"},
		{"child not object", fmt.Sprintf(`{"id":%q,"language":"go"}`, testUUID(1)), `[1]`, "body.content[0].content[0]", "invalid_type"},
		{"text not string", fmt.Sprintf(`{"id":%q,"language":"go"}`, testUUID(1)), `[{"type":"text","text":1}]`, "body.content[0].content[0].text", "invalid_type"},
		{"marked text", fmt.Sprintf(`{"id":%q,"language":"go"}`, testUUID(1)), `[{"type":"text","text":"x","marks":[{"type":"bold"}]}]`, "body.content[0].content[0].marks", "not_allowed"},
		{"content not array", fmt.Sprintf(`{"id":%q,"language":"go"}`, testUUID(1)), `{}`, "body.content[0].content", "invalid_type"},
		{"language not string", fmt.Sprintf(`{"id":%q,"language":1}`, testUUID(1)), `[]`, "body.content[0].attrs.language", "invalid_type"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, errs := mustNormalize(t, Blog, codeBlock(tc.attrs, tc.content))
			requireFieldError(t, errs, tc.field, tc.code)
		})
	}
}

func TestNormalize_BlogCalloutAndBlockquote(t *testing.T) {
	t.Parallel()

	t.Run("callout default variant", func(t *testing.T) {
		t.Parallel()
		node := fmt.Sprintf(`{"type":"callout","attrs":{"id":%q},"content":[%s]}`, testUUID(1), para(2, "Isi"))
		res, errs := mustNormalize(t, Blog, blogDoc(node))
		requireNoErrors(t, errs)
		if !strings.Contains(string(res.Body), `"variant":"info"`) {
			t.Fatalf("Body = %s", res.Body)
		}
		if !strings.Contains(res.PlainText, "Isi") {
			t.Fatalf("PlainText = %q", res.PlainText)
		}
	})

	t.Run("callout warning", func(t *testing.T) {
		t.Parallel()
		node := fmt.Sprintf(`{"type":"callout","attrs":{"id":%q,"variant":"warning"},"content":[]}`, testUUID(1))
		_, errs := mustNormalize(t, Blog, blogDoc(node))
		requireNoErrors(t, errs)
	})

	t.Run("callout invalid variant", func(t *testing.T) {
		t.Parallel()
		node := fmt.Sprintf(`{"type":"callout","attrs":{"id":%q,"variant":"danger"},"content":[]}`, testUUID(1))
		_, errs := mustNormalize(t, Blog, blogDoc(node))
		requireFieldError(t, errs, "body.content[0].attrs.variant", "not_allowed")
	})

	t.Run("blockquote contains paragraph", func(t *testing.T) {
		t.Parallel()
		node := fmt.Sprintf(`{"type":"blockquote","attrs":{"id":%q},"content":[%s]}`, testUUID(1), para(2, "Kutipan"))
		res, errs := mustNormalize(t, Blog, blogDoc(node))
		requireNoErrors(t, errs)
		if !strings.Contains(res.PlainText, "Kutipan") {
			t.Fatalf("PlainText = %q", res.PlainText)
		}
	})

	t.Run("container rejects image", func(t *testing.T) {
		t.Parallel()
		node := fmt.Sprintf(`{"type":"blockquote","attrs":{"id":%q},"content":[{"type":"image"}]}`, testUUID(1))
		_, errs := mustNormalize(t, Blog, blogDoc(node))
		requireFieldError(t, errs, "body.content[0].content[0]", "not_allowed")
	})

	t.Run("container child not object", func(t *testing.T) {
		t.Parallel()
		node := fmt.Sprintf(`{"type":"blockquote","attrs":{"id":%q},"content":[1]}`, testUUID(1))
		_, errs := mustNormalize(t, Blog, blogDoc(node))
		requireFieldError(t, errs, "body.content[0].content[0]", "invalid_type")
	})

	t.Run("container content not array", func(t *testing.T) {
		t.Parallel()
		node := fmt.Sprintf(`{"type":"blockquote","attrs":{"id":%q},"content":{}}`, testUUID(1))
		_, errs := mustNormalize(t, Blog, blogDoc(node))
		requireFieldError(t, errs, "body.content[0].content", "invalid_type")
	})
}

func TestNormalize_BlogImage(t *testing.T) {
	t.Parallel()

	imageWith := func(attrs string) string {
		return blogDoc(fmt.Sprintf(`{"type":"image","attrs":%s}`, attrs))
	}

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		raw := imageWith(fmt.Sprintf(`{"id":%q,"media_id":%q,"alt":"Alt","caption":"Cap","fit":"wide"}`, testUUID(1), testUUID(100)))
		res, errs := mustNormalize(t, Blog, raw)
		requireNoErrors(t, errs)
		if len(res.MediaIDs) != 1 || res.MediaIDs[0] != uuid.MustParse(testUUID(100)) {
			t.Fatalf("MediaIDs = %v", res.MediaIDs)
		}
		// Captions are not text nodes, so they are not part of blog PlainText.
		if res.PlainText != "" {
			t.Fatalf("PlainText = %q", res.PlainText)
		}
	})

	tests := []struct {
		name  string
		attrs string
		field string
		code  string
	}{
		{"invalid fit", fmt.Sprintf(`{"id":%q,"fit":"cover"}`, testUUID(1)), "body.content[0].attrs.fit", "not_allowed"},
		{"alt too long", fmt.Sprintf(`{"id":%q,"alt":%q}`, testUUID(1), strings.Repeat("a", 201)), "body.content[0].attrs.alt", "too_long"},
		{"caption too long", fmt.Sprintf(`{"id":%q,"caption":%q}`, testUUID(1), strings.Repeat("c", 301)), "body.content[0].attrs.caption", "too_long"},
		{"invalid media", fmt.Sprintf(`{"id":%q,"media_id":"x"}`, testUUID(1)), "body.content[0].attrs.media_id", "invalid_id"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, errs := mustNormalize(t, Blog, imageWith(tc.attrs))
			requireFieldError(t, errs, tc.field, tc.code)
		})
	}

	t.Run("submit requires media", func(t *testing.T) {
		t.Parallel()
		_, errs := mustSubmit(t, Blog, imageWith(fmt.Sprintf(`{"id":%q}`, testUUID(1))))
		requireFieldError(t, errs, "body.content[0].attrs.media_id", "required")
	})
}

func TestNormalize_BlogEmbed(t *testing.T) {
	t.Parallel()

	embedNode := func(attrs string) string {
		return blogDoc(fmt.Sprintf(`{"type":"embed","attrs":%s}`, attrs))
	}

	t.Run("youtube watch canonicalized", func(t *testing.T) {
		t.Parallel()
		res, errs := mustNormalize(t, Blog, embedNode(fmt.Sprintf(`{"id":%q,"provider":"youtube","url":"https://www.youtube.com/watch?v=dQw4w9WgXcQ","aspect":"9:16"}`, testUUID(1))))
		requireNoErrors(t, errs)
		attrs := firstNodeAttrs(t, res)
		if attrs["url"] != "https://www.youtube.com/watch?v=dQw4w9WgXcQ" || attrs["video_id"] != "dQw4w9WgXcQ" {
			t.Fatalf("attrs = %+v", attrs)
		}
		if attrs["aspect"] != "9:16" {
			t.Fatalf("aspect = %+v", attrs["aspect"])
		}
	})

	t.Run("youtube short url", func(t *testing.T) {
		t.Parallel()
		res, errs := mustNormalize(t, Blog, embedNode(fmt.Sprintf(`{"id":%q,"provider":"youtube","url":"https://youtu.be/dQw4w9WgXcQ"}`, testUUID(1))))
		requireNoErrors(t, errs)
		if firstNodeAttrs(t, res)["video_id"] != "dQw4w9WgXcQ" {
			t.Fatalf("attrs = %+v", firstNodeAttrs(t, res))
		}
	})

	t.Run("youtube video id fallback", func(t *testing.T) {
		t.Parallel()
		res, errs := mustNormalize(t, Blog, embedNode(fmt.Sprintf(`{"id":%q,"provider":"youtube","video_id":"dQw4w9WgXcQ"}`, testUUID(1))))
		requireNoErrors(t, errs)
		attrs := firstNodeAttrs(t, res)
		if attrs["url"] != "https://www.youtube.com/watch?v=dQw4w9WgXcQ" {
			t.Fatalf("attrs = %+v", attrs)
		}
	})

	t.Run("vimeo video id fallback", func(t *testing.T) {
		t.Parallel()
		res, errs := mustNormalize(t, Blog, embedNode(fmt.Sprintf(`{"id":%q,"provider":"vimeo","video_id":"123456"}`, testUUID(1))))
		requireNoErrors(t, errs)
		if firstNodeAttrs(t, res)["url"] != "https://vimeo.com/123456" {
			t.Fatalf("attrs = %+v", firstNodeAttrs(t, res))
		}
	})

	tests := []struct {
		name  string
		attrs string
		field string
		code  string
	}{
		{"missing provider", fmt.Sprintf(`{"id":%q,"url":"https://vimeo.com/123"}`, testUUID(1)), "body.content[0].attrs.provider", "required"},
		{"invalid provider", fmt.Sprintf(`{"id":%q,"provider":"tiktok","url":"https://vimeo.com/123"}`, testUUID(1)), "body.content[0].attrs.provider", "not_allowed"},
		{"missing url", fmt.Sprintf(`{"id":%q,"provider":"figma"}`, testUUID(1)), "body.content[0].attrs.url", "required"},
		{"invalid url", fmt.Sprintf(`{"id":%q,"provider":"figma","url":"https://evil.com/x"}`, testUUID(1)), "body.content[0].attrs.url", "invalid_url"},
		{"provider mismatch", fmt.Sprintf(`{"id":%q,"provider":"figma","url":"https://vimeo.com/123"}`, testUUID(1)), "body.content[0].attrs.provider", "not_allowed"},
		{"invalid video id", fmt.Sprintf(`{"id":%q,"provider":"youtube","video_id":"short"}`, testUUID(1)), "body.content[0].attrs.video_id", "invalid_id"},
		{"invalid aspect", fmt.Sprintf(`{"id":%q,"provider":"figma","url":"https://www.figma.com/file/AbC123","aspect":"21:9"}`, testUUID(1)), "body.content[0].attrs.aspect", "not_allowed"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, errs := mustNormalize(t, Blog, embedNode(tc.attrs))
			requireFieldError(t, errs, tc.field, tc.code)
		})
	}
}

func firstNodeAttrs(t *testing.T, res Result) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(res.Body, &doc); err != nil {
		t.Fatalf("invalid body: %v", err)
	}
	nodes := doc["content"].([]any)
	return nodes[0].(map[string]any)["attrs"].(map[string]any)
}

func TestNormalize_BlogPortfolioCard(t *testing.T) {
	t.Parallel()

	card := func(publicID string) string {
		return blogDoc(fmt.Sprintf(`{"type":"portfolioCard","attrs":{"id":%q,"public_id":%q}}`, testUUID(1), publicID))
	}

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		res, errs := mustNormalize(t, Blog, card("abc12345"))
		requireNoErrors(t, errs)
		if firstNodeAttrs(t, res)["public_id"] != "abc12345" {
			t.Fatalf("attrs = %+v", firstNodeAttrs(t, res))
		}
	})

	tests := []struct {
		name     string
		publicID string
		code     string
	}{
		{"uppercase", "ABC12345", "invalid_value"},
		{"too short", "abc1234", "invalid_value"},
		{"too long", "abc123456", "invalid_value"},
		{"symbols", "abc-2345", "invalid_value"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, errs := mustNormalize(t, Blog, card(tc.publicID))
			requireFieldError(t, errs, "body.content[0].attrs.public_id", tc.code)
		})
	}

	t.Run("missing", func(t *testing.T) {
		t.Parallel()
		node := fmt.Sprintf(`{"type":"portfolioCard","attrs":{"id":%q}}`, testUUID(1))
		_, errs := mustNormalize(t, Blog, blogDoc(node))
		requireFieldError(t, errs, "body.content[0].attrs.public_id", "required")
	})
}

func TestNormalize_BlogTable(t *testing.T) {
	t.Parallel()

	cell := func(kind, text string) string {
		return fmt.Sprintf(`{"type":%q,"content":[{"type":"paragraph","content":[{"type":"text","text":%q}]}]}`, kind, text)
	}
	row := func(cells ...string) string {
		return `{"type":"tableRow","content":[` + strings.Join(cells, ",") + `]}`
	}
	table := func(rows string) string {
		return blogDoc(fmt.Sprintf(`{"type":"table","attrs":{"id":%q},"content":[%s]}`, testUUID(1), rows))
	}

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		res, errs := mustNormalize(t, Blog, table(row(cell("tableHeader", "A"), cell("tableHeader", "B"))+","+row(cell("tableCell", "1"), cell("tableCell", "2"))))
		requireNoErrors(t, errs)
		if res.PlainText != "A B 1 2" {
			t.Fatalf("PlainText = %q", res.PlainText)
		}
	})

	t.Run("empty table", func(t *testing.T) {
		t.Parallel()
		_, errs := mustNormalize(t, Blog, table(""))
		requireNoErrors(t, errs)
	})

	cells := func(n int, kind, text string) string {
		parts := make([]string, n)
		for i := range parts {
			parts[i] = cell(kind, text)
		}
		return strings.Join(parts, ",")
	}

	tests := []struct {
		name  string
		raw   string
		field string
		code  string
	}{
		{"missing content", blogDoc(fmt.Sprintf(`{"type":"table","attrs":{"id":%q}}`, testUUID(1))), "body.content[0].content", "required"},
		{"content not array", blogDoc(fmt.Sprintf(`{"type":"table","attrs":{"id":%q},"content":{}}`, testUUID(1))), "body.content[0].content", "invalid_type"},
		{"row not object", table("1"), "body.content[0].content[0]", "invalid_type"},
		{"row wrong type", table(fmt.Sprintf(`{"type":"paragraph","attrs":{"id":%q}}`, testUUID(2))), "body.content[0].content[0]", "not_allowed"},
		{"row content not array", table(fmt.Sprintf(`{"type":"tableRow","content":{}}`)), "body.content[0].content[0].content", "invalid_type"},
		{"too many columns", table(row(cells(11, "tableCell", "x"))), "body.content[0].content[0].content", "too_many"},
		{"cell not object", table(`{"type":"tableRow","content":[1]}`), "body.content[0].content[0].content[0]", "invalid_type"},
		{"cell wrong type", table(row(fmt.Sprintf(`{"type":"paragraph","attrs":{"id":%q}}`, testUUID(2)))), "body.content[0].content[0].content[0]", "not_allowed"},
		{"cell content not array", table(`{"type":"tableRow","content":[{"type":"tableCell","content":{}}]}`), "body.content[0].content[0].content[0].content", "invalid_type"},
		{"cell child not object", table(`{"type":"tableRow","content":[{"type":"tableCell","content":[1]}]}`), "body.content[0].content[0].content[0].content[0]", "invalid_type"},
		{"cell child not paragraph", table(fmt.Sprintf(`{"type":"tableRow","content":[{"type":"tableCell","content":[{"type":"heading","attrs":{"level":2}}]}]}`)), "body.content[0].content[0].content[0].content[0]", "not_allowed"},
		{"cell too long", table(row(cell("tableCell", strings.Repeat("c", 201)))), "body.content[0].content[0].content[0]", "too_long"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, errs := mustNormalize(t, Blog, tc.raw)
			requireFieldError(t, errs, tc.field, tc.code)
		})
	}

	t.Run("too many rows", func(t *testing.T) {
		t.Parallel()
		rows := make([]string, 51)
		for i := range rows {
			rows[i] = row(cell("tableCell", "x"))
		}
		_, errs := mustNormalize(t, Blog, table(strings.Join(rows, ",")))
		requireFieldError(t, errs, "body.content[0].content", "too_many")
	})

	t.Run("limits accepted exactly", func(t *testing.T) {
		t.Parallel()
		rows := make([]string, 50)
		for i := range rows {
			rows[i] = row(cells(10, "tableCell", "x"))
		}
		_, errs := mustNormalize(t, Blog, table(strings.Join(rows, ",")))
		requireNoErrors(t, errs)
	})
}

func TestNormalize_BlogHorizontalRule(t *testing.T) {
	t.Parallel()
	node := fmt.Sprintf(`{"type":"horizontalRule","attrs":{"id":%q}}`, testUUID(1))
	res, errs := mustNormalize(t, Blog, blogDoc(node))
	requireNoErrors(t, errs)
	if !strings.Contains(string(res.Body), `"type":"horizontalRule"`) {
		t.Fatalf("Body = %s", res.Body)
	}
}

func TestNormalize_BlogPlainTextAndHeadings(t *testing.T) {
	t.Parallel()
	raw := blogDoc(
		heading(1, 2, "Persiapan"),
		para(2, "Halo dunia"),
		fmt.Sprintf(`{"type":"bulletList","attrs":{"id":%q},"content":[{"type":"listItem","content":[%s]}]}`, testUUID(3), para(4, "Satu")),
		fmt.Sprintf(`{"type":"codeBlock","attrs":{"id":%q,"language":"go"},"content":[{"type":"text","text":"func main()"}]}`, testUUID(5)),
		fmt.Sprintf(`{"type":"image","attrs":{"id":%q,"media_id":%q,"caption":"tidak ikut"}}`, testUUID(6), testUUID(100)),
	)
	res, errs := mustNormalize(t, Blog, raw)
	requireNoErrors(t, errs)
	want := "Persiapan Halo dunia Satu func main()"
	if res.PlainText != want {
		t.Fatalf("PlainText = %q, want %q", res.PlainText, want)
	}
	if strings.Contains(res.PlainText, "tidak ikut") {
		t.Fatalf("image caption leaked into PlainText: %q", res.PlainText)
	}
	if len(res.Headings) != 1 || res.Headings[0].Text != "Persiapan" {
		t.Fatalf("Headings = %+v", res.Headings)
	}
}

func TestNormalize_BlogReadingMinutes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		words int
		want  int
	}{
		{0, 1},
		{1, 1},
		{200, 1},
		{201, 2},
		{400, 2},
		{401, 3},
	}
	for _, tc := range tests {
		t.Run(fmt.Sprintf("%d_words", tc.words), func(t *testing.T) {
			t.Parallel()
			res, errs := mustNormalize(t, Blog, blogDoc(para(1, wordsText(tc.words))))
			requireNoErrors(t, errs)
			if res.ReadingMinutes != tc.want {
				t.Fatalf("ReadingMinutes(%d words) = %d, want %d", tc.words, res.ReadingMinutes, tc.want)
			}
		})
	}
}

func wordsText(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat("kata ", n-1) + "kata"
}

func TestValidateForSubmit_BlogWordLimits(t *testing.T) {
	t.Parallel()

	t.Run("149 words rejected", func(t *testing.T) {
		t.Parallel()
		_, errs := mustSubmit(t, Blog, blogDoc(para(1, wordsText(149))))
		requireFieldError(t, errs, "body", "too_few_words")
	})

	t.Run("150 words accepted", func(t *testing.T) {
		t.Parallel()
		_, errs := mustSubmit(t, Blog, blogDoc(para(1, wordsText(150))))
		requireNoErrors(t, errs)
	})

	t.Run("20000 words accepted", func(t *testing.T) {
		t.Parallel()
		_, errs := mustSubmit(t, Blog, blogDoc(para(1, wordsText(20000))))
		requireNoErrors(t, errs)
	})

	t.Run("20001 words rejected", func(t *testing.T) {
		t.Parallel()
		_, errs := mustSubmit(t, Blog, blogDoc(para(1, wordsText(20001))))
		requireFieldError(t, errs, "body", "too_long")
	})
}

func TestValidateForSubmit_BlogCounts(t *testing.T) {
	t.Parallel()

	t.Run("30 images accepted", func(t *testing.T) {
		t.Parallel()
		parts := make([]string, 30)
		for i := range parts {
			parts[i] = fmt.Sprintf(`{"type":"image","attrs":{"id":%q,"media_id":%q}}`, testUUID(i+1), testUUID(1000+i))
		}
		_, errs := mustSubmit(t, Blog, blogDoc(parts...))
		if hasCode(errs, "too_many") {
			t.Fatalf("unexpected too_many: %+v", errs)
		}
	})

	t.Run("31 images rejected", func(t *testing.T) {
		t.Parallel()
		parts := make([]string, 31)
		for i := range parts {
			parts[i] = fmt.Sprintf(`{"type":"image","attrs":{"id":%q,"media_id":%q}}`, testUUID(i+1), testUUID(1000+i))
		}
		_, errs := mustSubmit(t, Blog, blogDoc(parts...))
		requireFieldError(t, errs, "body.content", "too_many")
	})

	t.Run("15 embeds accepted", func(t *testing.T) {
		t.Parallel()
		parts := make([]string, 15)
		for i := range parts {
			parts[i] = fmt.Sprintf(`{"type":"embed","attrs":{"id":%q,"provider":"vimeo","video_id":%q}}`, testUUID(i+1), fmt.Sprintf("%d", 1000+i))
		}
		_, errs := mustSubmit(t, Blog, blogDoc(parts...))
		if hasCode(errs, "too_many") {
			t.Fatalf("unexpected too_many: %+v", errs)
		}
	})

	t.Run("16 embeds rejected", func(t *testing.T) {
		t.Parallel()
		parts := make([]string, 16)
		for i := range parts {
			parts[i] = fmt.Sprintf(`{"type":"embed","attrs":{"id":%q,"provider":"vimeo","video_id":%q}}`, testUUID(i+1), fmt.Sprintf("%d", 1000+i))
		}
		_, errs := mustSubmit(t, Blog, blogDoc(parts...))
		requireFieldError(t, errs, "body.content", "too_many")
	})
}

func TestNormalize_BlogMediaIDs(t *testing.T) {
	t.Parallel()
	raw := blogDoc(
		imageNode(1, 110),
		imageNode(2, 101),
		imageNode(3, 110),
	)
	res, errs := mustNormalize(t, Blog, raw)
	requireNoErrors(t, errs)
	if len(res.MediaIDs) != 2 {
		t.Fatalf("MediaIDs = %v", res.MediaIDs)
	}
	if res.MediaIDs[0] != uuid.MustParse(testUUID(101)) || res.MediaIDs[1] != uuid.MustParse(testUUID(110)) {
		t.Fatalf("MediaIDs not sorted/deduped: %v", res.MediaIDs)
	}
}

func TestNormalize_BlogSanitizesText(t *testing.T) {
	t.Parallel()
	node := fmt.Sprintf(`{"type":"paragraph","attrs":{"id":%q},"content":[{"type":"text","text":"a\u0000b\u0007c\u007f\nd\te"}]}`, testUUID(1))
	res, errs := mustNormalize(t, Blog, blogDoc(node))
	requireNoErrors(t, errs)
	if !strings.Contains(string(res.Body), `abc\nd\te`) {
		t.Fatalf("Body = %s", res.Body)
	}
	if res.PlainText != "abc d e" {
		t.Fatalf("PlainText = %q", res.PlainText)
	}
}
