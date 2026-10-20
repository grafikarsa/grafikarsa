package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// migrateAcademics memindahkan struktur akademik v1 ke v2:
//
//	v1 jurusan -> v2 majors   (dicocokkan by kode, lalu by nama; yang belum ada dibuat)
//	v1 kelas   -> v2 classes  (tingkat + rombel + tahun_ajaran + jurusan)
//	users.kelas_id -> enrollments
//
// PENTING: langkah ini harus jalan SEBELUM migrateUsers.
//
// User v1 menunjuk kelas & jurusan lewat UUID (users.kelas_id,
// users.jurusan_id), sedangkan v2 menyimpan class_id/major_id sebagai UUID
// miliknya sendiri. Tanpa mapping UUID v1 -> UUID v2, setiap user berakhir
// dengan class_id dan major_id NULL: 49 baris kelas dan keanggotaan kelas
// 1687 user hilang tanpa satu pun error. Dan karena users dimigrasi lebih
// dulu, mapping-nya belum ada saat user ditulis.
//
// Enrollments TIDAK dibuat di sini: enrollment meng_acuan user yang harus
// sudah ada di v2 (foreign key), jadi dibuat setelah migrateUsers —
// lihat migrateEnrollments.
//
// Urutan lain yang penting: academic_years & majors v2 adalah data referensi
// yang diisi seed — jadi seed harus jalan sebelum migrate-old.
func migrateAcademics(ctx context.Context, v1, v2 *pgxpool.Pool, s *v1schema, o options, st *stats) (map[string]string, error) {
	if err := migrateMajors(ctx, v1, v2, s, o, st); err != nil {
		return nil, fmt.Errorf("migrate majors: %w", err)
	}
	classMajor, err := migrateClasses(ctx, v1, v2, s, o, st)
	if err != nil {
		return nil, fmt.Errorf("migrate classes: %w", err)
	}
	return classMajor, nil
}

// migrateMajors memetakan v1 `jurusan` ke v2 `majors`.
//
// Pencocokan: kode dulu (case-insensitive), lalu nama. Kalau v2 belum punya
// major dengan kode/nama itu, major-nya DIBUAT — lebih baik satu major
// tambahan daripada satu jurusan v1 yang hilang (v1 prod punya 8 jurusan;
// seed v2 hanya memuat 5 dengan kode yang sebagian berbeda, mis. v1 "ani"
// vs v2 "animasi").
func migrateMajors(ctx context.Context, v1, v2 *pgxpool.Pool, s *v1schema, o options, st *stats) error {
	tab := s.firstTable("jurusan", "majors", "prodi", "study_programs")
	if tab == "" {
		return nil
	}
	kodeCol := colOr(s, tab, "kode", "code")
	namaCol := colOr(s, tab, "nama", "name")
	if kodeCol == "" || namaCol == "" {
		return nil
	}

	// Peta v2: kode & nama (lowercase) -> id.
	byCode := map[string]string{}
	byName := map[string]string{}
	if rows, err := v2.Query(ctx, `SELECT id::text, lower(code), lower(name) FROM majors`); err == nil {
		for rows.Next() {
			var id, code, name string
			if err := rows.Scan(&id, &code, &name); err == nil {
				byCode[code] = id
				byName[name] = id
			}
		}
		rows.Close()
	}

	rows, err := v1.Query(ctx, fmt.Sprintf(`SELECT id, %s, %s FROM %s ORDER BY id`, kodeCol, namaCol, tab))
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var srcID, kode, nama any
		if err := rows.Scan(&srcID, &kode, &nama); err != nil {
			return err
		}
		src := uuidString(srcID)
		k := strings.ToLower(strings.TrimSpace(fmt.Sprint(kode)))
		n := strings.ToLower(strings.TrimSpace(fmt.Sprint(nama)))
		if n == "" || n == "<nil>" {
			n = k
		}
		if k == "" || k == "<nil>" {
			// v2 majors.code NOT NULL -> pakai slug dari nama.
			k = slugify(n)
		}
		if k == "" || n == "" {
			st.Skipped["major_tanpa_nama"]++
			continue
		}

		targetID := byCode[k]
		if targetID == "" {
			targetID = byName[n]
		}
		if targetID == "" {
			if o.dryRun {
				targetID = "dry-major-" + src
				st.MajorsCreated++
			} else {
				if err := v2.QueryRow(ctx,
					`INSERT INTO majors (code, name) VALUES ($1,$2)
					 ON CONFLICT (code) DO UPDATE SET name=EXCLUDED.name
					 RETURNING id::text`, k, n).Scan(&targetID); err != nil {
					return fmt.Errorf("buat major %s: %w", k, err)
				}
				byCode[k] = targetID
				byName[n] = targetID
				st.MajorsCreated++
			}
		}
		if err := saveMap(ctx, v2, o.dryRun, "majors", src, "majors", targetID); err != nil {
			return err
		}
		st.Majors++
	}
	slog.Info("majors dipetakan", "total", st.Majors, "baru", st.MajorsCreated)
	return rows.Err()
}

