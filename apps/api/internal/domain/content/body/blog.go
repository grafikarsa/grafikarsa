package body

import (
	"fmt"
	"regexp"
)

var (
	knownBlogNodes = map[string]struct{}{
		"paragraph": {}, "heading": {}, "bulletList": {}, "orderedList": {},
		"listItem": {}, "blockquote": {}, "codeBlock": {}, "callout": {},
		"image": {}, "embed": {}, "portfolioCard": {}, "table": {},
		"tableRow": {}, "tableCell": {}, "tableHeader": {}, "horizontalRule": {},
		"hardBreak": {}, "text": {},
	}
	containerBlockTypes = map[string]struct{}{
		"paragraph": {}, "heading": {}, "bulletList": {}, "orderedList": {},
		"blockquote": {}, "codeBlock": {},
	}
	calloutVariants = map[string]struct{}{
		"info": {}, "tip": {}, "warning": {},
	}
	publicIDPattern = regexp.MustCompile(`^[0-9a-z]{8}$`)
)

func (v *validator) blog(root any) map[string]any {
	obj, ok := asObject(root)
	if !ok {
		v.add("body", "", "invalid_type", "Body blog harus berupa objek JSON.")
		return nil
	}

	out := map[string]any{"type": "doc", "content": []any{}}
	if docType, ok := asString(obj["type"]); !ok || docType != "doc" {
		v.add("body.type", "", "invalid_type", "Dokumen harus bertipe \"doc\".")
	}

	rawContent, ok := v.arrayField(obj, "content", "body.content", "", true)
	if !ok {
		return out
	}

	seen := make(map[string]bool, len(rawContent))
	nodes := make([]any, 0, len(rawContent))
	for i, rawNode := range rawContent {
		nodePath := fmt.Sprintf("body.content[%d]", i)
		nodeObj, ok := asObject(rawNode)
		if !ok {
			v.add(nodePath, "", "invalid_type", "Node harus berupa objek JSON.")
			continue
		}
		if node := v.blogBlock(nodeObj, nodePath, true, seen, 0); node != nil {
			nodes = append(nodes, node)
		}
	}
	out["content"] = nodes

	if v.submit {
		if v.imageCount > maxBlogImages {
			v.add("body.content", "", "too_many", fmt.Sprintf("Blog maksimal %d gambar.", maxBlogImages))
		}
		if v.embedCount > maxBlogEmbeds {
			v.add("body.content", "", "too_many", fmt.Sprintf("Blog maksimal %d embed.", maxBlogEmbeds))
		}
	}
	return out
}

