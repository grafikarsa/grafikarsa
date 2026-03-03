# MinIO Implementation Guide - Grafikarsa

## Overview

Grafikarsa menggunakan **MinIO** sebagai object storage untuk menyimpan file statis seperti avatar, banner, thumbnail portfolio, dan gambar dalam content block. Implementasi menggunakan **presigned URL** untuk upload langsung dari client ke MinIO tanpa melalui backend, meningkatkan performa dan mengurangi beban server.

---

## Mengapa MinIO?

1. **S3-Compatible**: API kompatibel dengan Amazon S3, mudah migrasi ke cloud jika diperlukan
2. **Self-Hosted**: Data tetap di server sendiri, kontrol penuh atas storage
3. **Presigned URL**: Client upload langsung ke MinIO, backend hanya generate URL
4. **Cost Effective**: Tidak ada biaya per-request seperti cloud storage
5. **High Performance**: Optimized untuk throughput tinggi

---

## Mengapa Presigned URL?

```
┌─────────────────────────────────────────────────────────────────┐
│                    TRADITIONAL UPLOAD                           │
│                                                                 │
│  Client ──[file]──> Backend ──[file]──> MinIO                  │
│                                                                 │
│  ❌ Backend jadi bottleneck                                     │
│  ❌ Memory usage tinggi untuk file besar                        │
│  ❌ Latency double (client→backend→minio)                       │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                    PRESIGNED URL UPLOAD                         │
│                                                                 │
│  Client ──[metadata]──> Backend ──[presigned URL]──> Client    │
│                                        │                        │
│  Client ──────────[file]───────────────┴──────────> MinIO      │
│                                                                 │
│  ✅ Backend hanya handle metadata                               │
│  ✅ Upload langsung ke MinIO                                    │
│  ✅ Scalable untuk file besar                                   │
└─────────────────────────────────────────────────────────────────┘
```

---

## Upload Flow

### Sequence Diagram

```
┌─────────┐          ┌─────────┐          ┌─────────┐
│ Client  │          │ Backend │          │  MinIO  │
└────┬────┘          └────┬────┘          └────┬────┘
     │                    │                    │
     │ 1. POST /uploads/presign               │
     │    {upload_type, filename, etc}        │
     │───────────────────>│                    │
     │                    │                    │
     │                    │ 2. Validate request│
     │                    │    - Check auth    │
     │                    │    - Check size    │
     │                    │    - Check type    │
     │                    │                    │
     │                    │ 3. Generate        │
     │                    │    presigned URL   │
     │                    │───────────────────>│
     │                    │                    │
     │                    │ 4. Store upload    │
     │                    │    metadata in DB  │
     │                    │                    │
     │ 5. Return:         │                    │
     │    - upload_id     │                    │
     │    - presigned_url │                    │
     │    - object_key    │                    │
     │    - headers       │                    │
     │<───────────────────│                    │
     │                    │                    │
     │ 6. PUT presigned_url                   │
     │    Headers: Content-Type               │
     │    Body: file binary                   │
     │────────────────────────────────────────>│
     │                    │                    │
     │ 7. 200 OK          │                    │
     │<────────────────────────────────────────│
     │                    │                    │
     │ 8. POST /uploads/confirm               │
     │    {upload_id, object_key}             │
     │───────────────────>│                    │
     │                    │                    │
     │                    │ 9. HEAD object    │
     │                    │    (verify exists) │
     │                    │───────────────────>│
     │                    │                    │
     │                    │ 10. Update DB      │
     │                    │     (avatar_url,   │
     │                    │      thumbnail_url)│
     │                    │                    │
     │ 11. Return:        │                    │
     │     - public URL   │                    │
     │<───────────────────│                    │
```

---

## API Endpoints

### 1. POST /uploads/presign

Request presigned URL untuk upload.

