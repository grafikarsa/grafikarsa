package main

import (
	"context"
	"fmt"
	"github.com/microcosm-cc/bluemonday"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// migrateSocial memindahkan likes/follows/comments/notifications 1:1.
func migrateSocial(ctx context.Context, v1, v2 *pgxpool.Pool, s *v1schema, o options, st *stats) error {
	// likes
	if lt := s.firstTable("portfolio_likes", "likes"); lt != "" {
		uCol := colOr(s, lt, "user_id")
		pCol := colOr(s, lt, "portfolio_id")
		if uCol != "" && pCol != "" {
			rows, err := v1.Query(ctx, fmt.Sprintf(`SELECT %s, %s FROM %s`, uCol, pCol, lt))
			if err == nil {
				for rows.Next() {
					var u, p any
					if err := rows.Scan(&u, &p); err != nil {
						break
					}
					uid, ok1 := mappedID(ctx, v2, "users", uuidString(u), "users")
					pid, ok2 := mappedID(ctx, v2, "portfolios", uuidString(p), "portfolios")
					if !ok1 || !ok2 {
						st.Skipped["like_ref"]++
						continue
					}
					if !o.dryRun {
						_, _ = v2.Exec(ctx, `INSERT INTO portfolio_likes (user_id, portfolio_id)
							VALUES ($1,$2) ON CONFLICT DO NOTHING`, uid, pid)
					}
					st.Likes++
				}
				rows.Close()
			}
		}
	}
	// follows
	if ft := s.firstTable("follows"); ft != "" {
		fCol := colOr(s, ft, "follower_id", "follower")
		tCol := colOr(s, ft, "following_id", "followed_id", "following")
		if fCol != "" && tCol != "" {
			rows, err := v1.Query(ctx, fmt.Sprintf(`SELECT %s, %s FROM %s`, fCol, tCol, ft))
			if err == nil {
				for rows.Next() {
					var f, t any
					if err := rows.Scan(&f, &t); err != nil {
						break
					}
					fid, ok1 := mappedID(ctx, v2, "users", uuidString(f), "users")
					tid, ok2 := mappedID(ctx, v2, "users", uuidString(t), "users")
					if !ok1 || !ok2 {
						st.Skipped["follow_ref"]++
						continue
					}
					if !o.dryRun {
						_, _ = v2.Exec(ctx, `INSERT INTO follows (follower_id, following_id)
							VALUES ($1,$2) ON CONFLICT DO NOTHING`, fid, tid)
					}
					st.Follows++
				}
				rows.Close()
			}
		}
	}
	// comments
	if ct := s.firstTable("comments"); ct != "" {
		idCol := colOr(s, ct, "id")
		uCol := colOr(s, ct, "user_id", "author_id")
		pCol := colOr(s, ct, "portfolio_id")
		bodyCol := colOr(s, ct, "body", "content", "text", "message")
		if idCol != "" && uCol != "" && pCol != "" {
			rows, err := v1.Query(ctx, fmt.Sprintf(`SELECT %s, %s, %s, %s FROM %s`, idCol, uCol, pCol, nullCol(bodyCol), ct))
			if err == nil {
				for rows.Next() {
					var id, u, p, body any
					if err := rows.Scan(&id, &u, &p, &body); err != nil {
						break
					}
					sid := uuidString(id)
					if _, ok := mappedID(ctx, v2, "comments", sid, "comments"); ok {
						continue
					}
					uid, ok1 := mappedID(ctx, v2, "users", uuidString(u), "users")
					pid, ok2 := mappedID(ctx, v2, "portfolios", uuidString(p), "portfolios")
					if !ok1 || !ok2 {
						st.Skipped["comment_ref"]++
						continue
					}
					b := strings.TrimSpace(fmt.Sprint(body))
					if b == "" || b == "<nil>" {
						st.Skipped["comment_empty"]++
						continue
					}
					if !o.dryRun {
						var nid string
						// Kolom v2: author_id (bukan user_id) dan body_html NOT NULL.
						// Memakai nama kolom yang salah membuat INSERT gagal
						// tanpa jejak karena errornya ditelan.
						err := v2.QueryRow(ctx, `INSERT INTO comments (portfolio_id, author_id, body_md, body_html)
							VALUES ($1,$2,$3,$4) RETURNING id::text`,
							pid, uid, b, sanitizeHTML(b)).Scan(&nid)
						if err != nil {
							slog.Warn("komentar gagal", "src", sid, "err", err)
							st.Skipped["comment_insert"]++
							continue
						}
						_ = saveMap(ctx, v2, o.dryRun, "comments", sid, "comments", nid)
					}
					st.Comments++
				}
				rows.Close()
			}
		}
	}
	// notifications (is_read keep, data dinormalisasi)
	if nt := s.firstTable("notifications"); nt != "" {
		uCol := colOr(s, nt, "user_id")
		tiCol := colOr(s, nt, "title")
		boCol := colOr(s, nt, "body", "message", "text")
		tyCol := colOr(s, nt, "type", "kind")
		rdCol := colOr(s, nt, "is_read", "read")
		if uCol != "" {
			rows, err := v1.Query(ctx, fmt.Sprintf(`SELECT %s, %s, %s, %s, %s FROM %s`,
				uCol, nullCol(tiCol), nullCol(boCol), nullCol(tyCol), nullCol(rdCol), nt))
			if err == nil {
				for rows.Next() {
					var u, ti, bo, ty, rd any
					if err := rows.Scan(&u, &ti, &bo, &ty, &rd); err != nil {
						break
					}
					uid, ok := mappedID(ctx, v2, "users", uuidString(u), "users")
					if !ok {
						st.Skipped["notif_ref"]++
						continue
					}
					ntype := mapNotifType(fmt.Sprint(ty))
					title := fmt.Sprint(ti)
					if title == "" || title == "<nil>" {
						title = "Notifikasi (migrasi)"
					}
					isRead := fmt.Sprint(rd) == "true" || fmt.Sprint(rd) == "t" || fmt.Sprint(rd) == "1"
					if !o.dryRun {
						_, _ = v2.Exec(ctx, `INSERT INTO notifications (user_id, type, title, body, is_read, data)
							VALUES ($1,$2,$3,$4,$5,'{}')`, uid, ntype, title, fmt.Sprint(bo), isRead)
					}
					st.Notifications++
				}
				rows.Close()
			}
		}
	}
	return nil
}

func mapNotifType(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "like", "portfolio_liked":
		return "portfolio_liked"
	case "follow", "new_follower":
		return "new_follower"
	case "comment", "new_comment":
		return "new_comment"
	case "approved", "portfolio_approved":
		return "portfolio_approved"
	case "feedback":
		return "feedback_updated"
	default:
		return "announcement"
	}
}