func (v *validator) blogBlock(obj map[string]any, path string, requireID bool, seen map[string]bool, listDepth int) map[string]any {
	attrs, _ := v.objectField(obj, "attrs", path+".attrs", "", false)
	id, _ := v.nodeIDField(attrs, path+".attrs.id", "", requireID, seen)
	ref := id

	typeName, ok := asString(obj["type"])
	if !ok {
		v.add(path+".type", ref, "unknown_type", "Jenis node tidak dikenal.")
		return nil
	}
	if _, known := knownBlogNodes[typeName]; !known {
		v.add(path+".type", ref, "unknown_type", "Jenis node tidak dikenal.")
		return nil
	}

	switch typeName {
	case "paragraph":
		res := v.inlineContent(obj["content"], path+".content", ref, blogMarks)
		v.addText(res.text)
		node := map[string]any{
			"type":    "paragraph",
			"content": res.content,
		}
		return nodeWithAttrs(node, attrsWithID(id))
	case "heading":
		level, levelOK := v.headingLevel(attrs, path, ref, 2, 4)
		res := v.inlineContent(obj["content"], path+".content", ref, blogMarks)
		v.addText(res.text)
		attrsOut := attrsWithID(id)
		if levelOK {
			attrsOut["level"] = level
			if level == 2 || level == 3 {
				v.addHeading(id, level, collapseSpace(res.text))
			}
		}
		return map[string]any{"type": "heading", "attrs": attrsOut, "content": res.content}
	case "bulletList", "orderedList":
		return v.blogList(obj, path, ref, id, typeName, listDepth)
	case "blockquote":
		node := map[string]any{
			"type":    "blockquote",
			"content": v.containerContent(obj, path, ref, listDepth),
		}
		return nodeWithAttrs(node, attrsWithID(id))
	case "callout":
		attrsOut := attrsWithID(id)
		if variant, ok := v.enumField(attrs, "variant", path+".attrs.variant", ref, calloutVariants, "info", false); ok {
			attrsOut["variant"] = variant
		}
		return map[string]any{
			"type":    "callout",
			"attrs":   attrsOut,
			"content": v.containerContent(obj, path, ref, listDepth),
		}
	case "codeBlock":
		return v.blogCodeBlock(obj, attrs, path, ref, id)
	case "image":
		return v.blogImage(attrs, path, ref, id)
	case "embed":
		return v.blogEmbed(attrs, path, ref, id)
	case "portfolioCard":
		attrsOut := attrsWithID(id)
		publicID, ok := v.stringField(attrs, "public_id", path+".attrs.public_id", ref, 0, false, true)
		if ok {
			if !publicIDPattern.MatchString(publicID) {
				v.add(path+".attrs.public_id", ref, "invalid_value", "ID portfolio tidak valid.")
			} else {
				attrsOut["public_id"] = publicID
			}
		}
		return nodeWithAttrs(map[string]any{"type": "portfolioCard"}, attrsOut)
	case "table":
		return v.blogTable(obj, path, ref, id)
	case "horizontalRule":
		return nodeWithAttrs(map[string]any{"type": "horizontalRule"}, attrsWithID(id))
	default:
		v.add(path, ref, "not_allowed", "Node ini tidak diizinkan pada posisi tersebut.")
		return nil
	}
}

func (v *validator) blogList(obj map[string]any, path, ref, id, listType string, depth int) map[string]any {
	out := map[string]any{
		"type":    listType,
		"content": []any{},
	}
	if attrs := attrsWithID(id); len(attrs) > 0 {
		out["attrs"] = attrs
	}
	depth++
	if depth > maxListDepth {
		v.add(path, ref, "too_many", fmt.Sprintf("Daftar maksimal %d tingkat.", maxListDepth))
		return out
	}

	rawItems, ok := v.arrayField(obj, "content", path+".content", ref, false)
	if !ok {
		return out
	}
	items := make([]any, 0, len(rawItems))
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
		items = append(items, v.blogListItem(itemObj, itemPath, ref, depth))
	}
	out["content"] = items
	return out
}

func (v *validator) blogListItem(obj map[string]any, path, ref string, depth int) map[string]any {
	attrs, _ := v.objectField(obj, "attrs", path+".attrs", ref, false)
	id, _ := v.nodeIDField(attrs, path+".attrs.id", ref, false, nil)

	out := map[string]any{"type": "listItem", "content": []any{}}
	if attrsOut := attrsWithID(id); len(attrsOut) > 0 {
		out["attrs"] = attrsOut
	}
	rawContent, ok := asArray(obj["content"])
	if !ok {
		if obj["content"] != nil {
			v.add(path+".content", ref, "invalid_type", "Isi item daftar harus berupa array.")
		}
		return out
	}
	children := make([]any, 0, len(rawContent))
	for i, rawChild := range rawContent {
		childPath := fmt.Sprintf("%s.content[%d]", path, i)
		childObj, ok := asObject(rawChild)
		if !ok {
			v.add(childPath, ref, "invalid_type", "Node harus berupa objek JSON.")
			continue
		}
		childType, _ := asString(childObj["type"])
		switch childType {
		case "paragraph", "bulletList", "orderedList":
			if node := v.blogBlock(childObj, childPath, false, nil, depth); node != nil {
				children = append(children, node)
			}
		default:
			v.invalidNodeType(childPath, ref, childType, "Item daftar hanya boleh berisi paragraf atau daftar.")
		}
	}
	out["content"] = children
	return out
}

