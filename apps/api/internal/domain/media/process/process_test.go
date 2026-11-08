package process

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"log/slog"
	"math"
	"os"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/davidbyttow/govips/v2/vips"
	"github.com/go-pdf/fpdf"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/media"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/storage"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/testutil"
)

func TestMain(m *testing.M) {
	if err := Startup(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	code := m.Run()
	Shutdown()
	os.Exit(code)
}

type processEnv struct {
	t    *testing.T
	pool *pgxpool.Pool
	st   *store.Store
	s3   *storage.Client
	p    *Processor
}

func newProcessEnv(t *testing.T) *processEnv {
	t.Helper()

	pool := testutil.NewSeededDB(t)
	s3Client := testutil.NewS3(t)
	st := store.NewStore(pool)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	return &processEnv{
		t:    t,
		pool: pool,
		st:   st,
		s3:   s3Client,
		p:    NewProcessor(st, s3Client, logger),
	}
}

// insertAsset writes the original to the private bucket and records the media
// row exactly like an upload would.
func (e *processEnv) insertAsset(kind string, mimeType string, data []byte) uuid.UUID {
	e.t.Helper()

	objectID, err := uuid.NewV7()
	if err != nil {
		e.t.Fatalf("buat id objek: %v", err)
	}
	key := fmt.Sprintf("orig/test/%s", objectID)

	ctx := context.Background()
	if err := e.s3.PutObject(ctx, e.s3.PrivateBucket, key, bytes.NewReader(data), int64(len(data)), mimeType); err != nil {
		e.t.Fatalf("unggah objek asli: %v", err)
	}

	row, err := e.st.InsertMediaAsset(ctx, store.InsertMediaAssetParams{
		Kind:        kind,
		Purpose:     "content",
		OriginalKey: key,
		MimeType:    mimeType,
		SizeBytes:   int64(len(data)),
	})
	if err != nil {
		e.t.Fatalf("simpan aset media: %v", err)
	}
	return row.ID
}

func (e *processEnv) asset(id uuid.UUID) store.MediaAsset {
	e.t.Helper()

	row, err := e.st.GetMediaAssetForProcessing(context.Background(), id)
	if err != nil {
		e.t.Fatalf("baca aset media: %v", err)
	}
	return row
}

func (e *processEnv) variants(row store.MediaAsset) map[string]media.Variant {
	e.t.Helper()

	variants := map[string]media.Variant{}
	if err := json.Unmarshal(row.Variants, &variants); err != nil {
		e.t.Fatalf("urai variants: %v", err)
	}
	return variants
}

func TestProcess_JPEGProducesResponsiveVariants(t *testing.T) {
	env := newProcessEnv(t)
	source := jpegFixture(t, 2000, 1000)
	id := env.insertAsset("image", "image/jpeg", source)

	if err := env.p.Process(context.Background(), id); err != nil {
		t.Fatalf("proses gambar: %v", err)
	}

	row := env.asset(id)
	if row.Status != "ready" {
		t.Fatalf("status = %q, mau ready (failure_reason=%v)", row.Status, row.FailureReason)
	}
	if row.Width == nil || *row.Width != 2000 || row.Height == nil || *row.Height != 1000 {
		t.Fatalf("dimensi asli = %v x %v, mau 2000 x 1000", row.Width, row.Height)
	}
	if row.ProcessedAt == nil {
		t.Fatal("processed_at belum diisi")
	}
	if row.Blurhash == nil || *row.Blurhash == "" {
		t.Fatal("blurhash kosong")
	}
	if row.DominantColor == nil || !regexp.MustCompile(`^#[0-9a-f]{6}$`).MatchString(*row.DominantColor) {
		t.Fatalf("dominant_color = %v, mau format #rrggbb", row.DominantColor)
	}

	variants := env.variants(row)
	if len(variants) != 10 {
		t.Fatalf("jumlah varian = %d, mau 10: %v", len(variants), variants)
	}

	contentTypes := map[string]string{"webp": "image/webp", "avif": "image/avif"}
	formats := map[string]vips.ImageType{"webp": vips.ImageTypeWEBP, "avif": vips.ImageTypeAVIF}

	for _, width := range []int{320, 640, 960, 1280, 1920} {
		for _, format := range []string{"webp", "avif"} {
			name := fmt.Sprintf("w%d.%s", width, format)
			variant, ok := variants[name]
			if !ok {
				t.Fatalf("varian %s tidak ada", name)
			}
			if variant.Format != format {
				t.Errorf("%s format = %q, mau %q", name, variant.Format, format)
			}
			if variant.W != width {
				t.Errorf("%s lebar = %d, mau %d", name, variant.W, width)
			}
			wantHeight := int(math.Round(1000 * float64(width) / 2000))
			if variant.H != wantHeight {
				t.Errorf("%s tinggi = %d, mau %d", name, variant.H, wantHeight)
			}
			if variant.Key != fmt.Sprintf("var/%s/%s", id, name) {
				t.Errorf("%s key = %q", name, variant.Key)
			}

			reader, info, err := env.s3.GetObject(context.Background(), env.s3.PrivateBucket, variant.Key)
			if err != nil {
				t.Fatalf("baca objek %s: %v", name, err)
			}
			data, err := io.ReadAll(reader)
			_ = reader.Close()
			if err != nil {
				t.Fatalf("baca isi %s: %v", name, err)
			}

			if info.ContentType != contentTypes[format] {
				t.Errorf("%s content type = %q, mau %q", name, info.ContentType, contentTypes[format])
			}
			if int64(len(data)) != variant.Bytes {
				t.Errorf("%s bytes = %d, metadata %d", name, len(data), variant.Bytes)
			}
			if bytes.Contains(data, []byte("Exif")) {
				t.Errorf("%s masih mengandung metadata EXIF", name)
			}

			decoded, err := vips.NewImageFromBuffer(data)
			if err != nil {
				t.Fatalf("urai ulang %s: %v", name, err)
			}
			metadata := decoded.Metadata()
			decodedWidth, decodedHeight := decoded.Width(), decoded.Height()
			decoded.Close()

			if metadata.Format != formats[format] {
				t.Errorf("%s terurai sebagai %v, mau %v", name, metadata.Format, formats[format])
			}
			if decodedWidth != variant.W || decodedHeight != variant.H {
				t.Errorf("%s terurai %dx%d, metadata %dx%d", name, decodedWidth, decodedHeight, variant.W, variant.H)
			}
		}
	}
}

func TestProcess_NarrowPNGUsesItsOwnWidth(t *testing.T) {
	env := newProcessEnv(t)
	id := env.insertAsset("image", "image/png", pngFixture(t, 200, 100))

	if err := env.p.Process(context.Background(), id); err != nil {
		t.Fatalf("proses png: %v", err)
	}

	row := env.asset(id)
	if row.Status != "ready" {
		t.Fatalf("status = %q, mau ready (failure_reason=%v)", row.Status, row.FailureReason)
	}
	if row.Width == nil || *row.Width != 200 || row.Height == nil || *row.Height != 100 {
		t.Fatalf("dimensi = %v x %v, mau 200 x 100", row.Width, row.Height)
	}

	variants := env.variants(row)
	if len(variants) != 2 {
		t.Fatalf("jumlah varian = %d, mau 2: %v", len(variants), variants)
	}
	for _, name := range []string{"w200.webp", "w200.avif"} {
		variant, ok := variants[name]
		if !ok {
			t.Fatalf("varian %s tidak ada", name)
		}
		if variant.W != 200 || variant.H != 100 {
			t.Errorf("%s = %dx%d, mau 200x100", name, variant.W, variant.H)
		}
	}

	// Reprocessing a ready asset must overwrite the same keys and stay ready.
	if err := env.p.Process(context.Background(), id); err != nil {
		t.Fatalf("proses ulang png: %v", err)
	}
	row = env.asset(id)
	if row.Status != "ready" || len(env.variants(row)) != 2 {
		t.Fatalf("setelah proses ulang: status=%q varian=%d", row.Status, len(env.variants(row)))
	}
}

func TestProcess_PDFRendersFirstPagePreview(t *testing.T) {
	env := newProcessEnv(t)
	id := env.insertAsset("pdf", "application/pdf", pdfFixture(t))

	if err := env.p.Process(context.Background(), id); err != nil {
		t.Fatalf("proses pdf: %v", err)
	}

	row := env.asset(id)
	if row.Status != "ready" {
		t.Fatalf("status = %q, mau ready (failure_reason=%v)", row.Status, row.FailureReason)
	}
	if row.PageCount == nil || *row.PageCount != 1 {
		t.Fatalf("page_count = %v, mau 1", row.PageCount)
	}
	if row.Width == nil || *row.Width <= 0 || row.Height == nil || *row.Height <= 0 {
		t.Fatalf("dimensi pratinjau = %v x %v", row.Width, row.Height)
	}
	if row.Blurhash == nil || *row.Blurhash == "" {
		t.Fatal("blurhash pratinjau kosong")
	}
	if row.DominantColor == nil || !regexp.MustCompile(`^#[0-9a-f]{6}$`).MatchString(*row.DominantColor) {
		t.Fatalf("dominant_color = %v, mau format #rrggbb", row.DominantColor)
	}

	variants := env.variants(row)
	if len(variants) != 2 {
		t.Fatalf("jumlah varian = %d, mau 2: %v", len(variants), variants)
	}
	base := fmt.Sprintf("w%d", *row.Width)
	for _, format := range []string{"webp", "avif"} {
		name := base + "." + format
		if _, ok := variants[name]; !ok {
			t.Fatalf("varian pratinjau %s tidak ada: %v", name, variants)
		}
	}
}

func TestProcess_ValidGLBHasNoVariants(t *testing.T) {
	env := newProcessEnv(t)
	id := env.insertAsset("model3d", "model/gltf-binary", glbFixture(t, true))

	if err := env.p.Process(context.Background(), id); err != nil {
		t.Fatalf("proses glb: %v", err)
	}

	row := env.asset(id)
	if row.Status != "ready" {
		t.Fatalf("status = %q, mau ready (failure_reason=%v)", row.Status, row.FailureReason)
	}
	if variants := env.variants(row); len(variants) != 0 {
		t.Fatalf("glb menghasilkan varian: %v", variants)
	}
}

func TestProcess_InvalidGLBFailsOnFinalAttempt(t *testing.T) {
	env := newProcessEnv(t)
	id := env.insertAsset("model3d", "model/gltf-binary", glbFixture(t, false))

	worker := NewWorker(env.p)
	job := &river.Job[media.ProcessArgs]{
		JobRow: &rivertype.JobRow{Attempt: 5, MaxAttempts: 5},
		Args:   media.ProcessArgs{AssetID: id},
	}
	if err := worker.Work(context.Background(), job); err != nil {
		t.Fatalf("percobaan terakhir harus menelan error, dapat %v", err)
	}

	row := env.asset(id)
	if row.Status != "failed" {
		t.Fatalf("status = %q, mau failed", row.Status)
	}
	if row.FailureReason == nil || *row.FailureReason == "" {
		t.Fatal("failure_reason kosong")
	}
}

func TestProcess_DeletedAssetIsNoOp(t *testing.T) {
	env := newProcessEnv(t)
	id := env.insertAsset("image", "image/png", pngFixture(t, 200, 100))

	if _, err := env.pool.Exec(context.Background(),
		"UPDATE media_assets SET deleted_at = now() WHERE id = $1", id); err != nil {
		t.Fatalf("hapus aset: %v", err)
	}

	if err := env.p.Process(context.Background(), id); err != nil {
		t.Fatalf("aset terhapus harus no-op, dapat %v", err)
	}

	var status string
	if err := env.pool.QueryRow(context.Background(),
		"SELECT status FROM media_assets WHERE id = $1", id).Scan(&status); err != nil {
		t.Fatalf("baca status: %v", err)
	}
	if status != "uploaded" {
		t.Fatalf("status = %q, mau tetap uploaded", status)
	}

	if _, err := env.s3.StatObject(context.Background(), env.s3.PrivateBucket,
		fmt.Sprintf("var/%s/w200.webp", id)); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("varian untuk aset terhapus seharusnya tidak ada, dapat %v", err)
	}
}

