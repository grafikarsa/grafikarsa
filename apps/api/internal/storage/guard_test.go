//nolint:misspell // Komentar dalam bahasa Indonesia; "implementasi" bukan salah ketik.
package storage_test

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// Guard kontrak S3-agnostic (PRD §7.5).
//
// PRD §7.5 ayat 2 menyebut daftar operasi S3 yang BOLEH dipakai:
//
//	PutObject, GetObject, HeadObject, DeleteObject, DeleteObjects,
//	ListObjectsV2, CreateBucket, HeadBucket, PutBucketVersioning
//
// Kalimatnya: "Dilarang pakai API admin/ekstensi vendor dari kode aplikasi."
//
// Tanpa guard, daftar itu cuma tertulis di dokumen — tidak ada yang mencegah
//有人 memanggil `s3.PutObjectAcl` atau `s3.DeleteBucket` di tengah coding.
// Test ini menjadikannya aturan yang ditegakkan: begitu ada operasi di luar
// daftar, atau ada import SDK di luar internal/storage, atau ada env var
// spesifik vendor, test gagal.
//
// Guard ini murah (hanya membaca file sumber) tapi menangkap pelanggaran
// yang paling mahal: kode yang hanya jalan di satu vendor storage.

var allowedS3Ops = map[string]bool{
	"PutObject":           true,
	"GetObject":           true,
	"HeadObject":          true,
	"DeleteObject":        true,
	"DeleteObjects":       true,
	"ListObjectsV2":       true,
	"CreateBucket":        true,
	"HeadBucket":          true,
	"PutBucketVersioning": true,
	// Helper SDK, bukan operasi pada bucket:
	"NewListObjectsV2Paginator": true,
	"NewPresignClient":          true,
	"NewDeleteObjectsPaginator": true,
	// Presign adalah GetObject yang ditandatangani (PRD §7.5 ayat 3
	// mengizinkan presign untuk kebutuhan internal/tooling).
	"PresignGetObject": true,
}

// repoRoot mencari akar repo dari lokasi file test ini.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("tidak bisa tahu lokasi file test")
	}
	// .../apps/api/internal/storage/guard_test.go -> akar repo
	root := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "go.work")); err == nil {
		return root
	}
	if _, err := os.Stat(filepath.Join(root, "apps", "api", "go.mod")); err == nil {
		return root
	}
	t.Fatalf("akar repo tidak ditemukan dari %s", thisFile)
	return ""
}

// goFiles mengembalikan seluruh file .go di bawah dir (lewat vendor & .git).
func goFiles(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			base := info.Name()
			if base == "vendor" || base == ".git" || base == "node_modules" || base == ".next" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			out = append(out, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("gagal menelusuri %s: %v", dir, err)
	}
	return out
}

// TestGuardSDKStorageHanyaDiPaketStorage memastikan tidak ada kode di luar
// internal/storage yang mengimpor SDK S3 atau SDK vendor.
//
// PRD §7.5 ayat 1 & 3: hanya `internal/storage/s3.go` yang tahu
// endpoint/region/path-style; handler/service hanya boleh bicara lewat
// interface.
func TestGuardSDKStorageHanyaDiPaketStorage(t *testing.T) {
	root := repoRoot(t)
	vendorSDK := regexp.MustCompile(`"github\.com/minio/minio-go|"github\.com/rustfs/rustfs-sdk|minio-go/v7|rustfs-sdk-go`)
	s3SDK := regexp.MustCompile(`aws-sdk-go-v2/service/s3"`)

	for _, file := range goFiles(t, filepath.Join(root, "apps", "api")) {
		rel, _ := filepath.Rel(root, file)
		if strings.Contains(rel, "internal/storage/") {
			continue
		}
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("gagal membaca %s: %v", rel, err)
		}
		src := string(b)
		if s3SDK.MatchString(src) {
			t.Errorf("%s mengimpor SDK S3 secara langsung; hanya internal/storage yang boleh tahu soal storage", rel)
		}
		if vendorSDK.MatchString(src) {
			t.Errorf("%s mengimpor SDK storage spesifik vendor; PRD §7.5 ayat 1 melarangnya di luar internal/storage", rel)
		}
	}
}