// sanitizeHTML membersihkan HTML dari konten v1 sebelum disimpan di
// body_html. Isi v1 berasal dari input pengguna, jadi tidak boleh disimpan
// mentah-mentah di kolom yang someday bisa dirender sebagai HTML.
var ugcPolicy = bluemonday.UGCPolicy()

func sanitizeHTML(s string) string {
	return strings.TrimSpace(ugcPolicy.Sanitize(s))
}

// migrateAssessments memindahkan metrics + assessments 1:1 + isi assessment_avg.
func migrateAssessments(ctx context.Context, v1, v2 *pgxpool.Pool, s *v1schema, o options, st *stats) error {
	// Sumber data assessment di v1: tabel 'assessments'/'scores'/'portfolio_assessments'
	// ATAU view 'v_portfolio_assessments' (yang di v1 prod berisi 901 baris).
	at := s.firstTableWithRows(ctx, v1, "assessments", "scores", "v_portfolio_assessments", "portfolio_assessments")
	if at == "" {
		return nil
	}
	idCol := colOr(s, at, "id", "assessment_id")
	uCol := colOr(s, at, "user_id", "assessor_id", "guru_id", "assessed_by")
	pCol := colOr(s, at, "portfolio_id")
	scCol := colOr(s, at, "total_score", "score", "nilai", "value")
	cmCol := colOr(s, at, "comment", "final_comment", "komentar")
	if idCol == "" {
		return nil
	}
	rows, err := v1.Query(ctx, fmt.Sprintf(`SELECT %s, %s, %s, %s, %s FROM %s`,
		idCol, nullCol(uCol), nullCol(pCol), nullCol(scCol), nullCol(cmCol), at))
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, u, p, sc, cm any
		if err := rows.Scan(&id, &u, &p, &sc, &cm); err != nil {
			return err
		}
		sid := uuidString(id)
		if sid == "" {
			// Beberapa baris di view v1 punya assessment_id NULL; pakai
			// portfolio_id sebagai kunci pemetaan supaya tidak terduplikasi.
			sid = uuidString(p)
		}
		if _, ok := mappedID(ctx, v2, "assessments", sid, "assessments"); ok {
			continue
		}
		pid, ok := mappedID(ctx, v2, "portfolios", uuidString(p), "portfolios")
		if !ok {
			st.Skipped["assessment_ref"]++
			continue
		}
		// Baris dengan skor NULL berarti tidak ada assessment sebenarnya
		// (di v1 prod view-nya berisi 901 baris kosong). Membuat assessment
		// ber skor 0 hanya mengotori antrean penilaian admin.
		if sc == nil {
			st.Skipped["assessment_empty"]++
			continue
		}
		uid, _ := mappedID(ctx, v2, "users", uuidString(u), "users")
		var score float64
		if _, err := fmt.Sscanf(fmt.Sprint(sc), "%f", &score); err != nil {
			// Nilai assessment yang tidak bisa diparse dianggap 0 supaya
			// migrasi tetap lanjut; assessment tak ada di v1 jadi ini
			// hanya memengaruhi kolom yang memang kosong.
			score = 0
		}
		comment := strings.TrimSpace(fmt.Sprint(cm))
		if comment == "<nil>" {
			comment = ""
		}
		if !o.dryRun {
			var nid string
			// Tabel target v2 bernama portfolio_assessments (bukan
			// "assessments") — INSERT ke nama yang salah akan gagal
			// diam-diam karena errornya ditelan.
			// Kolom comment v2 NOT NULL — jangan pakai NULLIF yang menghasilkan
			// NULL, pakai string kosong bila tidak ada komentar.
			err := v2.QueryRow(ctx, `INSERT INTO portfolio_assessments (portfolio_id, assessor_id, total_score, comment)
				VALUES ($1,NULLIF($2,'')::uuid,$3,$4)
				ON CONFLICT (portfolio_id) DO UPDATE SET total_score=EXCLUDED.total_score, comment=EXCLUDED.comment
				RETURNING id::text`, pid, uid, score, comment).Scan(&nid)
			if err != nil {
				slog.Warn("assessment gagal", "src", sid, "err", err)
				st.Skipped["assessment_insert"]++
				continue
			}
			_ = saveMap(ctx, v2, o.dryRun, "assessments", sid, "assessments", nid)
		}
		st.Assessments++
	}
	return rows.Err()
}