// migrateClasses memetakan v1 `kelas` ke v2 `classes`.
//
// v1 `kelas` (id, tahun_ajaran_id, jurusan_id, tingkat, rombel, nama) dan v2
// `classes` (academic_year_id, major_id, grade, class_group, name) basically
// sama bentuknya. Yang perlu dijaga: grade harus 10/11/12 dan class_group
// satu huruf kapital — keduanya punya CHECK constraint di v2, jadi baris yang
// tidak memenuhi dilewati dan DILAPORKAN, bukan dipaksa masuk.
//
// Return: peta id kelas v1 -> id major v2. User v1 tidak punya kolom
// jurusan_id, jadi major miliknya diturunkan dari kelasnya.
func migrateClasses(ctx context.Context, v1, v2 *pgxpool.Pool, s *v1schema, o options, st *stats) (map[string]string, error) {
	classMajor := map[string]string{}
	tab := s.firstTable("kelas", "classes", "class_rooms")
	if tab == "" {
		return classMajor, nil
	}
	tahunCol := colOr(s, tab, "tahun_ajaran_id", "academic_year_id")
	jurCol := colOr(s, tab, "jurusan_id", "major_id")
	tingkatCol := colOr(s, tab, "tingkat", "grade", "level")
	rombelCol := colOr(s, tab, "rombel", "class_group", "group")
	namaCol := colOr(s, tab, "nama", "name")
	if tahunCol == "" || jurCol == "" || tingkatCol == "" || rombelCol == "" || namaCol == "" {
		return classMajor, nil
	}

	// v1 tahun_ajaran_id -> tahun_mulai.
	yearStart := map[string]int{}
	if s.tables["tahun_ajaran"] {
		if rows, err := v1.Query(ctx, `SELECT id::text, tahun_mulai FROM tahun_ajaran`); err == nil {
			for rows.Next() {
				var id string
				var y int
				if err := rows.Scan(&id, &y); err == nil {
					yearStart[id] = y
				}
			}
			rows.Close()
		}
	}
	// v2 academic_years: start_year -> id.
	yearID := map[int]string{}
	if rows, err := v2.Query(ctx, `SELECT start_year, id::text FROM academic_years`); err == nil {
		for rows.Next() {
			var y int
			var id string
			if err := rows.Scan(&y, &id); err == nil {
				yearID[y] = id
			}
		}
		rows.Close()
	}
	if len(yearID) == 0 {
		return classMajor, fmt.Errorf("academic_years v2 kosong — jalankan seed dulu (academic_years & majors diisi seed, bukan migrasi)")
	}

	rows, err := v1.Query(ctx, fmt.Sprintf(
		`SELECT id, %s, %s, %s, %s, %s FROM %s ORDER BY id`,
		tahunCol, jurCol, tingkatCol, rombelCol, namaCol, tab))
	if err != nil {
		return classMajor, err
	}
	defer rows.Close()
	for rows.Next() {
		var srcID, tahun, jur, tingkat, rombel, nama any
		if err := rows.Scan(&srcID, &tahun, &jur, &tingkat, &rombel, &nama); err != nil {
			return classMajor, err
		}
		src := uuidString(srcID)
		grade := toInt(tingkat)
		if !validClassGrade(grade) {
			st.Skipped["class_grade_invalid"]++
			continue
		}
		group := strings.ToUpper(strings.TrimSpace(fmt.Sprint(rombel)))
		if !validClassGroup(group) {
			st.Skipped["class_group_invalid"]++
			continue
		}
		startYear, ok := yearStart[uuidString(tahun)]
		if !ok {
			st.Skipped["class_tahun_tak_ada"]++
			continue
		}
		ayID, ok := yearID[startYear]
		if !ok {
			st.Skipped["class_tahun_v2_tak_ada"]++
			continue
		}
		majorID, ok := mappedID(ctx, v2, "majors", uuidString(jur), "majors")
		if !ok {
			st.Skipped["class_major_tak_ada"]++
			continue
		}
		name := strings.TrimSpace(fmt.Sprint(nama))
		if name == "" || name == "<nil>" {
			st.Skipped["class_nama_kosong"]++
			continue
		}

		targetID := ""
		if o.dryRun {
			targetID = "dry-class-" + src
		} else {
			if err := v2.QueryRow(ctx,
				`INSERT INTO classes (academic_year_id, major_id, grade, class_group, name)
				 VALUES ($1,$2,$3,$4,$5)
				 ON CONFLICT (academic_year_id, major_id, grade, class_group)
				 DO UPDATE SET name=EXCLUDED.name
				 RETURNING id::text`, ayID, majorID, grade, group, name).Scan(&targetID); err != nil {
				return classMajor, fmt.Errorf("buat class %s: %w", name, err)
			}
		}
		if err := saveMap(ctx, v2, o.dryRun, "classes", src, "classes", targetID); err != nil {
			return classMajor, err
		}
		classMajor[src] = majorID
		st.Classes++
	}
	slog.Info("classes dipetakan", "total", st.Classes)
	return classMajor, rows.Err()
}

