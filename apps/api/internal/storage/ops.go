package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Store adalah interface yang dipakai handler & service.
//
// PRD §7.5 ayat 3: "Handler/service hanya panggil interface Storage" — jadi
// dependency diarahkan ke interface, bukan ke kode S3-nya. Dengan begitu
// (a) paket handler tidak perlu tahu endpoint/region/path-style sama sekali,
// dan (b) test bisa memakai fake tanpa menyentuh object storage.
//
// Kode S3 yang sungguhan tetap satu: internal/storage/s3.go.
type Store interface {
	// Put menyimpan objek dengan content type tertentu.
	Put(ctx context.Context, key, contentType string, body io.Reader) error
	// Get membuka objek untuk dibaca. Caller menutup hasilnya.
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	// Size mengembalikan ukuran objek tanpa membacanya seluruhnya.
	Size(ctx context.Context, key string) (int64, error)
	// Delete menghapus satu objek.
	Delete(ctx context.Context, key string) error
	// DeleteObjects menghapus beberapa objek dalam satu permintaan.
	DeleteObjects(ctx context.Context, keys []string) error
	// ListObjectsV2 mencantumkan objek dengan awalan tertentu.
	ListObjectsV2(ctx context.Context, prefix string, maxKeys int32) ([]ObjectInfo, error)
	// Health memeriksa apakah bucket bisa dihubungi.
	Health(ctx context.Context) error
}

// ObjectInfo adalah ringkasan objek hasil ListObjectsV2.
type ObjectInfo struct {
	Key          string
	Size         int64
	LastModified time.Time
	ETag         string
	StorageClass string
}

// deleteObjectsMax adalah batas hard S3 untuk DeleteObjects per permintaan.
const deleteObjectsMax = 1000

// DeleteObjects menghapus banyak objek sekaligus.
//
// Dipakai worker cleanup & migrasi: menghapus thousands of file satu per satu
// berarti thousands of request, yang lambat dan mahal. DeleteObjects mengirim
// maksimal 1000 key per permintaan (batas S3), sisanya dipecah otomatis.
//
// Objek yang sudah hilang dianggap sukses: S3 DeleteObjects diam-diam
// 成功了 untuk key yang tidak ada, dan itu persis yang kita mau saat
// menjalankan cleanup dua kali.
func (s *Storage) DeleteObjects(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	for start := 0; start < len(keys); start += deleteObjectsMax {
		end := start + deleteObjectsMax
		if end > len(keys) {
			end = len(keys)
		}
		batch := make([]types.ObjectIdentifier, 0, end-start)
		for _, k := range keys[start:end] {
			if k == "" {
				continue
			}
			batch = append(batch, types.ObjectIdentifier{Key: aws.String(k)})
		}
		if len(batch) == 0 {
			continue
		}
		_, err := s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(s.bucket),
			Delete: &types.Delete{Objects: batch, Quiet: aws.Bool(true)},
		})
		if err != nil {
			return fmt.Errorf("storage: delete %d objek: %w", len(batch), err)
		}
	}
	return nil
}

// ListObjectsV2 mencantumkan objek dengan awalan prefix.
//
// Dipakai tooling migrasi & audit: untuk mencocokkan isi bucket dengan daftar
// objek yang tercatat di database.
//
// maxKeys <= 0 berarti "tanpa batas praktis" (dipakai paginator sampai habis).
// Fungsi ini mengembalikan seluruh hasil, bukan satu halaman.
func (s *Storage) ListObjectsV2(ctx context.Context, prefix string, maxKeys int32) ([]ObjectInfo, error) {
	in := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
	}
	if prefix != "" {
		in.Prefix = aws.String(prefix)
	}
	if maxKeys > 0 {
		in.MaxKeys = aws.Int32(maxKeys)
	}

	var out []ObjectInfo
	p := s3.NewListObjectsV2Paginator(s.client, in)
	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("storage: list objects (prefix %q): %w", prefix, err)
		}
		for _, o := range page.Contents {
			if o.Key == nil {
				continue
			}
			info := ObjectInfo{Key: *o.Key}
			if o.Size != nil {
				info.Size = *o.Size
			}
			if o.LastModified != nil {
				info.LastModified = *o.LastModified
			}
			if o.ETag != nil {
				info.ETag = strings.Trim(*o.ETag, `"`)
			}
			if o.StorageClass != "" {
				info.StorageClass = string(o.StorageClass)
			}
			out = append(out, info)
		}
		if maxKeys > 0 && int32(len(out)) >= maxKeys {
			break
		}
	}
	if maxKeys > 0 && int32(len(out)) > maxKeys {
		out = out[:maxKeys]
	}
	return out, nil
}

// PresignGet membuat URL bertanda tangan sementara untuk satu objek.
//
// PRD §7.5 ayat 3: presign hanya untuk kebutuhan internal/tooling, bukan URL
// publik ke browser — URL yang diberikan ke browser tetap lewat BFF.
func (s *Storage) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if s.presign == nil {
		return "", fmt.Errorf("storage: presign tidak tersedia")
	}
	req, err := s.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("storage: presign %q: %w", key, err)
	}
	return req.URL, nil
}

// Compile-time check: kode S3 harus memenuhi interface yang dipakai
// handler. Kalau ada metode yang berubah tanda tangannya, compile time gagal
// di sini — bukan saat runtime di produksi.
var _ Store = (*Storage)(nil)
