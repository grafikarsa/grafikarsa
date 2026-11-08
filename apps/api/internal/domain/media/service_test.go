package media

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"io"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/ratelimit"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/storage"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/testutil"
)

type fakeJobInserter struct {
	args []river.JobArgs
}

func (f *fakeJobInserter) InsertTx(_ context.Context, tx pgx.Tx, args river.JobArgs, _ *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	if tx == nil {
		return nil, errors.New("transaction is nil")
	}
	f.args = append(f.args, args)
	return &rivertype.JobInsertResult{}, nil
}

type mediaEnv struct {
	t      *testing.T
	pool   *pgxpool.Pool
	rdb    *redis.Client
	s3     *storage.Client
	st     *store.Store
	jobs   *fakeJobInserter
	svc    *Service
	userID uuid.UUID
}

func newMediaEnv(t *testing.T, contentBlocked bool) *mediaEnv {
	t.Helper()

	pool := testutil.NewSeededDB(t)
	rdb := testutil.NewValkey(t)
	s3Client := testutil.NewS3(t)
	st := store.NewStore(pool)
	jobInserter := &fakeJobInserter{}

	var blockedAt *time.Time
	if contentBlocked {
		now := time.Now().UTC()
		blockedAt = &now
	}

	var userID uuid.UUID
	err := pool.QueryRow(context.Background(), `
		INSERT INTO users (
			account_type, status, username, display_name, password_hash,
			password_kind, activated_at, content_blocked_at
		) VALUES ('staff', 'active', 'pengunggah', 'Pengunggah Uji', 'x', 'user', now(), $1)
		RETURNING id`, blockedAt,
	).Scan(&userID)
	if err != nil {
		t.Fatalf("buat pengguna uji: %v", err)
	}

	return &mediaEnv{
		t:      t,
		pool:   pool,
		rdb:    rdb,
		s3:     s3Client,
		st:     st,
		jobs:   jobInserter,
		svc:    NewService(st, s3Client, jobInserter, ratelimit.New(rdb), rdb),
		userID: userID,
	}
}

func (e *mediaEnv) principal() *auth.Principal {
	return &auth.Principal{
		UserID: e.userID,
		Grants: []auth.Grant{{Permission: permissionContentCreate, Global: true}},
	}
}

func (e *mediaEnv) upload(purpose Purpose, filename string, content []byte) (Asset, error) {
	return e.svc.Upload(context.Background(), e.principal(), purpose, filename, bytes.NewReader(content))
}

func (e *mediaEnv) countAssets() int {
	e.t.Helper()

	var count int
	if err := e.pool.QueryRow(context.Background(), `SELECT count(*) FROM media_assets`).Scan(&count); err != nil {
		e.t.Fatalf("hitung media_assets: %v", err)
	}
	return count
}

func (e *mediaEnv) countObjects() int {
	e.t.Helper()

	out, err := e.s3.S3.ListObjectsV2(context.Background(), &s3.ListObjectsV2Input{
		Bucket: aws.String(e.s3.PrivateBucket),
	})
	if err != nil {
		e.t.Fatalf("daftar objek: %v", err)
	}
	return len(out.Contents)
}