// TestGuardOperasiS3HanyaYangDiizinkan memastikan paket internal/storage
// hanya memanggil operasi S3 dari daftar PRD §7.5.
//
// Polymorph: pemanggilan ditulis `s.client.<Operasi>(` atau
// `s.presign.<Operasi>(`. Pola lain (mis. lewat variabel) akan lolos dari
// guard ini — itu batas wajar dari guard berbasis sumber.
func TestGuardOperasiS3HanyaYangDiizinkan(t *testing.T) {
	root := repoRoot(t)
	call := regexp.MustCompile(`s\.(?:client|presign)\.([A-Za-z0-9]+)\(`)
	allowed := map[string]bool{}
	for k, v := range allowedS3Ops {
		allowed[k] = v
	}

	for _, file := range goFiles(t, filepath.Join(root, "apps", "api", "internal", "storage")) {
		rel, _ := filepath.Rel(root, file)
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("gagal membaca %s: %v", rel, err)
		}
		for _, m := range call.FindAllStringSubmatch(string(b), -1) {
			op := m[1]
			if !allowed[op] {
				t.Errorf("%s memanggil operasi S3 %q yang tidak ada di daftar PRD §7.5 ayat 2", rel, op)
			}
		}
	}
}

// TestGuardEnvAgnistikStorage memastikan tidak ada env var spesifik vendor
// storage di konfigurasi yang dipakai aplikasi.
//
// PRD §7.5 ayat 4: "Tidak ada *_RUSTFS_* / *_MINIO_* di .env.example, kode,
// compose, atau CI."
//
// Wenamun di compose ada satu pengecualian yang tidak bisa dihindari: nama
// env milik server storage itu sendiri (RustFS membaca RUSTFS_*, MinIO
// membaca MINIO_ROOT_*). Yang dijaga guard ini adalah maksudnya, yaitu
// APLIKASI tidak boleh bergantung pada vendor: semua env di .env.example,
// di CI, dan di blok service selain `storage:` di compose harus memakai
// S3_* saja.
//
// Kalau suatu saat storage-nya diganti ke MinIO, yang berubah hanya blok
// `storage:` di compose (image + nama env server) — kode API tidak tersentuh.
func TestGuardEnvAgnistikStorage(t *testing.T) {
	root := repoRoot(t)
	vendorEnv := regexp.MustCompile(`(?i)(RUSTFS_|MINIO_)[A-Z_]*`)

	// Berkas yang wajib bebas total.
	strict := []string{
		filepath.Join(root, ".env.example"),
		filepath.Join(root, "apps", "web", ".env.example"),
		filepath.Join(root, ".github", "workflows", "ci.yml"),
		filepath.Join(root, ".github", "workflows", "deploy.yml"),
	}
	for _, f := range strict {
		if _, err := os.Stat(f); err != nil {
			continue
		}
		rel, _ := filepath.Rel(root, f)
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("gagal membaca %s: %v", rel, err)
		}
		for i, line := range strings.Split(string(b), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if vendorEnv.MatchString(line) {
				t.Errorf("%s baris %d memakai env var spesifik vendor storage: %q", rel, i+1, line)
			}
		}
	}

	// Compose: vendor env hanya boleh di dalam blok service `storage:`.
	compose := []string{
		filepath.Join(root, "docker-compose.yml"),
		filepath.Join(root, "docker-compose.deploy.yml"),
	}
	for _, f := range compose {
		if _, err := os.Stat(f); err != nil {
			continue
		}
		rel, _ := filepath.Rel(root, f)
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("gagal membaca %s: %v", rel, err)
		}
		service := ""
		for i, line := range strings.Split(string(b), "\n") {
			trimmed := strings.TrimSpace(line)
			// Baris "  <nama>:" pada indent 2 = awal blok service.
			if m := regexp.MustCompile(`^  ([a-zA-Z0-9_-]+):\s*$`).FindStringSubmatch(line); m != nil {
				service = m[1]
				continue
			}
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				continue
			}
			if !vendorEnv.MatchString(trimmed) {
				continue
			}
			if service != "storage" {
				t.Errorf("%s baris %d (service %q) memakai env var spesifik vendor storage: %q",
					rel, i+1, service, trimmed)
			}
		}
	}
}

// TestGuardInterfaceStorageDipakaiHandler memastikan handler bergantung pada
// interface, bukan implementasi konkret.
//
// PRD §7.5 ayat 3: "Handler/service hanya panggil interface Storage".
// Kalau handler memakai *storage.Storage langsung, perubahan vendor ikut
// merembet ke seluruh handler.
func TestGuardInterfaceStorageDipakaiHandler(t *testing.T) {
	root := repoRoot(t)
	concrete := regexp.MustCompile(`Storage\s+\*storage\.Storage`)
	for _, file := range goFiles(t, filepath.Join(root, "apps", "api", "internal", "http")) {
		rel, _ := filepath.Rel(root, file)
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("gagal membaca %s: %v", rel, err)
		}
		if concrete.MatchString(string(b)) {
			t.Errorf("%s memakai *storage.Storage (implementasi konkret); pakai interface storage.Store", rel)
		}
	}
}