**Request:**
```json
{
  "upload_type": "avatar",
  "filename": "profile.jpg",
  "content_type": "image/jpeg",
  "file_size": 102400
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "upload_id": "upload-uuid-123",
    "presigned_url": "https://minio.grafikarsa.com/bucket/path?X-Amz-...",
    "object_key": "avatars/user-id/uuid.jpg",
    "expires_in": 900,
    "method": "PUT",
    "headers": {
      "Content-Type": "image/jpeg"
    }
  }
}
```

### 2. POST /uploads/confirm

Konfirmasi upload selesai dan update database.

**Request:**
```json
{
  "upload_id": "upload-uuid-123",
  "object_key": "avatars/user-id/uuid.jpg"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "type": "avatar",
    "url": "https://cdn.grafikarsa.com/avatars/user-id/uuid.jpg"
  }
}
```

### 3. DELETE /uploads/{object_key}

Hapus file dari MinIO.

### 4. GET /uploads/presign-view

Generate presigned URL untuk view file private (jika diperlukan).

---

## Upload Types & Constraints

| Type | Purpose | Max Size | Allowed MIME Types | Path Pattern |
|------|---------|----------|-------------------|--------------|
| `avatar` | User profile picture | 2 MB | image/jpeg, image/png, image/webp | `avatars/{user_id}/{uuid}.{ext}` |
| `banner` | User profile banner | 5 MB | image/jpeg, image/png, image/webp | `banners/{user_id}/{uuid}.{ext}` |
| `thumbnail` | Portfolio thumbnail | 5 MB | image/jpeg, image/png, image/webp | `thumbnails/{portfolio_id}/{uuid}.{ext}` |
| `portfolio_image` | Image in content block | 10 MB | image/jpeg, image/png, image/webp, image/gif | `portfolio-images/{portfolio_id}/{uuid}.{ext}` |

### Mengapa Batasan Ini?

- **Avatar 2MB**: Profile picture tidak perlu resolusi tinggi, 2MB cukup untuk 1000x1000px
- **Banner 5MB**: Banner lebih lebar, butuh resolusi lebih tinggi
- **Thumbnail 5MB**: Preview portfolio, perlu kualitas baik tapi tidak full-size
- **Portfolio Image 10MB**: Konten utama portfolio, bisa high-res, support GIF untuk animasi

---

## MinIO Bucket Structure

```
grafikarsa/
├── avatars/
│   └── {user_id}/
│       ├── abc123.jpg
│       └── def456.png
│
├── banners/
│   └── {user_id}/
│       └── xyz789.jpg
│
├── thumbnails/
│   └── {portfolio_id}/
│       └── thumb123.jpg
│
└── portfolio-images/
    └── {portfolio_id}/
        ├── img001.jpg
        ├── img002.png
        └── img003.gif
```

### Mengapa Struktur Ini?

1. **Organized by Type**: Mudah manage permission per folder
2. **User/Portfolio ID as Subfolder**: 
   - Isolasi file per user/portfolio
   - Mudah cleanup saat user/portfolio dihapus
   - Avoid filename collision
3. **UUID Filename**: 
   - Unique, tidak ada collision
   - Tidak expose nama file asli (security)
   - Cache-friendly (immutable URL)

---

## Backend Implementation

### Required Packages

```go
import (
    "github.com/minio/minio-go/v7"
    "github.com/minio/minio-go/v7/pkg/credentials"
)
```

### MinIO Client Initialization

```go
// internal/storage/minio.go

type MinIOClient struct {
    client     *minio.Client
    bucket     string
    publicURL  string
}

func NewMinIOClient(cfg *config.Config) (*MinIOClient, error) {
    client, err := minio.New(cfg.MinIO.Endpoint, &minio.Options{
        Creds:  credentials.NewStaticV4(cfg.MinIO.AccessKey, cfg.MinIO.SecretKey, ""),
        Secure: cfg.MinIO.UseSSL,
    })
    if err != nil {
        return nil, err
    }

    return &MinIOClient{
        client:    client,
        bucket:    cfg.MinIO.Bucket,
        publicURL: cfg.MinIO.PublicURL,
    }, nil
}
```