// migrateEnrollments membuat enrollments dari users.kelas_id.
//
// Tahun akademiknya diambil dari kelas milik user itu (bukan "tahun aktif"),
// supaya anggota kelas tahun 2024 tidak dipaksa jadi anggota tahun aktif.
//
// Status enrollment mengikuti status user: yang sudah 'lulus' di v1 jadi
// 'graduated', sisanya 'enrolled'. Nilai 'active' yang pernah dipakai di sini
// TIDAK ada di CHECK constraint v2 — jadi insertnya gagal dan enrollment
// kosong tanpa ada yang memberitahu.
func migrateEnrollments(ctx context.Context, v1, v2 *pgxpool.Pool, s *v1schema, o options, st *stats) error {
	if !s.tables["users"] {
		return nil
	}
	kelasCol := colOr(s, "users", "kelas_id", "class_id")
	if kelasCol == "" {
		return nil
	}
	// Tahun akademik per kelas (satu query, bukan per user).
	classYear := map[string]string{}
	if rows, err := v2.Query(ctx, `SELECT id::text, academic_year_id::text FROM classes`); err == nil {
		for rows.Next() {
			var id, ay string
			if err := rows.Scan(&id, &ay); err == nil {
				classYear[id] = ay
			}
		}
		rows.Close()
	}
	var activeYear string
	_ = v2.QueryRow(ctx, `SELECT id::text FROM academic_years WHERE is_active ORDER BY start_year DESC LIMIT 1`).Scan(&activeYear)

	rows, err := v1.Query(ctx, fmt.Sprintf(`SELECT id, %s FROM users WHERE %s IS NOT NULL`, kelasCol, kelasCol))
	if err != nil {
		return err
	}
	defer rows.Close()

	made, skipped := 0, 0
	for rows.Next() {
		var srcID, kelas any
		if err := rows.Scan(&srcID, &kelas); err != nil {
			return err
		}
		classV2, ok := mappedID(ctx, v2, "classes", uuidString(kelas), "classes")
		if !ok {
			skipped++
			continue
		}
		if o.dryRun {
			// Dry-run: tidak menulis, tapi tetap dihitung supaya operator
			// tahu berapa enrollment yang akan dibuat.
			made++
			continue
		}
		targetID, ok := mappedID(ctx, v2, "users", uuidString(srcID), "users")
		if !ok {
			skipped++
			continue
		}
		ayID := classYear[classV2]
		if ayID == "" {
			ayID = activeYear
		}
		if ayID == "" {
			skipped++
			continue
		}
		if _, err := v2.Exec(ctx, `INSERT INTO enrollments (user_id, class_id, academic_year_id, status, is_current)
			SELECT $1, $2, $3,
			       CASE WHEN u.status='lulus' THEN 'graduated' ELSE 'enrolled' END,
			       TRUE
			FROM users u WHERE u.id=$1
			ON CONFLICT DO NOTHING`, targetID, classV2, ayID); err != nil {
			return err
		}
		made++
	}
	st.Enrollments = made
	if skipped > 0 {
		st.Skipped["enrollment_tak_ada_kelas"] += skipped
	}
	slog.Info("enrollments dibuat dari users.kelas_id", "count", made, "dilewati", skipped)
	return rows.Err()
}

// validClassGrade cek CHECK constraint v2 (grade = 10/11/12).
func validClassGrade(g int) bool { return g == 10 || g == 11 || g == 12 }

// validClassGroup cek CHECK constraint v2 (class_group satu huruf kapital).
func validClassGroup(g string) bool {
	return len(g) == 1 && g[0] >= 'A' && g[0] <= 'Z'
}

// slugify mengubah teks menjadi slug aman untuk kolom majors.code.
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}
