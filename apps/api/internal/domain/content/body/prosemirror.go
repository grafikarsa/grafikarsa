package body

import (
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"
)

var (
	textBlockMarks = map[string]struct{}{
		"bold": {}, "italic": {}, "link": {}, "code": {},
	}
	blogMarks = map[string]struct{}{
		"bold": {}, "italic": {}, "strike": {}, "code": {}, "link": {},
	}
	allMarkTypes = map[string]struct{}{
		"bold": {}, "italic": {}, "strike": {}, "code": {}, "link": {},
	}
)

type inlineResult struct {
	content []any
	text    string
	chars   int
}

func (v *validator) inlineContent(raw any, path, ref string, allowedMarks map[string]struct{}) inlineResult {
	result := inlineResult{content: []any{}}
	if raw == nil {
		return result
	}
	nodes, ok := asArray(raw)
	if !ok {
		v.add(path, ref, "invalid_type", "Isi paragraf harus berupa array.")
		return result
	}

	var plain strings.Builder
	for i, rawNode := range nodes {
		nodePath := fmt.Sprintf("%s[%d]", path, i)
		obj, ok := asObject(rawNode)
		if !ok {
			v.add(nodePath, ref, "invalid_type", "Node teks harus berupa objek JSON.")
			continue
		}
		typeName, ok := asString(obj["type"])
		if !ok {
			v.add(nodePath+".type", ref, "unknown_type", "Jenis node tidak dikenal.")
			continue
		}

		switch typeName {
		case "text":
			textValue, ok := asString(obj["text"])
			if !ok {
				v.add(nodePath+".text", ref, "invalid_type", "Teks node harus berupa teks.")
				continue
			}
			textValue = sanitizeText(textValue)
			textNode := map[string]any{"type": "text", "text": textValue}
			if marks := v.markList(obj["marks"], nodePath+".marks", ref, allowedMarks); len(marks) > 0 {
				textNode["marks"] = marks
			}
			result.content = append(result.content, textNode)
			result.chars += utf8.RuneCountInString(textValue)
			plain.WriteString(textValue)
		case "hardBreak":
			result.content = append(result.content, map[string]any{"type": "hardBreak"})
			plain.WriteString(" ")
		default:
			v.invalidNodeType(nodePath, ref, typeName, "Node ini tidak diizinkan di dalam paragraf.")
		}
	}
	result.text = plain.String()
	return result
}

func (v *validator) markList(raw any, path, ref string, allowed map[string]struct{}) []any {
	if raw == nil {
		return nil
	}
	rawMarks, ok := asArray(raw)
	if !ok {
		v.add(path, ref, "invalid_type", "Marks harus berupa array.")
		return nil
	}
	marks := make([]any, 0, len(rawMarks))
	for i, rawMark := range rawMarks {
		markPath := fmt.Sprintf("%s[%d]", path, i)
		obj, ok := asObject(rawMark)
		if !ok {
			v.add(markPath, ref, "invalid_type", "Mark harus berupa objek JSON.")
			continue
		}
		typeName, ok := asString(obj["type"])
		if !ok {
			v.add(markPath+".type", ref, "unknown_type", "Jenis mark tidak dikenal.")
			continue
		}
		if _, known := allMarkTypes[typeName]; !known {
			v.add(markPath+".type", ref, "unknown_type", "Jenis mark tidak dikenal.")
			continue
		}
		if _, ok := allowed[typeName]; !ok {
			v.add(markPath+".type", ref, "not_allowed", "Mark ini tidak diizinkan di sini.")
			continue
		}

		mark := map[string]any{"type": typeName}
		if typeName == "link" {
			attrs, _ := v.objectField(obj, "attrs", markPath+".attrs", ref, true)
			href, ok := v.stringField(attrs, "href", markPath+".attrs.href", ref, 0, false, true)
			if !ok {
				continue
			}
			if !validLinkHref(href) {
				v.add(markPath+".attrs.href", ref, "invalid_url", "Tautan harus https:// atau path internal.")
				continue
			}
			mark["attrs"] = map[string]any{"href": href}
		}
		marks = append(marks, mark)
	}
	return marks
}

func validLinkHref(href string) bool {
	if href == "" || strings.ContainsAny(href, " \t\r\n") {
		return false
	}
	if strings.HasPrefix(href, "/") {
		return !strings.HasPrefix(href, "//")
	}
	return isHTTPSURL(href)
}