### Generate Presigned URL

```go
// internal/storage/minio.go

func (m *MinIOClient) GeneratePresignedPutURL(
    objectKey string,
    contentType string,
    expiry time.Duration,
) (string, error) {
    presignedURL, err := m.client.PresignedPutObject(
        context.Background(),
        m.bucket,
        objectKey,
        expiry,
    )
    if err != nil {
        return "", err
    }
    return presignedURL.String(), nil
}
```

### Verify Object Exists

```go
// internal/storage/minio.go

func (m *MinIOClient) ObjectExists(objectKey string) (bool, error) {
    _, err := m.client.StatObject(
        context.Background(),
        m.bucket,
        objectKey,
        minio.StatObjectOptions{},
    )
    if err != nil {
        errResponse := minio.ToErrorResponse(err)
        if errResponse.Code == "NoSuchKey" {
            return false, nil
        }
        return false, err
    }
    return true, nil
}
```

### Delete Object

```go
// internal/storage/minio.go

func (m *MinIOClient) DeleteObject(objectKey string) error {
    return m.client.RemoveObject(
        context.Background(),
        m.bucket,
        objectKey,
        minio.RemoveObjectOptions{},
    )
}
```

### Get Public URL

```go
// internal/storage/minio.go

func (m *MinIOClient) GetPublicURL(objectKey string) string {
    return fmt.Sprintf("%s/%s", m.publicURL, objectKey)
}
```

---

## Handler Implementation

### Upload Handler Structure

```go
// internal/handler/upload_handler.go

type UploadHandler struct {
    minioClient *storage.MinIOClient
    uploadRepo  *repository.UploadRepository
    userRepo    *repository.UserRepository
    portfolioRepo *repository.PortfolioRepository
}

func NewUploadHandler(
    minioClient *storage.MinIOClient,
    uploadRepo *repository.UploadRepository,
    userRepo *repository.UserRepository,
    portfolioRepo *repository.PortfolioRepository,
) *UploadHandler {
    return &UploadHandler{
        minioClient:   minioClient,
        uploadRepo:    uploadRepo,
        userRepo:      userRepo,
        portfolioRepo: portfolioRepo,
    }
}
```

### Presign Handler

```go
// internal/handler/upload_handler.go

func (h *UploadHandler) Presign(c *fiber.Ctx) error {
    userID := middleware.GetUserID(c)
    
    var req dto.PresignRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(400).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Invalid request"))
    }

    // Validate upload type & constraints
    constraints, ok := uploadConstraints[req.UploadType]
    if !ok {
        return c.Status(400).JSON(dto.ErrorResponse("INVALID_UPLOAD_TYPE", "Invalid upload type"))
    }

    if req.FileSize > constraints.MaxSize {
        return c.Status(400).JSON(dto.ErrorResponse("FILE_TOO_LARGE", "File exceeds max size"))
    }

    if !isAllowedContentType(req.ContentType, constraints.AllowedTypes) {
        return c.Status(400).JSON(dto.ErrorResponse("INVALID_CONTENT_TYPE", "Content type not allowed"))
    }

    // Generate object key
    objectKey := generateObjectKey(req.UploadType, *userID, req.PortfolioID, req.Filename)

    // Generate presigned URL
    presignedURL, err := h.minioClient.GeneratePresignedPutURL(objectKey, req.ContentType, 15*time.Minute)
    if err != nil {
        return c.Status(500).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Failed to generate presigned URL"))
    }

    // Store upload metadata
    upload := &domain.Upload{
        ID:          uuid.New(),
        UserID:      *userID,
        UploadType:  req.UploadType,
        ObjectKey:   objectKey,
        ContentType: req.ContentType,
        FileSize:    req.FileSize,
        PortfolioID: req.PortfolioID,
        BlockID:     req.BlockID,
        ExpiresAt:   time.Now().Add(15 * time.Minute),
    }
    h.uploadRepo.Create(upload)

    return c.JSON(dto.SuccessResponse(dto.PresignResponse{
        UploadID:     upload.ID.String(),
        PresignedURL: presignedURL,
        ObjectKey:    objectKey,
        ExpiresIn:    900,
        Method:       "PUT",
        Headers: map[string]string{
            "Content-Type": req.ContentType,
        },
    }, ""))
}
```

