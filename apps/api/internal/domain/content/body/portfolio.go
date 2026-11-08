package body

import (
	"fmt"
	"regexp"
)

var (
	portfolioBlockTypes = map[string]struct{}{
		"image": {}, "gallery": {}, "video": {}, "embed": {}, "file": {},
		"model3d": {}, "text": {}, "code": {}, "table": {}, "link": {}, "divider": {},
	}
	portfolioMediaBlocks = map[string]struct{}{
		"image": {}, "gallery": {}, "video": {}, "embed": {}, "file": {}, "model3d": {},
	}
	fitValues = map[string]struct{}{
		"contain": {}, "wide": {},
	}
	galleryLayouts = map[string]struct{}{
		"grid": {}, "carousel": {},
	}
	aspectValues = map[string]struct{}{
		"16:9": {}, "4:3": {}, "1:1": {}, "9:16": {},
	}
	linkStyles = map[string]struct{}{
		"button": {}, "card": {},
	}
	videoProviders = map[string]struct{}{
		"youtube": {}, "vimeo": {},
	}
	portfolioEmbedProviders = map[string]struct{}{
		"figma": {}, "canva": {}, "google_slides": {}, "google_docs": {},
		"google_drive": {}, "sketchfab": {}, "codepen": {}, "codesandbox": {},
		"spotify": {},
	}
	blogEmbedProviders = map[string]struct{}{
		"figma": {}, "canva": {}, "google_slides": {}, "google_docs": {},
		"google_drive": {}, "sketchfab": {}, "codepen": {}, "codesandbox": {},
		"spotify": {}, "youtube": {}, "vimeo": {},
	}
	codeLanguages = map[string]struct{}{
		"plaintext": {}, "bash": {}, "c": {}, "cpp": {}, "csharp": {},
		"css": {}, "dart": {}, "dockerfile": {}, "go": {}, "html": {},
		"java": {}, "javascript": {}, "json": {}, "kotlin": {}, "markdown": {},
		"php": {}, "python": {}, "ruby": {}, "rust": {}, "scss": {},
		"sql": {}, "swift": {}, "typescript": {}, "tsx": {}, "jsx": {},
		"yaml": {}, "xml": {}, "arduino": {}, "lua": {}, "r": {},
	}

	youtubeVideoIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)
	vimeoVideoIDPattern   = regexp.MustCompile(`^[0-9]+$`)
)

func (v *validator) portfolio(root any) map[string]any {
	obj, ok := asObject(root)
	if !ok {
		v.add("body", "", "invalid_type", "Body portfolio harus berupa objek JSON.")
		return nil
	}

	out := map[string]any{}
	rawBlocks, ok := v.arrayField(obj, "blocks", "body.blocks", "", true)
	if !ok {
		out["blocks"] = []any{}
		return out
	}

	if len(rawBlocks) > maxPortfolioBlocks {
		v.add("body.blocks", "", "too_many", fmt.Sprintf("Portfolio maksimal %d blok.", maxPortfolioBlocks))
		rawBlocks = rawBlocks[:maxPortfolioBlocks]
	}

	seen := make(map[string]bool, len(rawBlocks))
	blocks := make([]any, 0, len(rawBlocks))
	for i, rawBlock := range rawBlocks {
		block := v.portfolioBlock(rawBlock, i, seen)
		if block != nil {
			blocks = append(blocks, block)
		}
	}
	out["blocks"] = blocks

	if v.submit {
		if !v.hasMediaBlock {
			v.add("body.blocks", "", "media_required", "Portfolio minimal memiliki satu blok media.")
		}
		if len(v.media) > maxPortfolioMedia {
			v.add("body.blocks", "", "too_many", fmt.Sprintf("Total media per portfolio maksimal %d file.", maxPortfolioMedia))
		}
	}

	return out
}