func tinyPNG(t *testing.T) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestUpload_StoresOriginalAndEnqueuesJob(t *testing.T) {
	env := newMediaEnv(t, false)
	content := tinyPNG(t)

	asset, err := env.upload(PurposeContent, "foto portfolio.png", content)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	if asset.Kind != KindImage || asset.Purpose != PurposeContent || asset.MimeType != "image/png" {
		t.Errorf("asset = %+v, mau kind image, purpose content, mime image/png", asset)
	}
	if asset.SizeBytes != int64(len(content)) {
		t.Errorf("SizeBytes = %d, mau %d", asset.SizeBytes, len(content))
	}
	if asset.Status != "uploaded" {
		t.Errorf("Status = %q, mau uploaded", asset.Status)
	}
	if asset.OwnerID != env.userID {
		t.Errorf("OwnerID = %s, mau %s", asset.OwnerID, env.userID)
	}
	if asset.OriginalFilename == nil || *asset.OriginalFilename != "foto portfolio.png" {
		t.Errorf("OriginalFilename = %v, mau foto portfolio.png", asset.OriginalFilename)
	}
	if asset.Variants == nil || len(asset.Variants) != 0 {
		t.Errorf("Variants = %v, mau map kosong", asset.Variants)
	}

	var (
		key      string
		size     int64
		checksum []byte
		status   string
	)
	err = env.pool.QueryRow(context.Background(), `
		SELECT original_key, size_bytes, checksum_sha256, status
		FROM media_assets WHERE id = $1`, asset.ID).
		Scan(&key, &size, &checksum, &status)
	if err != nil {
		t.Fatalf("baca media_assets: %v", err)
	}
	if size != int64(len(content)) || status != "uploaded" || len(checksum) != 32 {
		t.Errorf("baris media = size %d, status %s, checksum %d byte", size, status, len(checksum))
	}

	reader, _, err := env.s3.GetObject(context.Background(), env.s3.PrivateBucket, key)
	if err != nil {
		t.Fatalf("objek asli tidak ada: %v", err)
	}
	defer reader.Close()
	stored, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("baca objek asli: %v", err)
	}
	if !bytes.Equal(stored, content) {
		t.Error("isi objek asli berbeda dari unggahan")
	}

	if len(env.jobs.args) != 1 {
		t.Fatalf("job = %d, mau 1", len(env.jobs.args))
	}
	args, ok := env.jobs.args[0].(ProcessArgs)
	if !ok {
		t.Fatalf("job args = %T, mau media.ProcessArgs", env.jobs.args[0])
	}
	if args.AssetID != asset.ID {
		t.Errorf("job asset id = %s, mau %s", args.AssetID, asset.ID)
	}
	if env.jobs.args[0].Kind() != "media.process" {
		t.Errorf("job kind = %q, mau media.process", env.jobs.args[0].Kind())
	}
}

func TestUpload_TextRenamedPNGReturnsUnsupportedMediaType(t *testing.T) {
	env := newMediaEnv(t, false)

	_, err := env.upload(PurposeContent, "foto.png", []byte("ini hanya teks biasa, bukan gambar"))
	if !apperr.Is(err, "UNSUPPORTED_MEDIA_TYPE") {
		t.Fatalf("error = %v, mau UNSUPPORTED_MEDIA_TYPE", err)
	}
	if got := env.countAssets(); got != 0 {
		t.Errorf("media_assets = %d, mau 0", got)
	}
	if got := env.countObjects(); got != 0 {
		t.Errorf("objek = %d, mau 0", got)
	}
}

func TestUpload_OversizeReturnsPayloadTooLargeAndLeavesNoObject(t *testing.T) {
	env := newMediaEnv(t, false)
	if _, err := env.st.UpsertSetting(context.Background(), store.UpsertSettingParams{
		Key:   "media.max_image_mb",
		Value: json.RawMessage(`1`),
	}); err != nil {
		t.Fatalf("set media.max_image_mb: %v", err)
	}

	// A valid PNG header followed by more than the 1 MB cap.
	content := append(tinyPNG(t), make([]byte, 1536*1024)...)

	_, err := env.upload(PurposeContent, "besar.png", content)
	if !apperr.Is(err, "PAYLOAD_TOO_LARGE") {
		t.Fatalf("error = %v, mau PAYLOAD_TOO_LARGE", err)
	}
	if got := env.countAssets(); got != 0 {
		t.Errorf("media_assets = %d, mau 0", got)
	}
	if got := env.countObjects(); got != 0 {
		t.Errorf("objek = %d, mau 0", got)
	}
}

func TestUpload_DailyQuotaExceededReturnsLimitReached(t *testing.T) {
	env := newMediaEnv(t, false)
	if _, err := env.st.UpsertSetting(context.Background(), store.UpsertSettingParams{
		Key:   "media.daily_upload_mb",
		Value: json.RawMessage(`0`),
	}); err != nil {
		t.Fatalf("set media.daily_upload_mb: %v", err)
	}

	_, err := env.upload(PurposeContent, "foto.png", tinyPNG(t))
	if !apperr.Is(err, "LIMIT_REACHED") {
		t.Fatalf("error = %v, mau LIMIT_REACHED", err)
	}
	if got := env.countAssets(); got != 0 {
		t.Errorf("media_assets = %d, mau 0", got)
	}
	if got := env.countObjects(); got != 0 {
		t.Errorf("objek = %d, mau 0", got)
	}
}

