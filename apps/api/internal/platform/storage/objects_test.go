package storage_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/google/uuid"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/storage"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/testutil"
)

func TestObjects_CRUDAndCopy(t *testing.T) {
	ctx := context.Background()
	client := testutil.NewS3(t)

	key := "orig/2026/11/" + uuid.NewString()
	content := []byte("isi berkas uji")
	if err := client.PutObject(ctx, client.PrivateBucket, key, bytes.NewReader(content), int64(len(content)), "text/plain"); err != nil {
		t.Fatalf("PutObject: %v", err)
	}

	reader, info, err := client.GetObject(ctx, client.PrivateBucket, key)
	if err != nil {
		t.Fatalf("GetObject: %v", err)
	}
	defer reader.Close()

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("baca objek: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("isi objek = %q, mau %q", got, content)
	}
	if info.Size != int64(len(content)) {
		t.Errorf("Size = %d, mau %d", info.Size, len(content))
	}
	if info.ContentType != "text/plain" {
		t.Errorf("ContentType = %q, mau text/plain", info.ContentType)
	}
	if info.ETag == "" {
		t.Error("ETag kosong")
	}

	stat, err := client.StatObject(ctx, client.PrivateBucket, key)
	if err != nil {
		t.Fatalf("StatObject: %v", err)
	}
	if stat.Size != info.Size || stat.ETag != info.ETag {
		t.Errorf("StatObject = %+v, mau sama dengan %+v", stat, info)
	}

	copiedKey := "copy/" + uuid.NewString()
	if err := client.CopyObject(ctx, client.PrivateBucket, key, client.PublicBucket, copiedKey); err != nil {
		t.Fatalf("CopyObject: %v", err)
	}

	copied, copiedInfo, err := client.GetObject(ctx, client.PublicBucket, copiedKey)
	if err != nil {
		t.Fatalf("GetObject hasil salin: %v", err)
	}
	defer copied.Close()
	copiedBytes, err := io.ReadAll(copied)
	if err != nil {
		t.Fatalf("baca hasil salin: %v", err)
	}
	if !bytes.Equal(copiedBytes, content) {
		t.Fatalf("isi hasil salin = %q, mau %q", copiedBytes, content)
	}
	if copiedInfo.Size != int64(len(content)) {
		t.Errorf("ukuran hasil salin = %d, mau %d", copiedInfo.Size, len(content))
	}

	if err := client.DeleteObject(ctx, client.PrivateBucket, key); err != nil {
		t.Fatalf("DeleteObject: %v", err)
	}
	if _, _, err := client.GetObject(ctx, client.PrivateBucket, key); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("GetObject setelah hapus error = %v, mau ErrNotFound", err)
	}
}

func TestGetObject_MissingReturnsErrNotFound(t *testing.T) {
	ctx := context.Background()
	client := testutil.NewS3(t)

	if _, _, err := client.GetObject(ctx, client.PrivateBucket, "tidak/ada"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("GetObject error = %v, mau ErrNotFound", err)
	}
	if _, err := client.StatObject(ctx, client.PrivateBucket, "tidak/ada"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("StatObject error = %v, mau ErrNotFound", err)
	}
	if err := client.CopyObject(ctx, client.PrivateBucket, "tidak/ada", client.PublicBucket, "salin"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("CopyObject error = %v, mau ErrNotFound", err)
	}
}

func TestDeleteObject_MissingIsNotAnError(t *testing.T) {
	ctx := context.Background()
	client := testutil.NewS3(t)

	if err := client.DeleteObject(ctx, client.PrivateBucket, "tidak/ada"); err != nil {
		t.Fatalf("DeleteObject error = %v, mau nil", err)
	}
}