func (v *validator) containerContent(obj map[string]any, path, ref string, listDepth int) []any {
	out := []any{}
	rawContent, ok := asArray(obj["content"])
	if !ok {
		if obj["content"] != nil {
			v.add(path+".content", ref, "invalid_type", "Isi node harus berupa array.")
		}
		return out
	}
	for i, rawChild := range rawContent {
		childPath := fmt.Sprintf("%s.content[%d]", path, i)
		childObj, ok := asObject(rawChild)
		if !ok {
			v.add(childPath, ref, "invalid_type", "Node harus berupa objek JSON.")
			continue
		}
		childType, _ := asString(childObj["type"])
		if _, allowed := containerBlockTypes[childType]; !allowed {
			v.invalidNodeType(childPath, ref, childType, "Node ini tidak diizinkan di dalam node ini.")
			continue
		}
		if node := v.blogBlock(childObj, childPath, false, nil, listDepth); node != nil {
			out = append(out, node)
		}
	}
	return out
}

func (v *validator) blogCodeBlock(obj map[string]any, attrs map[string]any, path, ref, id string) map[string]any {
	attrsOut := attrsWithID(id)
	language, ok := v.stringField(attrs, "language", path+".attrs.language", ref, 0, true, false)
	if !ok || language == "" {
		language = "plaintext"
	}
	attrsOut["language"] = language

	content, text, chars := v.codeContent(obj["content"], path+".content", ref)
	if chars > maxCodeRunes {
		v.add(path+".content", ref, "too_long", fmt.Sprintf("Kode maksimal %d karakter.", maxCodeRunes))
	}
	v.addText(text)
	return map[string]any{"type": "codeBlock", "attrs": attrsOut, "content": content}
}

func (v *validator) codeContent(raw any, path, ref string) ([]any, string, int) {
	if raw == nil {
		return []any{}, "", 0
	}
	rawNodes, ok := asArray(raw)
	if !ok {
		v.add(path, ref, "invalid_type", "Isi blok kode harus berupa array.")
		return []any{}, "", 0
	}
	nodes := make([]any, 0, len(rawNodes))
	var text string
	chars := 0
	for i, rawNode := range rawNodes {
		nodePath := fmt.Sprintf("%s[%d]", path, i)
		obj, ok := asObject(rawNode)
		if !ok {
			v.add(nodePath, ref, "invalid_type", "Node teks harus berupa objek JSON.")
			continue
		}
		typeName, _ := asString(obj["type"])
		if typeName != "text" {
			v.invalidNodeType(nodePath, ref, typeName, "Blok kode hanya boleh berisi teks.")
			continue
		}
		value, ok := asString(obj["text"])
		if !ok {
			v.add(nodePath+".text", ref, "invalid_type", "Teks node harus berupa teks.")
			continue
		}
		if marks, ok := asArray(obj["marks"]); ok && len(marks) > 0 {
			v.add(nodePath+".marks", ref, "not_allowed", "Teks blok kode tidak boleh memiliki mark.")
		}
		value = sanitizeText(value)
		chars += len([]rune(value))
		text += value
		nodes = append(nodes, map[string]any{"type": "text", "text": value})
	}
	return nodes, text, chars
}

func (v *validator) blogImage(attrs map[string]any, path, ref, id string) map[string]any {
	v.imageCount++

	out := attrsWithID(id)
	out["media_id"] = nil
	out["fit"] = "contain"
	if mediaID, ok := v.mediaIDField(attrs, "media_id", path+".attrs.media_id", ref, true); ok {
		out["media_id"] = mediaID
	}
	if alt, ok := v.stringField(attrs, "alt", path+".attrs.alt", ref, maxAltRunes, false, false); ok && alt != "" {
		out["alt"] = alt
	}
	if caption, ok := v.stringField(attrs, "caption", path+".attrs.caption", ref, maxCaptionRunes, false, false); ok && caption != "" {
		out["caption"] = caption
	}
	if fit, ok := v.enumField(attrs, "fit", path+".attrs.fit", ref, fitValues, "contain", false); ok {
		out["fit"] = fit
	}
	return map[string]any{"type": "image", "attrs": out}
}

