# Feedback Attachments Feature

## Overview

Fitur attachment pada sistem feedback memungkinkan user untuk melampirkan file (screenshot, dokumen) saat mengirim feedback. Ini sangat berguna untuk:
- Bug report dengan screenshot error
- Saran fitur dengan mockup/wireframe
- Dokumentasi pendukung lainnya

## Technical Implementation

### Database Schema

**Migration:** `db/migrations/add_feedback_attachments.sql`

```sql
ALTER TABLE feedback 
ADD COLUMN attachment_url TEXT;

CREATE INDEX idx_feedback_has_attachment 
ON feedback(attachment_url) 
WHERE attachment_url IS NOT NULL;
```

### Backend (Go)

#### Domain Model
```go
type Feedback struct {
    ID            uuid.UUID
    UserID        *uuid.UUID
    Kategori      FeedbackKategori
    Pesan         string
    AttachmentURL *string  // NEW: URL file attachment
    Status        FeedbackStatus
    AdminNotes    *string
    ResolvedBy    *uuid.UUID
    ResolvedAt    *time.Time
    CreatedAt     time.Time
    UpdatedAt     time.Time
}
```

#### DTO
```go
type CreateFeedbackRequest struct {
    Kategori      FeedbackKategori `json:"kategori"`
    Pesan         string           `json:"pesan"`
    AttachmentURL *string          `json:"attachment_url,omitempty"`
}

type FeedbackResponse struct {
    // ... other fields
    AttachmentURL *string `json:"attachment_url,omitempty"`
}
```

#### Upload Handler

**Upload Type:** `feedback_attachment`

**Limits:**
- Max file size: 5MB
- Allowed types:
  - Images: JPEG, PNG, WebP, GIF
  - Documents: PDF, DOC, DOCX

**Storage Path:** `feedback-attachments/{fileID}.{ext}`

**Flow:**
1. User selects file
2. Frontend calls `POST /upload/presign` with `upload_type: "feedback_attachment"`
3. Backend generates presigned URL for MinIO
4. Frontend uploads file directly to MinIO
5. Frontend calls `POST /upload/confirm`
6. Backend returns public URL
7. Frontend includes URL in feedback submission

### Frontend (Next.js + React)

#### API Client

**Type Definition:**
```typescript
export interface CreateFeedbackRequest {
    kategori: FeedbackKategori;
    pesan: string;
    attachment_url?: string;
}

export interface Feedback {
    // ... other fields
    attachment_url?: string;
}
```

**Upload API:**
```typescript
uploadsApi.uploadFile(file, 'feedback_attachment')
```

#### Components

**FeedbackForm** (`apps/web/components/shared/feedback-form.tsx`)

Features:
- File input with drag & drop support
- File type validation (images, PDF, DOC)
- File size validation (max 5MB)
- Upload progress indicator
- Preview uploaded file
- Remove attachment button
- Automatic upload on file select

UI Elements:
- Paperclip icon button to attach file
- File preview card with icon (image/document)
- Remove button (X icon)
- Helper text: "Gambar, PDF, atau DOC (maks. 5MB)"

**Admin Feedback Page** (`apps/web/app/admin/(dashboard)/feedback/page.tsx`)

Attachment Display:
- **Table View:** Paperclip icon indicator next to message
- **Kanban View:** "Ada lampiran" badge below message
- **Detail Dialog:** 
  - Image attachments: Full preview with lightbox
  - Document attachments: Download link with file icon

## User Flow

### User Submitting Feedback with Attachment

1. User clicks feedback button (floating or settings)
2. Dialog opens with FeedbackForm
3. User selects kategori (bug/saran/lainnya)
4. User writes message (10-2000 chars)
5. User clicks "Lampirkan File" button
6. File picker opens
7. User selects file (screenshot, PDF, etc.)
8. File validates (type & size)
9. File uploads to MinIO automatically
10. Upload progress shown
11. File preview appears with remove option
12. User clicks "Kirim"
13. Feedback submitted with attachment URL
14. Success toast shown

### Admin Viewing Feedback with Attachment

1. Admin navigates to /admin/feedback
2. Feedbacks with attachments show paperclip icon
3. Admin clicks feedback to view detail
4. Detail dialog opens
5. If image: Full preview displayed, clickable to open in new tab
6. If document: Download link with file icon
7. Admin can view attachment while updating status/notes

## File Types & Use Cases

### Images (JPEG, PNG, WebP, GIF)
- **Bug Reports:** Screenshot of error messages, broken UI
- **Feature Suggestions:** Mockup, wireframe, design inspiration
- **General Feedback:** Visual examples