### Confirm Handler

```go
// internal/handler/upload_handler.go

func (h *UploadHandler) Confirm(c *fiber.Ctx) error {
    userID := middleware.GetUserID(c)

    var req dto.ConfirmUploadRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(400).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Invalid request"))
    }

    // Get upload metadata
    upload, err := h.uploadRepo.FindByID(req.UploadID)
    if err != nil {
        return c.Status(404).JSON(dto.ErrorResponse("UPLOAD_NOT_FOUND", "Upload not found"))
    }

    // Verify ownership
    if upload.UserID != *userID {
        return c.Status(403).JSON(dto.ErrorResponse("FORBIDDEN", "Not authorized"))
    }

    // Check if already confirmed
    if upload.ConfirmedAt != nil {
        return c.Status(400).JSON(dto.ErrorResponse("UPLOAD_ALREADY_CONFIRMED", "Already confirmed"))
    }

    // Verify object exists in MinIO
    exists, err := h.minioClient.ObjectExists(upload.ObjectKey)
    if err != nil || !exists {
        return c.Status(400).JSON(dto.ErrorResponse("OBJECT_NOT_FOUND", "File not found in storage"))
    }

    // Update database based on upload type
    publicURL := h.minioClient.GetPublicURL(upload.ObjectKey)
    
    switch upload.UploadType {
    case "avatar":
        h.userRepo.UpdateAvatarURL(*userID, publicURL)
    case "banner":
        h.userRepo.UpdateBannerURL(*userID, publicURL)
    case "thumbnail":
        h.portfolioRepo.UpdateThumbnailURL(upload.PortfolioID, publicURL)
    case "portfolio_image":
        // Return URL, client will update content block
    }

    // Mark upload as confirmed
    h.uploadRepo.MarkConfirmed(upload.ID)

    return c.JSON(dto.SuccessResponse(dto.ConfirmUploadResponse{
        Type:      upload.UploadType,
        URL:       publicURL,
        ObjectKey: upload.ObjectKey,
    }, "Upload confirmed"))
}
```

---

## Database Schema

### uploads table

```sql
CREATE TABLE uploads (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    upload_type VARCHAR(50) NOT NULL,
    object_key VARCHAR(500) NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    file_size BIGINT NOT NULL,
    portfolio_id UUID REFERENCES portfolios(id),
    block_id UUID REFERENCES content_blocks(id),
    expires_at TIMESTAMP NOT NULL,
    confirmed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_uploads_user_id ON uploads(user_id);
CREATE INDEX idx_uploads_expires_at ON uploads(expires_at);
```

### Cleanup Job

```sql
-- Hapus upload yang expired dan belum dikonfirmasi
DELETE FROM uploads 
WHERE confirmed_at IS NULL 
AND expires_at < NOW() - INTERVAL '1 hour';
```

---

## Frontend Implementation

### Upload Service