func (v *validator) blogEmbed(attrs map[string]any, path, ref, id string) map[string]any {
	v.embedCount++

	out := attrsWithID(id)
	out["aspect"] = "16:9"

	provider, providerOK := v.enumField(attrs, "provider", path+".attrs.provider", ref, blogEmbedProviders, "", true)
	if providerOK {
		out["provider"] = provider
	}

	rawURL, urlOK := v.stringField(attrs, "url", path+".attrs.url", ref, 0, true, false)
	var videoID string
	var videoIDOK bool
	if providerOK && (provider == "youtube" || provider == "vimeo") {
		videoID, videoIDOK = v.stringField(attrs, "video_id", path+".attrs.video_id", ref, 0, false, false)
	}

	var detected Embed
	embedOK := false
	switch {
	case urlOK:
		detected, embedOK = ParseEmbed(rawURL)
		if !embedOK {
			v.add(path+".attrs.url", ref, "invalid_url", "URL embed tidak dikenali atau tidak diizinkan.")
		}
	case providerOK && (provider == "youtube" || provider == "vimeo") && videoIDOK:
		if vid, ok := videoFromID(provider, videoID); ok {
			detected, embedOK = vid, true
		} else {
			v.add(path+".attrs.video_id", ref, "invalid_id", "ID video tidak valid.")
		}
	default:
		v.add(path+".attrs.url", ref, "required", "URL embed wajib diisi.")
	}

	if embedOK {
		if _, ok := blogEmbedProviders[detected.Provider]; !ok {
			v.add(path+".attrs.provider", ref, "not_allowed", "Provider embed ini tidak diizinkan.")
		} else if providerOK && provider != detected.Provider {
			v.add(path+".attrs.provider", ref, "not_allowed", "Provider tidak cocok dengan URL embed.")
		} else {
			out["provider"] = detected.Provider
			out["url"] = detected.URL
			if detected.VideoID != "" {
				out["video_id"] = detected.VideoID
			}
		}
	}

	if aspect, ok := v.enumField(attrs, "aspect", path+".attrs.aspect", ref, aspectValues, "16:9", false); ok {
		out["aspect"] = aspect
	}
	return map[string]any{"type": "embed", "attrs": out}
}

func videoFromID(provider, id string) (Embed, bool) {
	switch provider {
	case "youtube":
		if !youtubeVideoIDPattern.MatchString(id) {
			return Embed{}, false
		}
		return Embed{Provider: "youtube", URL: "https://www.youtube.com/watch?v=" + id, VideoID: id}, true
	case "vimeo":
		if !vimeoVideoIDPattern.MatchString(id) {
			return Embed{}, false
		}
		return Embed{Provider: "vimeo", URL: "https://vimeo.com/" + id, VideoID: id}, true
	default:
		return Embed{}, false
	}
}