func (v *validator) portfolioBlock(raw any, index int, seen map[string]bool) map[string]any {
	path := fmt.Sprintf("body.blocks[%d]", index)
	obj, ok := asObject(raw)
	if !ok {
		v.add(path, "", "invalid_type", "Blok harus berupa objek JSON.")
		return nil
	}

	id, _ := v.blockIDField(obj, path+".id", "", true, seen)

	typeName, typeOK := asString(obj["type"])
	if !typeOK {
		v.add(path+".type", id, "unknown_type", "Jenis blok tidak dikenal.")
		return nil
	}
	if _, known := portfolioBlockTypes[typeName]; !known {
		v.add(path+".type", id, "unknown_type", "Jenis blok tidak dikenal.")
		return nil
	}

	dataPath := path + ".data"
	dataObj, _ := v.objectField(obj, "data", dataPath, id, false)

	if _, isMedia := portfolioMediaBlocks[typeName]; isMedia {
		v.hasMediaBlock = true
	}

	var data map[string]any
	switch typeName {
	case "image":
		data = v.imageData(dataObj, dataPath, id)
	case "gallery":
		data = v.galleryData(dataObj, dataPath, id)
	case "video":
		data = v.videoData(dataObj, dataPath, id)
	case "embed":
		data = v.embedData(dataObj, dataPath, id, portfolioEmbedProviders)
	case "file":
		data = v.fileData(dataObj, dataPath, id)
	case "model3d":
		data = v.model3dData(dataObj, dataPath, id)
	case "text":
		data = v.textBlockData(dataObj, dataPath, id)
	case "code":
		data = v.codeData(dataObj, dataPath, id)
	case "table":
		data = v.tableData(dataObj, dataPath, id)
	case "link":
		data = v.linkData(dataObj, dataPath, id)
	case "divider":
		data = map[string]any{}
	}

	return map[string]any{
		"id":   id,
		"type": typeName,
		"data": data,
	}
}

func (v *validator) imageData(obj map[string]any, path, ref string) map[string]any {
	out := map[string]any{
		"media_id": nil,
		"fit":      "contain",
	}
	if id, ok := v.mediaIDField(obj, "media_id", path+".media_id", ref, true); ok {
		out["media_id"] = id
	}
	if alt, ok := v.stringField(obj, "alt", path+".alt", ref, maxAltRunes, false, false); ok && alt != "" {
		out["alt"] = alt
	}
	if caption, ok := v.stringField(obj, "caption", path+".caption", ref, maxCaptionRunes, false, false); ok && caption != "" {
		out["caption"] = caption
		v.addText(caption)
	}
	if fit, ok := v.enumField(obj, "fit", path+".fit", ref, fitValues, "contain", false); ok {
		out["fit"] = fit
	}
	return out
}

func (v *validator) galleryData(obj map[string]any, path, ref string) map[string]any {
	out := map[string]any{
		"items":  []any{},
		"layout": "grid",
	}
	items := make([]any, 0)
	if rawItems, ok := v.arrayField(obj, "items", path+".items", ref, true); ok {
		if len(rawItems) > maxGalleryItems {
			v.add(path+".items", ref, "too_many", fmt.Sprintf("Galeri maksimal %d gambar.", maxGalleryItems))
			rawItems = rawItems[:maxGalleryItems]
		}
		for i, rawItem := range rawItems {
			itemPath := fmt.Sprintf("%s.items[%d]", path, i)
			itemObj, ok := asObject(rawItem)
			if !ok {
				v.add(itemPath, ref, "invalid_type", "Item galeri harus berupa objek JSON.")
				continue
			}
			item := map[string]any{"media_id": nil}
			if id, ok := v.mediaIDField(itemObj, "media_id", itemPath+".media_id", ref, true); ok {
				item["media_id"] = id
			}
			if alt, ok := v.stringField(itemObj, "alt", itemPath+".alt", ref, maxAltRunes, false, false); ok && alt != "" {
				item["alt"] = alt
			}
			if caption, ok := v.stringField(itemObj, "caption", itemPath+".caption", ref, maxCaptionRunes, false, false); ok && caption != "" {
				item["caption"] = caption
				v.addText(caption)
			}
			items = append(items, item)
		}
	}
	out["items"] = items

	if v.submit && len(items) < minGalleryItems {
		v.add(path+".items", ref, "too_few", fmt.Sprintf("Galeri minimal %d gambar.", minGalleryItems))
	}
	if layout, ok := v.enumField(obj, "layout", path+".layout", ref, galleryLayouts, "grid", false); ok {
		out["layout"] = layout
	}
	return out
}