func isHTTPSURL(raw string) bool {
	if raw == "" || strings.ContainsAny(raw, " \t\r\n") {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil {
		return false
	}
	return strings.EqualFold(u.Scheme, "https")
}

// invalidNodeType reports unknown_type for a truly unknown node and
// not_allowed for a known node used in the wrong position.
func (v *validator) invalidNodeType(path, ref, typeName, notAllowedMessage string) {
	if _, known := knownBlogNodes[typeName]; known {
		v.add(path, ref, "not_allowed", notAllowedMessage)
		return
	}
	v.add(path+".type", ref, "unknown_type", "Jenis node tidak dikenal.")
}

// textDoc validates the limited ProseMirror subset used by portfolio text
// blocks (paragraph, H2-H3, lists, hardBreak, and inline marks).
func (v *validator) textDoc(obj map[string]any, path, ref string) map[string]any {
	doc := map[string]any{"type": "doc", "content": []any{}}

	docType, ok := asString(obj["type"])
	if !ok || docType != "doc" {
		v.add(path+".type", ref, "invalid_type", "Dokumen teks harus bertipe \"doc\".")
	}

	rawContent, exists := obj["content"]
	var contentNodes []any
	if exists && rawContent != nil {
		var ok bool
		contentNodes, ok = asArray(rawContent)
		if !ok {
			v.add(path+".content", ref, "invalid_type", "Isi dokumen harus berupa array.")
			contentNodes = nil
		}
	}

	nodes := make([]any, 0, len(contentNodes))
	chars := 0
	for i, rawNode := range contentNodes {
		nodePath := fmt.Sprintf("%s.content[%d]", path, i)
		obj, ok := asObject(rawNode)
		if !ok {
			v.add(nodePath, ref, "invalid_type", "Node harus berupa objek JSON.")
			continue
		}
		typeName, _ := asString(obj["type"])
		switch typeName {
		case "paragraph":
			res := v.inlineContent(obj["content"], nodePath+".content", ref, textBlockMarks)
			chars += res.chars
			v.addText(res.text)
			nodes = append(nodes, map[string]any{
				"type":    "paragraph",
				"content": res.content,
			})
		case "heading":
			attrs, _ := v.objectField(obj, "attrs", nodePath+".attrs", ref, true)
			level, ok := v.headingLevel(attrs, nodePath, ref, 2, 3)
			res := v.inlineContent(obj["content"], nodePath+".content", ref, textBlockMarks)
			chars += res.chars
			v.addText(res.text)
			node := map[string]any{"type": "heading", "content": res.content}
			if ok {
				node["attrs"] = map[string]any{"level": level}
			}
			nodes = append(nodes, node)
		case "bulletList", "orderedList":
			listNodes, listChars := v.textDocList(obj, nodePath, ref, typeName)
			chars += listChars
			nodes = append(nodes, listNodes...)
		case "listItem", "hardBreak", "text":
			v.add(nodePath, ref, "not_allowed", "Node ini tidak diizinkan pada level atas dokumen teks.")
		default:
			v.invalidNodeType(nodePath, ref, typeName, "Node ini tidak diizinkan pada level atas dokumen teks.")
		}
	}
	if chars > maxTextBlockRunes {
		v.add(path+".content", ref, "too_long", fmt.Sprintf("Teks blok maksimal %d karakter.", maxTextBlockRunes))
	}
	doc["content"] = nodes
	return doc
}

func (v *validator) textDocList(obj map[string]any, path, ref, listType string) ([]any, int) {
	node := map[string]any{"type": listType, "content": []any{}}
	rawItems, ok := asArray(obj["content"])
	if !ok {
		if obj["content"] != nil {
			v.add(path+".content", ref, "invalid_type", "Isi daftar harus berupa array.")
		}
		return []any{node}, 0
	}
	items := make([]any, 0, len(rawItems))
	chars := 0
	for i, rawItem := range rawItems {
		itemPath := fmt.Sprintf("%s.content[%d]", path, i)
		itemObj, ok := asObject(rawItem)
		if !ok {
			v.add(itemPath, ref, "invalid_type", "Item daftar harus berupa objek JSON.")
			continue
		}
		itemType, _ := asString(itemObj["type"])
		if itemType != "listItem" {
			v.invalidNodeType(itemPath, ref, itemType, "Daftar hanya boleh berisi item daftar.")
			continue
		}
		content, itemChars := v.textDocListItemContent(itemObj, itemPath, ref)
		chars += itemChars
		item := map[string]any{"type": "listItem", "content": content}
		items = append(items, item)
	}
	node["content"] = items
	return []any{node}, chars
}

func (v *validator) textDocListItemContent(obj map[string]any, path, ref string) ([]any, int) {
	rawContent, ok := asArray(obj["content"])
	if !ok {
		if obj["content"] != nil {
			v.add(path+".content", ref, "invalid_type", "Isi item daftar harus berupa array.")
		}
		return []any{}, 0
	}
	nodes := make([]any, 0, len(rawContent))
	chars := 0
	for i, rawNode := range rawContent {
		nodePath := fmt.Sprintf("%s.content[%d]", path, i)
		child, ok := asObject(rawNode)
		if !ok {
			v.add(nodePath, ref, "invalid_type", "Node harus berupa objek JSON.")
			continue
		}
		typeName, _ := asString(child["type"])
		switch typeName {
		case "paragraph":
			res := v.inlineContent(child["content"], nodePath+".content", ref, textBlockMarks)
			chars += res.chars
			v.addText(res.text)
			nodes = append(nodes, map[string]any{"type": "paragraph", "content": res.content})
		case "bulletList", "orderedList":
			listNodes, listChars := v.textDocList(child, nodePath, ref, typeName)
			chars += listChars
			nodes = append(nodes, listNodes...)
		default:
			v.invalidNodeType(nodePath, ref, typeName, "Item daftar hanya boleh berisi paragraf atau daftar.")
		}
	}
	return nodes, chars
}
