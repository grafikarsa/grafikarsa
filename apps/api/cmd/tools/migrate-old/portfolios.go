package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// migratePortfolios memindahkan portfolios + blocks + media refs.
// Payload divalidasi per tipe; yang gagal masuk quarantine_blocks (PRD §16.2).
func migratePortfolios(ctx context.Context, v1, v2 *pgxpool.Pool, s *v1schema, o options, st *stats) error {
	pTable := s.firstTable("portfolios")
	if pTable == "" {
		st.Skipped["portfolios"] = -1
		return fmt.Errorf("tabel portfolios tidak ada di v1")
	}
	idCol := colOr(s, pTable, "id")
	ownerCol := colOr(s, pTable, "owner_id", "user_id", "author_id")
	titleCol := colOr(s, pTable, "title", "judul")
	slugCol := colOr(s, pTable, "slug")
	statusCol := colOr(s, pTable, "status")
	thumbCol := colOr(s, pTable, "thumbnail_url", "thumbnail", "cover_url", "cover_image")
	bTable := s.firstTable("content_blocks", "portfolio_blocks", "blocks")
	if idCol == "" {
		return fmt.Errorf("kolom id portfolios v1 tidak dikenali")
	}

	rows, err := v1.Query(ctx, fmt.Sprintf(`SELECT %s, %s, %s, %s, %s, %s FROM %s ORDER BY %s`,
		idCol, nullCol(ownerCol), nullCol(titleCol), nullCol(slugCol), nullCol(statusCol), nullCol(thumbCol), pTable, idCol))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var srcID, owner, title, slug, status, thumb any
		if err := rows.Scan(&srcID, &owner, &title, &slug, &status, &thumb); err != nil {
			return err
		}
		sid := fmt.Sprint(srcID)
		if _, ok := mappedID(ctx, v2, "portfolios", sid, "portfolios"); ok {
			continue
		}
		ownerV2, ok := mappedID(ctx, v2, "users", fmt.Sprint(owner), "users")
		if !ok && !o.dryRun {
			slog.Warn("portfolio dilewati: owner belum termigrasi", "src", sid)
			st.Skipped["portfolio_owner"]++
			continue
		}
		t := strings.TrimSpace(fmt.Sprint(title))
		if t == "" {
			t = "Tanpa judul (migrasi)"
		}
		if len(t) < 10 {
			t += " (migrasi)"
		}
		sl := strings.TrimSpace(fmt.Sprint(slug))
		if sl == "" || sl == "<nil>" {
			sl = "migrasi-" + sid
		}
		stt := mapV1Status(fmt.Sprint(status))

		var targetID string
		if !o.dryRun {
			err := v2.QueryRow(ctx, `INSERT INTO portfolios (owner_id, title, slug, status)
				VALUES ($1,$2,$3,$4)
				ON CONFLICT (slug) DO UPDATE SET title=EXCLUDED.title
				RETURNING id::text`, ownerV2, t, sl, stt).Scan(&targetID)
			if err != nil {
				slog.Warn("portfolio gagal", "src", sid, "err", err)
				st.Skipped["portfolio_insert"]++
				continue
			}
			if err := saveMap(ctx, v2, o.dryRun, "portfolios", sid, "portfolios", targetID); err != nil {
				return err
			}
			// Thumbnail URL lama -> legacy_urls (bukan overwrite buta).
			if th := strings.TrimSpace(fmt.Sprint(thumb)); th != "" && th != "<nil>" {
				registerLegacyURL(ctx, v2, th, o)
			}
			st.Portfolios++
		} else {
			st.Portfolios++
			targetID = "dry-run"
		}
		if bTable != "" {
			migrateBlocks(ctx, v1, v2, s, bTable, sid, targetID, o, st)
		}
		_ = targetID
	}
	return rows.Err()
}

func mapV1Status(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "pending_review", "pending":
		return "pending_review"
	case "rejected":
		return "rejected"
	case "archived":
		return "archived"
	case "published":
		return "published"
	default:
		return "draft"
	}
}

func migrateBlocks(ctx context.Context, v1, v2 *pgxpool.Pool, s *v1schema, bTable, srcPID, targetPID string, o options, st *stats) {
	idCol := colOr(s, bTable, "id")
	typeCol := colOr(s, bTable, "type", "block_type", "kind")
	payloadCol := colOr(s, bTable, "payload", "content", "data", "body")
	posCol := colOr(s, bTable, "position", "order", "sort_order")
	pidCol := colOr(s, bTable, "portfolio_id", "content_id")
	if idCol == "" || typeCol == "" {
		return
	}
	where := ""
	if pidCol != "" {
		where = fmt.Sprintf(` WHERE %s = '%s'`, pidCol, strings.ReplaceAll(srcPID, "'", ""))
	}
	order := idCol
	if posCol != "" {
		order = posCol
	}
	rows, err := v1.Query(ctx, fmt.Sprintf(`SELECT %s, %s, %s, %s FROM %s%s ORDER BY %s`,
		idCol, typeCol, nullCol(payloadCol), nullCol(posCol), bTable, where, order))
	if err != nil {
		return
	}
	defer rows.Close()
	pos := 0
	for rows.Next() {
		var bid, btype, payload, ppos any
		if err := rows.Scan(&bid, &btype, &payload, &ppos); err != nil {
			return
		}
		bt := normalizeBlockType(fmt.Sprint(btype))
		if bt == "" {
			quarantine(ctx, v2, o, targetPID, fmt.Sprint(bid), fmt.Sprint(btype), fmt.Sprint(payload), "tipe block tidak dikenal")
			st.BlocksQuarantined++
			continue
		}
		pl, ok := normalizePayload(bt, fmt.Sprint(payload))
		if !ok {
			quarantine(ctx, v2, o, targetPID, fmt.Sprint(bid), bt, fmt.Sprint(payload), "payload tidak valid untuk tipe "+bt)
			st.BlocksQuarantined++
			continue
		}
		if !o.dryRun {
			pb, _ := json.Marshal(pl)
			if _, err := v2.Exec(ctx, `INSERT INTO portfolio_blocks (portfolio_id, block_type, position, payload)
				VALUES ($1,$2,$3,$4)`, targetPID, bt, pos, pb); err != nil {
				quarantine(ctx, v2, o, targetPID, fmt.Sprint(bid), bt, fmt.Sprint(payload), "insert gagal: "+err.Error())
				st.BlocksQuarantined++
				continue
			}
		}
		pos++
		st.Blocks++
	}
}