```typescript
// services/upload.ts

interface PresignResponse {
  upload_id: string;
  presigned_url: string;
  object_key: string;
  expires_in: number;
  method: string;
  headers: Record<string, string>;
}

interface ConfirmResponse {
  type: string;
  url: string;
  object_key: string;
}

export async function uploadFile(
  file: File,
  uploadType: 'avatar' | 'banner' | 'thumbnail' | 'portfolio_image',
  portfolioId?: string,
  blockId?: string
): Promise<string> {
  // 1. Get presigned URL
  const presignRes = await api.post<PresignResponse>('/uploads/presign', {
    upload_type: uploadType,
    filename: file.name,
    content_type: file.type,
    file_size: file.size,
    portfolio_id: portfolioId,
    block_id: blockId,
  });

  const { upload_id, presigned_url, object_key, headers } = presignRes.data.data;

  // 2. Upload to MinIO
  const uploadRes = await fetch(presigned_url, {
    method: 'PUT',
    headers: headers,
    body: file,
  });

  if (!uploadRes.ok) {
    throw new Error('Upload to storage failed');
  }

  // 3. Confirm upload
  const confirmRes = await api.post<ConfirmResponse>('/uploads/confirm', {
    upload_id,
    object_key,
  });

  return confirmRes.data.data.url;
}
```

### Usage Example - Avatar Upload

```typescript
// components/AvatarUpload.tsx

const handleAvatarChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
  const file = e.target.files?.[0];
  if (!file) return;

  // Validate client-side
  if (file.size > 2 * 1024 * 1024) {
    toast.error('File terlalu besar. Maksimal 2MB');
    return;
  }

  if (!['image/jpeg', 'image/png', 'image/webp'].includes(file.type)) {
    toast.error('Format file tidak didukung');
    return;
  }

  try {
    setUploading(true);
    const newAvatarUrl = await uploadFile(file, 'avatar');
    setAvatarUrl(newAvatarUrl);
    toast.success('Avatar berhasil diupload');
  } catch (error) {
    toast.error('Gagal upload avatar');
  } finally {
    setUploading(false);
  }
};
```

### Usage Example - Portfolio Image

```typescript
// components/PortfolioEditor.tsx

const handleImageBlockUpload = async (file: File, blockId: string) => {
  try {
    const imageUrl = await uploadFile(
      file,
      'portfolio_image',
      portfolioId,
      blockId
    );

    // Update block payload with new image URL
    updateBlock(blockId, {
      payload: {
        url: imageUrl,
        caption: '',
      },
    });
  } catch (error) {
    toast.error('Gagal upload gambar');
  }
};
```

---

## Environment Configuration

### .env

```bash
# MinIO Configuration
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=your-secret-key-here
MINIO_USE_SSL=false
MINIO_BUCKET=grafikarsa

# Public URL for accessing files
# Development: MinIO direct
MINIO_PUBLIC_URL=http://localhost:9000/grafikarsa

# Production: Through CDN/Nginx
# MINIO_PUBLIC_URL=https://cdn.grafikarsa.com
```

### Docker Compose

```yaml
services:
  minio:
    image: minio/minio:latest
    container_name: grafikarsa-minio
    command: server /data --console-address ":9001"
    environment:
      - MINIO_ROOT_USER=${MINIO_ACCESS_KEY}
      - MINIO_ROOT_PASSWORD=${MINIO_SECRET_KEY}
    volumes:
      - minio_data:/data
    ports:
      - "9000:9000"  # API
      - "9001:9001"  # Console
    healthcheck:
      test: ["CMD", "mc", "ready", "local"]
      interval: 10s
      timeout: 5s
      retries: 5

  minio-setup:
    image: minio/mc:latest
    depends_on:
      minio:
        condition: service_healthy
    entrypoint: >
      /bin/sh -c "
      mc alias set myminio http://minio:9000 ${MINIO_ACCESS_KEY} ${MINIO_SECRET_KEY};
      mc mb myminio/${MINIO_BUCKET} --ignore-existing;
      mc anonymous set download myminio/${MINIO_BUCKET}/avatars;
      mc anonymous set download myminio/${MINIO_BUCKET}/banners;
      mc anonymous set download myminio/${MINIO_BUCKET}/thumbnails;
      mc anonymous set download myminio/${MINIO_BUCKET}/portfolio-images;
      exit 0;
      "

volumes:
  minio_data:
```

---

## Nginx Configuration (Production)