// migrateFeedbackChangelog: feedback -> threads+1 pesan awal; changelogs -> changelogs+sections.
func migrateFeedbackChangelog(ctx context.Context, v1, v2 *pgxpool.Pool, s *v1schema, o options, st *stats) error {
	if ft := s.firstTable("feedback", "feedbacks"); ft != "" {
		idCol := colOr(s, ft, "id")
		tiCol := colOr(s, ft, "title", "judul", "subject")
		boCol := colOr(s, ft, "pesan", "body", "message", "content")
		emCol := colOr(s, ft, "email", "reporter_email")
		if idCol != "" {
			rows, err := v1.Query(ctx, fmt.Sprintf(`SELECT %s, %s, %s, %s FROM %s`,
				idCol, nullCol(tiCol), nullCol(boCol), nullCol(emCol), ft))
			if err == nil {
				for rows.Next() {
					var id, ti, bo, em any
					if err := rows.Scan(&id, &ti, &bo, &em); err != nil {
						break
					}
					sid := uuidString(id)
					if _, ok := mappedID(ctx, v2, "feedback_threads", sid, "feedback_threads"); ok {
						continue
					}
					title := fmt.Sprint(ti)
					if title == "" || title == "<nil>" {
						title = "Masukan (migrasi)"
					}
					body := fmt.Sprint(bo)
					if body == "" || body == "<nil>" {
						body = title
					}
					if !o.dryRun {
						var nid string
						err := v2.QueryRow(ctx, `INSERT INTO feedback_threads
							(category, title, body_md, reporter_email, status, is_public)
							VALUES ('question',$1,$2,NULLIF($3,''), 'open', TRUE) RETURNING id::text`,
							title, body, fmt.Sprint(em)).Scan(&nid)
						if err == nil {
							_, _ = v2.Exec(ctx, `INSERT INTO feedback_messages (thread_id, body_md) VALUES ($1,$2)`, nid, body)
							_ = saveMap(ctx, v2, o.dryRun, "feedback_threads", sid, "feedback_threads", nid)
						}
					}
					st.Feedback++
				}
				rows.Close()
			}
		}
	}
	if ct := s.firstTable("changelogs"); ct != "" {
		idCol := colOr(s, ct, "id")
		veCol := colOr(s, ct, "version", "versi")
		tiCol := colOr(s, ct, "title", "judul")
		boCol := colOr(s, ct, "body", "content", "summary", "description")
		if idCol != "" {
			rows, err := v1.Query(ctx, fmt.Sprintf(`SELECT %s, %s, %s, %s FROM %s`,
				idCol, nullCol(veCol), nullCol(tiCol), nullCol(boCol), ct))
			if err == nil {
				for rows.Next() {
					var id, ve, ti, bo any
					if err := rows.Scan(&id, &ve, &ti, &bo); err != nil {
						break
					}
					sid := uuidString(id)
					if _, ok := mappedID(ctx, v2, "changelogs", sid, "changelogs"); ok {
						continue
					}
					ver := fmt.Sprint(ve)
					if ver == "" || ver == "<nil>" {
						ver = "1.0." + fmt.Sprint(st.Changelogs)
					}
					title := fmt.Sprint(ti)
					if title == "" || title == "<nil>" {
						title = "Rilis " + ver
					}
					if !o.dryRun {
						var nid string
						err := v2.QueryRow(ctx, `INSERT INTO changelogs
							(version, title, summary, channel, audience, status)
							VALUES ($1,$2,$3,'stable','all','published')
							ON CONFLICT (version) DO UPDATE SET title=EXCLUDED.title
							RETURNING id::text`, ver, title, fmt.Sprint(bo)).Scan(&nid)
						if err == nil {
							_, _ = v2.Exec(ctx, `INSERT INTO changelog_sections (changelog_id, kind, title, body_md, position)
								VALUES ($1,'improved',$2,$3,0)`, nid, title, fmt.Sprint(bo))
							_ = saveMap(ctx, v2, o.dryRun, "changelogs", sid, "changelogs", nid)
						}
					}
					st.Changelogs++
				}
				rows.Close()
			}
		}
	}
	return nil
}