func TestUpload_ContentBlockedAllowsAvatarButNotContent(t *testing.T) {
	env := newMediaEnv(t, true)
	blocked := &auth.Principal{UserID: env.userID}
	content := tinyPNG(t)

	if _, err := env.svc.Upload(context.Background(), blocked, PurposeAvatar, "avatar.png", bytes.NewReader(content)); err != nil {
		t.Fatalf("Upload avatar: %v", err)
	}

	_, err := env.svc.Upload(context.Background(), blocked, PurposeContent, "konten.png", bytes.NewReader(content))
	if !apperr.Is(err, "CONTENT_BLOCKED") {
		t.Fatalf("Upload content error = %v, mau CONTENT_BLOCKED", err)
	}
}

func TestGet_OtherOwnerReturnsNotFound(t *testing.T) {
	env := newMediaEnv(t, false)
	asset, err := env.upload(PurposeContent, "foto.png", tinyPNG(t))
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	otherID := env.userID
	if err := env.pool.QueryRow(context.Background(), `
		INSERT INTO users (account_type, status, username, display_name, password_hash, password_kind, activated_at)
		VALUES ('staff', 'active', 'pengguna.lain', 'Pengguna Lain', 'x', 'user', now())
		RETURNING id`).Scan(&otherID); err != nil {
		t.Fatalf("buat pengguna lain: %v", err)
	}

	_, err = env.svc.Get(context.Background(), &auth.Principal{UserID: otherID}, asset.ID)
	if !apperr.Is(err, "NOT_FOUND") {
		t.Fatalf("Get pengguna lain error = %v, mau NOT_FOUND", err)
	}
}

func TestDelete_UnreferencedRemovesRowAndObject(t *testing.T) {
	env := newMediaEnv(t, false)
	asset, err := env.upload(PurposeContent, "foto.png", tinyPNG(t))
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	var key string
	if err := env.pool.QueryRow(context.Background(),
		`SELECT original_key FROM media_assets WHERE id = $1`, asset.ID,
	).Scan(&key); err != nil {
		t.Fatalf("baca original_key: %v", err)
	}

	if err := env.svc.Delete(context.Background(), env.principal(), asset.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	var deletedAt *time.Time
	if err := env.pool.QueryRow(context.Background(),
		`SELECT deleted_at FROM media_assets WHERE id = $1`, asset.ID,
	).Scan(&deletedAt); err != nil {
		t.Fatalf("baca deleted_at: %v", err)
	}
	if deletedAt == nil {
		t.Error("deleted_at nil setelah Delete")
	}
	if _, _, err := env.s3.GetObject(context.Background(), env.s3.PrivateBucket, key); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("objek setelah Delete error = %v, mau storage.ErrNotFound", err)
	}
	if _, err := env.svc.Get(context.Background(), env.principal(), asset.ID); !apperr.Is(err, "NOT_FOUND") {
		t.Errorf("Get setelah Delete error = %v, mau NOT_FOUND", err)
	}
}

func TestDelete_ReferencedReturnsConflict(t *testing.T) {
	env := newMediaEnv(t, false)
	asset, err := env.upload(PurposeAvatar, "avatar.png", tinyPNG(t))
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	if _, err := env.pool.Exec(context.Background(), `
		INSERT INTO profiles (user_id, avatar_media_id) VALUES ($1, $2)`,
		env.userID, asset.ID); err != nil {
		t.Fatalf("pasang avatar profil: %v", err)
	}

	err = env.svc.Delete(context.Background(), env.principal(), asset.ID)
	if !apperr.Is(err, "CONFLICT") {
		t.Fatalf("Delete error = %v, mau CONFLICT", err)
	}
}