```nginx
# Proxy untuk akses file publik
location /storage/ {
    proxy_pass http://minio:9000/grafikarsa/;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    
    # Cache static files
    proxy_cache_valid 200 7d;
    add_header Cache-Control "public, max-age=604800, immutable";
    
    # CORS untuk frontend
    add_header Access-Control-Allow-Origin "*";
}
```

---

## Security Considerations

### 1. Presigned URL Expiry
- URL expire dalam 15 menit
- Cukup untuk upload, tidak bisa disalahgunakan lama

### 2. Content Type Validation
- Validasi di backend sebelum generate presigned URL
- MinIO juga validasi Content-Type saat upload

### 3. File Size Validation
- Validasi di backend sebelum generate presigned URL
- Reject request jika melebihi limit

### 4. Ownership Verification
- Confirm endpoint verifikasi user adalah pemilik upload
- Portfolio image verifikasi user adalah owner portfolio

### 5. Public Access Control
- Hanya folder tertentu yang public (avatars, banners, thumbnails, portfolio-images)
- Folder lain private by default

---

## Error Handling

| Error Code | HTTP Status | Cause | Solution |
|------------|-------------|-------|----------|
| `FILE_TOO_LARGE` | 400 | File melebihi batas | Compress atau resize file |
| `INVALID_CONTENT_TYPE` | 400 | MIME type tidak diizinkan | Gunakan format yang didukung |
| `INVALID_UPLOAD_TYPE` | 400 | Upload type tidak valid | Gunakan: avatar, banner, thumbnail, portfolio_image |
| `UPLOAD_NOT_FOUND` | 404 | Upload ID tidak ditemukan | Request presign ulang |
| `UPLOAD_ALREADY_CONFIRMED` | 400 | Upload sudah dikonfirmasi | Tidak perlu confirm lagi |
| `OBJECT_NOT_FOUND` | 400 | File tidak ada di MinIO | Upload ulang ke presigned URL |
| `FORBIDDEN` | 403 | Bukan pemilik upload | Login dengan akun yang benar |

---

## Cleanup Strategy

### 1. Expired Uploads
```go
// Jalankan setiap jam via cron
func CleanupExpiredUploads() {
    // Hapus record upload yang expired dan belum dikonfirmasi
    uploadRepo.DeleteExpired()
}
```

### 2. Orphaned Files
```go
// Jalankan daily via cron
func CleanupOrphanedFiles() {
    // List semua file di MinIO
    // Bandingkan dengan database
    // Hapus file yang tidak ada di database
}
```

### 3. User/Portfolio Deletion
```go
// Saat user dihapus
func DeleteUserFiles(userID uuid.UUID) {
    minioClient.DeletePrefix(fmt.Sprintf("avatars/%s/", userID))
    minioClient.DeletePrefix(fmt.Sprintf("banners/%s/", userID))
}

// Saat portfolio dihapus
func DeletePortfolioFiles(portfolioID uuid.UUID) {
    minioClient.DeletePrefix(fmt.Sprintf("thumbnails/%s/", portfolioID))
    minioClient.DeletePrefix(fmt.Sprintf("portfolio-images/%s/", portfolioID))
}
```

---

## Testing

### Manual Testing dengan cURL

```bash
# 1. Get presigned URL
curl -X POST http://localhost:8080/api/v1/uploads/presign \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "upload_type": "avatar",
    "filename": "test.jpg",
    "content_type": "image/jpeg",
    "file_size": 102400
  }'

# 2. Upload to MinIO
curl -X PUT "<presigned_url>" \
  -H "Content-Type: image/jpeg" \
  --data-binary @test.jpg

# 3. Confirm upload
curl -X POST http://localhost:8080/api/v1/uploads/confirm \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "upload_id": "<upload_id>",
    "object_key": "<object_key>"
  }'
```

### MinIO Console

Akses MinIO Console di `http://localhost:9001` untuk:
- Melihat file yang diupload
- Manage bucket dan permission
- Monitor storage usage