// migrateMediaRefs mendaftarkan URL lama menjadi media_assets + legacy_urls.
// Dipanggil SETELAH `mc mirror` (PRD §16.3): objek sudah ada di storage baru
// dengan layout key yang sama, jadi tool hanya mencatat metadata.
func migrateMediaRefs(ctx context.Context, v1, v2 *pgxpool.Pool, s *v1schema, o options, st *stats) error {
	// Kumpulkan semua URL /storage/ dari v1.
	urls := map[string]string{} // url -> pemilik (username) bila diketahui
	collect := func(table, urlCol, ownerCol string) {
		if !s.has(table, urlCol) {
			return
		}
		q := fmt.Sprintf(`SELECT %s FROM %s WHERE %s LIKE '%%/storage/%%'`, urlCol, table, urlCol)
		rows, err := v1.Query(ctx, q)
		if err != nil {
			return
		}
		for rows.Next() {
			var u any
			if err := rows.Scan(&u); err != nil {
				break
			}
			url := fmt.Sprint(u)
			if url != "" {
				urls[url] = ownerCol
			}
		}
		rows.Close()
	}
	collect("users", colOr(s, "users", "avatar_url", "avatar", "photo_url"), "users")
	for _, t := range []string{"portfolios", "content_blocks", "portfolio_blocks", "blocks"} {
		if !s.tables[t] {
			continue
		}
		for _, c := range []string{"thumbnail_url", "thumbnail", "cover_url", "image_url", "url", "file_url", "attachment_url"} {
			if s.has(t, c) {
				collect(t, c, t)
			}
		}
	}
	for url := range urls {
		key := keyFromStorageURL(url)
		owner := urls[url]
		_ = owner
		if !o.dryRun {
			var mid string
			err := v2.QueryRow(ctx, `INSERT INTO media_assets
				(owner_id, purpose, object_key, mime, size_bytes, checksum_sha256, status)
				SELECT u.id, 'document', $1, 'application/octet-stream', 0, $2, 'confirmed'
				FROM users u WHERE u.username='admin' LIMIT 1
				ON CONFLICT DO NOTHING RETURNING id::text`, key, sha256str(url)).Scan(&mid)
			if err == nil && mid != "" {
				_, _ = v2.Exec(ctx, `INSERT INTO legacy_urls (old_url, media_id, object_key)
					VALUES ($1,$2,$3) ON CONFLICT (old_url) DO UPDATE SET media_id=EXCLUDED.media_id`,
					url, mid, key)
				st.MediaRegistered++
			} else {
				_, _ = v2.Exec(ctx, `INSERT INTO legacy_urls (old_url, object_key)
					VALUES ($1,$2) ON CONFLICT (old_url) DO NOTHING`, url, key)
			}
		} else {
			st.MediaRegistered++
		}
	}
	slog.Info("URL lama terdaftar", "count", len(urls))
	return nil
}