### Documents (PDF, DOC, DOCX)
- **Bug Reports:** Error logs, stack traces
- **Feature Suggestions:** Detailed specifications, requirements doc
- **General Feedback:** Supporting documentation

## Security & Validation

### Frontend Validation
- File type check using MIME type
- File size check (5MB limit)
- User-friendly error messages

### Backend Validation
- Upload type whitelist
- Content-Type validation
- File size limit enforcement
- MinIO object existence verification

### Storage Security
- Files stored in MinIO with public read access
- Unique file IDs prevent collisions
- No user-controlled paths (prevents directory traversal)
- Presigned URLs expire after 15 minutes

## Performance Considerations

### Upload Optimization
- Direct upload to MinIO (no backend proxy)
- Presigned URLs reduce backend load
- Async upload (non-blocking UI)

### Display Optimization
- Image lazy loading
- Thumbnail generation (future enhancement)
- CDN integration (future enhancement)

## Future Enhancements

1. **Multiple Attachments**
   - Allow 2-3 files per feedback
   - Gallery view for multiple images

2. **Image Compression**
   - Auto-compress large images
   - Maintain quality while reducing size

3. **Thumbnail Generation**
   - Generate thumbnails for faster loading
   - Full-size on click

4. **Drag & Drop**
   - Drag files directly into textarea
   - Visual drop zone

5. **Attachment Analytics**
   - Track attachment usage
   - Most common file types
   - Average file sizes

6. **Admin Attachment Management**
   - Bulk delete orphaned files
   - Storage usage dashboard
   - File type statistics

## Testing Checklist

### User Flow
- [ ] Upload image (JPEG, PNG, WebP, GIF)
- [ ] Upload document (PDF, DOC, DOCX)
- [ ] File size validation (>5MB rejected)
- [ ] File type validation (invalid types rejected)
- [ ] Remove attachment before submit
- [ ] Submit feedback with attachment
- [ ] Submit feedback without attachment

### Admin Flow
- [ ] View feedback with image attachment
- [ ] View feedback with document attachment
- [ ] Open image in new tab
- [ ] Download document
- [ ] Paperclip indicator in table view
- [ ] Attachment badge in kanban view

### Edge Cases
- [ ] Upload fails (network error)
- [ ] Invalid file type
- [ ] File too large
- [ ] Slow upload (progress indicator)
- [ ] Cancel upload mid-way
- [ ] Submit without waiting for upload

## API Endpoints

### Upload Flow
```
POST /api/v1/upload/presign
Body: {
  "upload_type": "feedback_attachment",
  "filename": "screenshot.png",
  "content_type": "image/png",
  "file_size": 1234567
}
Response: {
  "upload_id": "uuid",
  "presigned_url": "https://minio.../feedback-attachments/uuid.png",
  "object_key": "feedback-attachments/uuid.png",
  "expires_in": 900
}

PUT {presigned_url}
Body: <file binary>

POST /api/v1/upload/confirm
Body: {
  "upload_id": "uuid",
  "object_key": "feedback-attachments/uuid.png"
}
Response: {
  "type": "feedback_attachment",
  "url": "https://cdn.../feedback-attachments/uuid.png",
  "object_key": "feedback-attachments/uuid.png"
}
```

### Feedback Submission
```
POST /api/v1/feedback
Body: {
  "kategori": "bug",
  "pesan": "Error saat upload portfolio",
  "attachment_url": "https://cdn.../feedback-attachments/uuid.png"
}
```

## Configuration

### Environment Variables
```env
# MinIO Configuration (already configured)
MINIO_ENDPOINT=minio:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_BUCKET=grafikarsa
MINIO_PUBLIC_URL=http://localhost:9000/grafikarsa
```

### Upload Limits (Backend)
```go
var uploadLimits = map[string]int64{
    "feedback_attachment": 5 * 1024 * 1024, // 5MB
}

var allowedTypes = map[string][]string{
    "feedback_attachment": {
        "image/jpeg", "image/png", "image/webp", "image/gif",
        "application/pdf",
        "application/msword",
        "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
    },
}
```

## Monitoring & Maintenance

### Metrics to Track
- Total attachments uploaded
- Average file size
- Most common file types
- Upload success/failure rate
- Storage usage

### Cleanup Tasks
- Delete attachments when feedback is deleted (future)
- Archive old attachments (>1 year)
- Compress large images automatically

## Troubleshooting

### Upload Fails
- Check MinIO connection
- Verify presigned URL not expired
- Check file size limits
- Validate CORS settings

### Attachment Not Displaying
- Verify public URL is accessible
- Check MinIO bucket policy
- Validate attachment_url in database

### Performance Issues
- Enable CDN for static files
- Implement image compression
- Use thumbnail generation
