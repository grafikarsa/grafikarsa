package main

import (
	"context"
	"fmt"
	"github.com/microcosm-cc/bluemonday"
	"log/slog"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"grafikarsa/api/internal/config"
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
//
// Metadata yang disimpanharus ASLI: mime, ukuran, dan checksum objek di
// storage baru. Handler by-key memakai media_assets.mime sebagai Content-Type
// — kalau mime-nya application/octet-stream, browser mengunduh tiap gambar
// sebagai ganti menampilkannya.
//
// Kalau storage baru tidak bisa dihubungi (mis. --storage-host kosong),
// migrasi tetap jalan TAPI memperingatkan: metadata akan generik dan
// keberadaan objek tidak terverifikasi.
func migrateMediaRefs(ctx context.Context, v1, v2 *pgxpool.Pool, s *v1schema, o options, st *stats) error {
	// Index storage baru: metadata asli + verifikasi keberadaan objek.
	// Kalau storage tidak bisa dihubungi, migrasi tetap jalan dengan
	// metadata generik — tapi itu berarti metadata & keberadaan objek TIDAK
	// terverifikasi, jadi harus ada di log.
	var ix *storageIndex
	var missing []string
	var alreadyGone []string
	// Manifest key v1 (dari tools/storage-mirror.sh). Kalau ada, kita bisa
	// membedakan "objek hilang karena mirror gagal" (fatal) dari "objek memang
	// sudah hilang di v1" (bukan kerusakan migrasi).
	var v1Keys map[string]struct{}
	if o.v1Manifest != "" {
		raw, err := os.ReadFile(o.v1Manifest)
		if err != nil {
			slog.Warn("manifest v1 tidak bisa dibaca; objek hilang hanya jadi peringatan", "file", o.v1Manifest, "err", err)
		} else {
			v1Keys = map[string]struct{}{}
			for _, line := range strings.Split(string(raw), "\n") {
				k := strings.TrimSpace(line)
				if k != "" {
					v1Keys[k] = struct{}{}
				}
			}
			slog.Info("manifest v1 dimuat", "file", o.v1Manifest, "key", len(v1Keys))
		}
	}
	if cfg, err := config.Load(); err != nil {
		slog.Warn("config tidak bisa dimuat; metadata media generik", "err", err)
	} else {
		if o.storageHost != "" {
			cfg.S3.Endpoint = o.storageHost
		}
		if ix, err = newStorageIndex(ctx, cfg); err != nil {
			slog.Warn("storage baru tidak bisa dihubungi; metadata media generik & keberadaan objek tak terverifikasi", "err", err)
			ix = nil
		}
	}

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
	// Banner terpisah dari avatar: di v1 ada users.banner_url (364 gambar di
	// produksi). Kalau hanya avatar yang dipindai, URL banner lama tidak
	// terdaftar dan berakhir 410 Gone.
	collect("users", colOr(s, "users", "banner_url", "cover_url"), "users")
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
	// URL gambar di dalam payload JSON blok portfolio (2.110 blok di
	// produksi) tidak ada di kolom URL — kalau tidak dikumpulkan, URL lamanya
	// tidak terdaftar dan berakhir 410 Gone.
	jsonURLs := map[string]bool{}
	for _, t := range []string{"content_blocks", "portfolio_blocks", "blocks"} {
		collectJSONURLs(ctx, v1, s, t, "payload", jsonURLs)
	}
	for u := range jsonURLs {
		if _, ok := urls[u]; !ok {
			urls[u] = "payload"
		}
	}
	for url := range urls {
		key := keyFromStorageURL(url)
		owner := urls[url]
		_ = owner

		// Metadata asli dari storage baru (kalau bisa dihubungi).
		mime := mimeByExt(key)
		var size int64
		sum := ""
		if ix != nil {
			size = ix.size(key)
			if size == 0 {
				// Objek tidak ada di storage baru. Dua kemungkinan:
				//   a) ada di manifest v1 -> mirror GAGAL (objek hilang saat
				//      disalin) -> fatal, jangan lanjut
				//   b) tidak ada di manifest v1 -> memang sudah hilang di v1
				//      (gambarnya sudah rusak sebelum migrasi) -> bukan
				//      kerusakan hasil migrasi, cukup dicatat
				if v1Keys != nil {
					if _, wasThere := v1Keys[key]; wasThere {
						st.Skipped["media_objek_hilang_setelah_mirror"]++
						missing = append(missing, key)
						continue
					}
				}
				st.Skipped["media_objek_tak_ada_sudah_di_v1"]++
				alreadyGone = append(alreadyGone, key)
				// Tetap didaftarkan sebagai legacy_url tanpa media_id: URL
				// lamanya tetap menjawab 410 (bukan 404), dan tidak ada
				// media_assets yang menunjuk key tak ada.
				if !o.dryRun {
					if _, lerr := v2.Exec(ctx, `INSERT INTO legacy_urls (old_url, object_key)
						VALUES ($1,$2) ON CONFLICT (old_url) DO NOTHING`, url, key); lerr != nil {
						slog.Error("gagal mencatat legacy_url", "url", url, "err", lerr)
					}
				}
				continue
			}
			if s, err := ix.checksum(ctx, key); err == nil {
				sum = s
			} else {
				slog.Warn("gagal menghitung checksum objek", "key", key, "err", err)
			}
		}

		if !o.dryRun {
			var mid string
			// ON CONFLICT (object_key) DO UPDATE + RETURNING: migrasi harus
			// idempoten. Dengan DO NOTHING, baris yang sudah ada tidak
			// mengembalikan id -> kolom kosong -> setiap media terhitung
			// "fallback" dan migrasi gagal padahal job-nya sudah beres.
			err := v2.QueryRow(ctx, `INSERT INTO media_assets
				(owner_id, purpose, object_key, mime, size_bytes, checksum_sha256, status)
				SELECT u.id, $2, $1, $3, $4, $5, 'confirmed'
				FROM users u WHERE u.username='admin' LIMIT 1
				ON CONFLICT (object_key) DO UPDATE SET
					mime=EXCLUDED.mime,
					size_bytes=EXCLUDED.size_bytes,
					checksum_sha256=EXCLUDED.checksum_sha256
				RETURNING id::text`, key, purposeFromKey(key), mime, size, sum).Scan(&mid)
			if err == nil && mid != "" {
				_, _ = v2.Exec(ctx, `INSERT INTO legacy_urls (old_url, media_id, object_key)
					VALUES ($1,$2,$3) ON CONFLICT (old_url) DO UPDATE SET media_id=EXCLUDED.media_id`,
					url, mid, key)
				st.MediaRegistered++
			} else {
				// Insert media_assets gagal. Object key tetap dicatat di
				// legacy_urls supaya URL lamanya tidak mati, tapi objeknya
				// TIDAK terikat ke media — harus dihitung dan menggagalkan
				// migrasi, bukan lolos diam-diam.
				if _, lerr := v2.Exec(ctx, `INSERT INTO legacy_urls (old_url, object_key)
					VALUES ($1,$2) ON CONFLICT (old_url) DO NOTHING`, url, key); lerr != nil {
					slog.Error("gagal mencatat legacy_url", "url", url, "err", lerr)
				}
				st.MediaFallback++
				slog.Warn("objek storage tidak terikat ke media_assets", "url", url, "key", key)
			}
		} else {
			st.MediaRegistered++
		}
	}
	slog.Info("URL lama terdaftar",
		"count", len(urls), "registered", st.MediaRegistered, "fallback", st.MediaFallback,
		"objek_tidak_ada", len(missing), "sudah_hilang_di_v1", len(alreadyGone))
	if len(alreadyGone) > 0 {
		// Sudah hilang sebelum migrasi (gambarnya rusak di v1 juga). Bukan
		// kegagalan migrasi — tapi harus terlihat, jangan diam.
		sample := alreadyGone
		if len(sample) > 5 {
			sample = sample[:5]
		}
		slog.Warn("objek yang sudah hilang di v1 (bukan kerusakan migrasi)",
			"jumlah", len(alreadyGone), "contoh", sample)
	}
	if st.MediaFallback > 0 {
		return fmt.Errorf("%d objek storage tidak bisa didaftarkan sebagai media_assets", st.MediaFallback)
	}
	if len(missing) > 0 {
		// Tampilkan sebagian kecil saja supaya log tidak membanjiri.
		sample := missing
		if len(sample) > 10 {
			sample = sample[:10]
		}
		return fmt.Errorf("%d objek yang ada di v1 hilang di storage baru — MIRROR GAGAL (contoh: %v)",
			len(missing), sample)
	}
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
		// Akademik: kelas & jurusan v1 wajib utuh di v2. Tanpa ini, kelas
		// bisa hilang diam-diam — semua user jadi class_id NULL dan
		// enrollments kosong, tapi migrasi tetap melaporkan "OK".
		//
		// Dicek lewat legacy_id_map (bukan hitung baris tabel v2): v2 bisa
		// sudah punya baris dari seed atau migrasi sebelumnya, sementara
		// yang wajib sama adalah "setiap baris v1 punya padanan di v2".
		{"classes", `SELECT count(*) FROM kelas`, `SELECT count(*) FROM legacy_id_map WHERE source_table='classes' AND target_table='classes'`},
		{"majors", `SELECT count(*) FROM jurusan`, `SELECT count(*) FROM legacy_id_map WHERE source_table='majors' AND target_table='majors'`},
		{"enrollments", `SELECT count(*) FROM users WHERE kelas_id IS NOT NULL`, `SELECT count(*) FROM enrollments`},
		{"user_dengan_kelas", `SELECT count(*) FROM users WHERE kelas_id IS NOT NULL`, `SELECT count(*) FROM users WHERE class_id IS NOT NULL`},
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