func (v *validator) videoData(obj map[string]any, path, ref string) map[string]any {
	out := map[string]any{}
	provider, providerOK := v.enumField(obj, "provider", path+".provider", ref, videoProviders, "", true)
	if providerOK {
		out["provider"] = provider
	}

	videoID, idOK := v.stringField(obj, "video_id", path+".video_id", ref, 0, true, true)
	if idOK && providerOK {
		switch provider {
		case "youtube":
			if !youtubeVideoIDPattern.MatchString(videoID) {
				v.add(path+".video_id", ref, "invalid_id", "ID video YouTube tidak valid.")
				idOK = false
			}
		case "vimeo":
			if !vimeoVideoIDPattern.MatchString(videoID) {
				v.add(path+".video_id", ref, "invalid_id", "ID video Vimeo tidak valid.")
				idOK = false
			}
		}
	}
	if idOK {
		out["video_id"] = videoID
	}

	if raw, exists := obj["start"]; exists && raw != nil {
		start, ok := intValue(raw)
		switch {
		case !ok:
			v.add(path+".start", ref, "invalid_type", "Waktu mulai harus berupa bilangan bulat.")
		case start < 0:
			v.add(path+".start", ref, "invalid_value", "Waktu mulai tidak boleh negatif.")
		default:
			out["start"] = start
		}
	}
	if title, ok := v.stringField(obj, "title", path+".title", ref, 0, true, false); ok && title != "" {
		out["title"] = title
	}
	return out
}

func (v *validator) embedData(obj map[string]any, path, ref string, allowed map[string]struct{}) map[string]any {
	out := map[string]any{"aspect": "16:9"}
	provider, providerOK := v.enumField(obj, "provider", path+".provider", ref, allowed, "", true)
	rawURL, urlOK := v.stringField(obj, "url", path+".url", ref, 0, true, true)

	detected, embedOK := Embed{}, false
	if urlOK {
		detected, embedOK = ParseEmbed(rawURL)
		if !embedOK {
			v.add(path+".url", ref, "invalid_url", "URL embed tidak dikenali atau tidak diizinkan.")
		}
	}
	if embedOK {
		if _, ok := allowed[detected.Provider]; !ok {
			v.add(path+".provider", ref, "not_allowed", "Provider embed ini tidak diizinkan di sini.")
		} else if providerOK && provider != detected.Provider {
			v.add(path+".provider", ref, "not_allowed", "Provider tidak cocok dengan URL embed.")
		} else {
			out["provider"] = detected.Provider
			out["url"] = detected.URL
		}
	}
	if aspect, ok := v.enumField(obj, "aspect", path+".aspect", ref, aspectValues, "16:9", false); ok {
		out["aspect"] = aspect
	}
	return out
}

func (v *validator) fileData(obj map[string]any, path, ref string) map[string]any {
	out := map[string]any{"media_id": nil}
	if id, ok := v.mediaIDField(obj, "media_id", path+".media_id", ref, true); ok {
		out["media_id"] = id
	}
	if title, ok := v.stringField(obj, "title", path+".title", ref, maxFileTitleRunes, true, false); ok && title != "" {
		out["title"] = title
	}
	return out
}

func (v *validator) model3dData(obj map[string]any, path, ref string) map[string]any {
	out := map[string]any{
		"media_id":    nil,
		"auto_rotate": false,
	}
	if id, ok := v.mediaIDField(obj, "media_id", path+".media_id", ref, true); ok {
		out["media_id"] = id
	}
	if poster, ok := v.mediaIDField(obj, "poster_media_id", path+".poster_media_id", ref, false); ok {
		out["poster_media_id"] = poster
	}
	out["auto_rotate"] = v.boolField(obj, "auto_rotate", path+".auto_rotate", ref, false)
	return out
}

func (v *validator) textBlockData(obj map[string]any, path, ref string) map[string]any {
	rawDoc, exists := obj["doc"]
	if !exists || rawDoc == nil {
		v.add(path+".doc", ref, "required", "Dokumen teks wajib diisi.")
		return map[string]any{"type": "doc", "content": []any{}}
	}
	docObj, ok := asObject(rawDoc)
	if !ok {
		v.add(path+".doc", ref, "invalid_type", "Dokumen teks harus berupa objek JSON.")
		return map[string]any{"type": "doc", "content": []any{}}
	}
	return v.textDoc(docObj, path+".doc", ref)
}