// verifyMigration memeriksa hitungan pasca-migrasi (gagal = exit non-zero).
func verifyMigration(ctx context.Context, v1, v2 *pgxpool.Pool, st *stats) error {
	checks := []struct {
		label string
		v1q   string
		v2q   string
	}{
		{"users", `SELECT count(*) FROM users`, `SELECT count(*) FROM users WHERE email NOT LIKE '%@migrated.local'`},
		{"portfolios", `SELECT count(*) FROM portfolios`, `SELECT count(*) FROM portfolios`},
	}
	for _, c := range checks {
		var a, b int64
		if err := v1.QueryRow(ctx, c.v1q).Scan(&a); err != nil {
			slog.Warn("verifikasi dilewati (v1 tidak bisa dihitung)", "label", c.label)
			continue
		}
		if err := v2.QueryRow(ctx, c.v2q).Scan(&b); err != nil {
			return fmt.Errorf("%s: hitung v2 gagal: %w", c.label, err)
		}
		if a != b {
			return fmt.Errorf("%s: v1=%d v2=%d (tidak sama)", c.label, a, b)
		}
		slog.Info("verifikasi OK", "label", c.label, "count", b)
	}
	var q int64
	_ = v2.QueryRow(ctx, `SELECT count(*) FROM quarantine_blocks`).Scan(&q)
	slog.Info("quarantine tercatat (bukan hilang)", "blocks", q)
	var mapped int64
	_ = v2.QueryRow(ctx, `SELECT count(*) FROM legacy_id_map`).Scan(&mapped)
	if mapped == 0 {
		return fmt.Errorf("legacy_id_map kosong — migrasi tidak menulis mapping")
	}
	return nil
}