func TestFailureReason_IsSingleLineAndShort(t *testing.T) {
	long := failureReason(errors.New("baris satu\nbaris dua\t" + string(bytes.Repeat([]byte("x"), 400))))
	if len(long) > failureReasonLimit {
		t.Fatalf("panjang alasan = %d, mau <= %d", len(long), failureReasonLimit)
	}
	if bytes.ContainsAny([]byte(long), "\n\t") {
		t.Fatalf("alasan masih multi-baris: %q", long)
	}

	multibyte := failureReason(errors.New(strings.Repeat("é", 400)))
	if len(multibyte) > failureReasonLimit {
		t.Fatalf("panjang alasan multibyte = %d, mau <= %d", len(multibyte), failureReasonLimit)
	}
	if !utf8.ValidString(multibyte) {
		t.Fatalf("alasan multibyte bukan UTF-8 valid: %q", multibyte)
	}
}

func jpegFixture(t *testing.T, width, height int) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(x * 255 / max(width-1, 1)),
				G: uint8(y * 255 / max(height-1, 1)),
				B: 128,
				A: 255,
			})
		}
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("buat jpeg uji: %v", err)
	}
	return buf.Bytes()
}

func pngFixture(t *testing.T, width, height int) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 20, G: 120, B: 220, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("buat png uji: %v", err)
	}
	return buf.Bytes()
}

func pdfFixture(t *testing.T) []byte {
	t.Helper()

	pdf := fpdf.NewCustom(&fpdf.InitType{
		OrientationStr: "P",
		UnitStr:        "pt",
		Size:           fpdf.SizeType{Wd: 200, Ht: 100},
	})
	pdf.SetFillColor(42, 120, 200)
	pdf.AddPage()
	pdf.Rect(0, 0, 200, 100, "F")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		t.Fatalf("buat pdf uji: %v", err)
	}
	return buf.Bytes()
}

func glbFixture(t *testing.T, valid bool) []byte {
	t.Helper()

	data := make([]byte, 32)
	copy(data[0:4], "glTF")
	binary.LittleEndian.PutUint32(data[4:8], 2)
	length := uint32(len(data))
	if !valid {
		length = 4096
	}
	binary.LittleEndian.PutUint32(data[8:12], length)
	return data
}