func (v *validator) codeData(obj map[string]any, path, ref string) map[string]any {
	out := map[string]any{}
	if language, ok := v.enumField(obj, "language", path+".language", ref, codeLanguages, "", true); ok {
		out["language"] = language
	}
	code, _ := v.stringField(obj, "code", path+".code", ref, maxCodeRunes, false, true)
	out["code"] = code
	if filename, ok := v.stringField(obj, "filename", path+".filename", ref, 0, true, false); ok && filename != "" {
		out["filename"] = filename
	}
	return out
}

func (v *validator) tableData(obj map[string]any, path, ref string) map[string]any {
	out := map[string]any{
		"headers": []any{},
		"rows":    []any{},
	}

	headers := make([]any, 0)
	if rawHeaders, ok := v.arrayField(obj, "headers", path+".headers", ref, true); ok {
		if len(rawHeaders) > maxTableColumns {
			v.add(path+".headers", ref, "too_many", fmt.Sprintf("Tabel maksimal %d kolom.", maxTableColumns))
			rawHeaders = rawHeaders[:maxTableColumns]
		}
		for i, rawCell := range rawHeaders {
			cellPath := fmt.Sprintf("%s.headers[%d]", path, i)
			if cell, ok := v.tableCell(rawCell, cellPath, ref); ok {
				headers = append(headers, cell)
			}
		}
	}
	out["headers"] = headers

	rows := make([]any, 0)
	if rawRows, ok := v.arrayField(obj, "rows", path+".rows", ref, true); ok {
		if len(rawRows) > maxTableRows {
			v.add(path+".rows", ref, "too_many", fmt.Sprintf("Tabel maksimal %d baris.", maxTableRows))
			rawRows = rawRows[:maxTableRows]
		}
		for r, rawRow := range rawRows {
			rowPath := fmt.Sprintf("%s.rows[%d]", path, r)
			rawCells, ok := asArray(rawRow)
			if !ok {
				v.add(rowPath, ref, "invalid_type", "Baris tabel harus berupa array.")
				continue
			}
			if len(rawCells) > maxTableColumns {
				v.add(rowPath, ref, "too_many", fmt.Sprintf("Tabel maksimal %d kolom.", maxTableColumns))
				rawCells = rawCells[:maxTableColumns]
			}
			row := make([]any, 0, len(rawCells))
			for c, rawCell := range rawCells {
				cellPath := fmt.Sprintf("%s[%d]", rowPath, c)
				if cell, ok := v.tableCell(rawCell, cellPath, ref); ok {
					row = append(row, cell)
				}
			}
			rows = append(rows, row)
		}
	}
	out["rows"] = rows
	return out
}

func (v *validator) tableCell(raw any, path, ref string) (string, bool) {
	cell, ok := asString(raw)
	if !ok {
		v.add(path, ref, "invalid_type", "Sel tabel harus berupa teks.")
		return "", false
	}
	cell = sanitizeText(cell)
	if len([]rune(cell)) > maxTableCellRunes {
		v.add(path, ref, "too_long", fmt.Sprintf("Sel tabel maksimal %d karakter.", maxTableCellRunes))
	}
	v.addText(cell)
	return cell, true
}

func (v *validator) linkData(obj map[string]any, path, ref string) map[string]any {
	out := map[string]any{}
	if label, ok := v.stringField(obj, "label", path+".label", ref, maxLinkLabelRunes, true, true); ok {
		out["label"] = label
		v.addText(label)
	}
	rawURL, urlOK := v.stringField(obj, "url", path+".url", ref, 0, true, true)
	if urlOK {
		if !isHTTPSURL(rawURL) {
			v.add(path+".url", ref, "invalid_url", "Tautan harus menggunakan https://.")
		} else {
			out["url"] = rawURL
		}
	}
	if style, ok := v.enumField(obj, "style", path+".style", ref, linkStyles, "", true); ok {
		out["style"] = style
	}
	return out
}