func (v *validator) blogTable(obj map[string]any, path, ref, id string) map[string]any {
	out := map[string]any{
		"type":    "table",
		"content": []any{},
	}
	if attrs := attrsWithID(id); len(attrs) > 0 {
		out["attrs"] = attrs
	}
	rawRows, ok := v.arrayField(obj, "content", path+".content", ref, true)
	if !ok {
		return out
	}
	if len(rawRows) > maxTableRows {
		v.add(path+".content", ref, "too_many", fmt.Sprintf("Tabel maksimal %d baris.", maxTableRows))
		rawRows = rawRows[:maxTableRows]
	}

	rows := make([]any, 0, len(rawRows))
	for r, rawRow := range rawRows {
		rowPath := fmt.Sprintf("%s.content[%d]", path, r)
		rowObj, ok := asObject(rawRow)
		if !ok {
			v.add(rowPath, ref, "invalid_type", "Baris tabel harus berupa objek JSON.")
			continue
		}
		rowType, _ := asString(rowObj["type"])
		if rowType != "tableRow" {
			v.invalidNodeType(rowPath, ref, rowType, "Isi tabel harus berupa baris tabel.")
			continue
		}

		rawCells, ok := asArray(rowObj["content"])
		if !ok {
			if rowObj["content"] != nil {
				v.add(rowPath+".content", ref, "invalid_type", "Isi baris harus berupa array.")
			}
			rows = append(rows, map[string]any{"type": "tableRow", "content": []any{}})
			continue
		}
		if len(rawCells) > maxTableColumns {
			v.add(rowPath+".content", ref, "too_many", fmt.Sprintf("Tabel maksimal %d kolom.", maxTableColumns))
			rawCells = rawCells[:maxTableColumns]
		}

		cells := make([]any, 0, len(rawCells))
		for c, rawCell := range rawCells {
			cellPath := fmt.Sprintf("%s.content[%d]", rowPath, c)
			cellObj, ok := asObject(rawCell)
			if !ok {
				v.add(cellPath, ref, "invalid_type", "Sel tabel harus berupa objek JSON.")
				continue
			}
			cellType, _ := asString(cellObj["type"])
			if cellType != "tableCell" && cellType != "tableHeader" {
				v.invalidNodeType(cellPath, ref, cellType, "Isi baris harus berupa sel tabel.")
				continue
			}
			cells = append(cells, v.blogTableCell(cellObj, cellPath, ref, cellType))
		}
		rows = append(rows, map[string]any{"type": "tableRow", "content": cells})
	}
	out["content"] = rows
	return out
}

func (v *validator) blogTableCell(obj map[string]any, path, ref, cellType string) map[string]any {
	cell := map[string]any{"type": cellType, "content": []any{}}
	rawContent, ok := asArray(obj["content"])
	if !ok {
		if obj["content"] != nil {
			v.add(path+".content", ref, "invalid_type", "Isi sel harus berupa array.")
		}
		return cell
	}

	paragraphs := make([]any, 0, len(rawContent))
	chars := 0
	var textParts []string
	for i, rawChild := range rawContent {
		childPath := fmt.Sprintf("%s.content[%d]", path, i)
		childObj, ok := asObject(rawChild)
		if !ok {
			v.add(childPath, ref, "invalid_type", "Node harus berupa objek JSON.")
			continue
		}
		childType, _ := asString(childObj["type"])
		if childType != "paragraph" {
			v.invalidNodeType(childPath, ref, childType, "Sel tabel hanya boleh berisi paragraf.")
			continue
		}
		res := v.inlineContent(childObj["content"], childPath+".content", ref, blogMarks)
		chars += res.chars
		if res.text != "" {
			textParts = append(textParts, res.text)
		}
		paragraphs = append(paragraphs, map[string]any{"type": "paragraph", "content": res.content})
	}
	if chars > maxTableCellRunes {
		v.add(path, ref, "too_long", fmt.Sprintf("Sel tabel maksimal %d karakter.", maxTableCellRunes))
	}
	v.addText(stringsJoinSpace(textParts))
	cell["content"] = paragraphs
	return cell
}

func (v *validator) headingLevel(attrs map[string]any, path, ref string, min, max int) (int, bool) {
	raw, exists := attrs["level"]
	if !exists || raw == nil {
		v.add(path+".attrs.level", ref, "required", "Level heading wajib diisi.")
		return 0, false
	}
	level, ok := intValue(raw)
	if !ok {
		v.add(path+".attrs.level", ref, "invalid_type", "Level heading harus berupa bilangan bulat.")
		return 0, false
	}
	if level < int64(min) || level > int64(max) {
		v.add(path+".attrs.level", ref, "invalid_value", fmt.Sprintf("Level heading harus %d–%d.", min, max))
		return 0, false
	}
	return int(level), true
}

func attrsWithID(id string) map[string]any {
	attrs := map[string]any{}
	if id != "" {
		attrs["id"] = id
	}
	return attrs
}

func nodeWithAttrs(node, attrs map[string]any) map[string]any {
	if len(attrs) > 0 {
		node["attrs"] = attrs
	}
	return node
}

func stringsJoinSpace(parts []string) string {
	out := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		if out != "" {
			out += " "
		}
		out += part
	}
	return out
}