func normalizeBlockType(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "text", "markdown", "paragraph", "heading":
		return "text"
	case "image", "img", "photo", "gallery":
		return "image"
	case "table":
		return "table"
	case "youtube", "video":
		return "youtube"
	case "button", "link", "cta":
		return "button"
	case "embed", "iframe":
		return "embed"
	case "figma":
		return "figma"
	case "canva":
		return "canva"
	case "ppt", "slides", "presentation":
		return "ppt"
	case "pdf", "doc", "document", "file":
		return "pdf"
	default:
		return ""
	}
}

// normalizePayload memaksa payload ke bentuk kanonis per tipe; false = quarantine.
func normalizePayload(bt, raw string) (map[string]any, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = "{}"
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		// Bukan JSON: anggap teks mentah untuk block text.
		if bt == "text" {
			return map[string]any{"markdown": raw}, true
		}
		return nil, false
	}
	if m == nil {
		m = map[string]any{}
	}
	switch bt {
	case "text":
		md, _ := m["markdown"].(string)
		if md == "" {
			if b, ok := m["body"].(string); ok {
				md = b
			} else if c, ok := m["content"].(string); ok {
				md = c
			}
		}
		if len(md) < 1 || len(md) > 20000 {
			return nil, false
		}
		return map[string]any{"markdown": md}, true
	case "image":
		mid, _ := m["media_id"].(string)
		url, _ := m["url"].(string)
		alt, _ := m["alt"].(string)
		if mid == "" && url == "" {
			return nil, false
		}
		if len(alt) < 5 {
			alt = "Gambar portfolio (migrasi)"
		}
		out := map[string]any{"alt": alt}
		if mid != "" {
			out["media_id"] = mid
		}
		if url != "" {
			out["url"] = url
		}
		if cap, ok := m["caption"].(string); ok && cap != "" {
			out["caption"] = cap
		}
		return out, true
	case "table":
		h, _ := m["headers"].([]any)
		rws, _ := m["rows"].([]any)
		if len(h) == 0 || len(h) > 10 || len(rws) > 50 {
			return nil, false
		}
		return map[string]any{"headers": h, "rows": rws}, true
	case "youtube":
		vid, _ := m["video_id"].(string)
		if len(vid) != 11 {
			return nil, false
		}
		return map[string]any{"video_id": vid}, true
	case "button":
		label, _ := m["label"].(string)
		url, _ := m["url"].(string)
		if len(label) < 1 || len(label) > 50 || !strings.HasPrefix(url, "https://") {
			return nil, false
		}
		return map[string]any{"label": label, "url": url}, true
	default: // embed-like: butuh url https
		url, _ := m["url"].(string)
		if !strings.HasPrefix(url, "https://") {
			return nil, false
		}
		return map[string]any{"url": url}, true
	}
}

func quarantine(ctx context.Context, v2 *pgxpool.Pool, o options, portfolioID, srcBlock, btype, raw, reason string) {
	if o.dryRun || portfolioID == "dry-run" {
		return
	}
	if _, err := uuid.Parse(portfolioID); err != nil {
		return
	}
	payload, _ := json.Marshal(map[string]string{"raw": raw})
	_, _ = v2.Exec(ctx, `INSERT INTO quarantine_blocks (portfolio_id, source_block_id, block_type, raw_payload, reason)
		VALUES ($1,$2,$3,$4,$5)`, portfolioID, srcBlock, btype, payload, reason)
}

func registerLegacyURL(ctx context.Context, v2 *pgxpool.Pool, oldURL string, o options) {
	if o.dryRun || !strings.Contains(oldURL, "/storage/") {
		return
	}
	_, _ = v2.Exec(ctx, `INSERT INTO legacy_urls (old_url, object_key) VALUES ($1,$2)
		ON CONFLICT (old_url) DO NOTHING`, oldURL, keyFromStorageURL(oldURL))
}

func keyFromStorageURL(u string) string {
	if i := strings.Index(u, "/storage/"); i >= 0 {
		return strings.TrimPrefix(u[i+len("/storage/"):], "grafikarsa/")
	}
	return u
}
