# REST API Design Standards & Best Practices

Dokumen ini berisi panduan lengkap untuk mendesain REST API yang konsisten, scalable, dan mudah dipahami.

---

## 1. URL & Endpoint Convention

### Aturan Dasar
- Gunakan **lowercase** untuk semua URL
- Gunakan **kebab-case** (tanda hubung) untuk multi-word resources
- Gunakan **noun (kata benda)** untuk resource, bukan verb
- Gunakan **plural** untuk collection resources

### Contoh

| ✅ Benar | ❌ Salah |
|----------|----------|
| `/users` | `/getUsers` |
| `/order-items` | `/orderItems` |
| `/products/123` | `/product/123` |
| `/users/5/orders` | `/getUserOrders` |

### Struktur Hierarki
```
GET    /users                    # List semua users
GET    /users/{id}               # Get single user
GET    /users/{id}/orders        # Get orders milik user
GET    /users/{id}/orders/{id}   # Get specific order milik user
```

### Versioning
Gunakan prefix version di URL:
```
/api/v1/users
/api/v2/users
```

---

## 2. HTTP Methods

| Method | Fungsi | Idempotent | Safe |
|--------|--------|------------|------|
| `GET` | Mengambil resource | ✅ | ✅ |
| `POST` | Membuat resource baru | ❌ | ❌ |
| `PUT` | Update seluruh resource | ✅ | ❌ |
| `PATCH` | Update sebagian resource | ❌ | ❌ |
| `DELETE` | Menghapus resource | ✅ | ❌ |

### Penggunaan

```http
GET    /users          # List users
POST   /users          # Create user
GET    /users/123      # Get user by ID
PUT    /users/123      # Replace entire user
PATCH  /users/123      # Update partial user
DELETE /users/123      # Delete user
```

---

## 3. HTTP Status Codes

### Success (2xx)
| Code | Nama | Penggunaan |
|------|------|------------|
| `200` | OK | Request berhasil (GET, PUT, PATCH) |
| `201` | Created | Resource berhasil dibuat (POST) |
| `204` | No Content | Berhasil tanpa response body (DELETE) |

### Client Error (4xx)
| Code | Nama | Penggunaan |
|------|------|------------|
| `400` | Bad Request | Request tidak valid / malformed |
| `401` | Unauthorized | Authentication diperlukan |
| `403` | Forbidden | Tidak punya akses |
| `404` | Not Found | Resource tidak ditemukan |
| `409` | Conflict | Konflik data (duplicate, dll) |
| `422` | Unprocessable Entity | Validasi gagal |
| `429` | Too Many Requests | Rate limit exceeded |

### Server Error (5xx)
| Code | Nama | Penggunaan |
|------|------|------------|
| `500` | Internal Server Error | Error tidak terduga di server |
| `502` | Bad Gateway | Upstream server error |
| `503` | Service Unavailable | Server sedang maintenance |

---

## 4. Request Format

### Headers
```http
Content-Type: application/json
Accept: application/json
Authorization: Bearer <token>
X-Request-ID: <uuid>
```

### Query Parameters
Gunakan untuk filtering, sorting, dan pagination:
```
GET /users?status=active&sort=-created_at&page=1&limit=20
```

| Parameter | Fungsi | Contoh |
|-----------|--------|--------|
| `filter` | Filter data | `?status=active` |
| `sort` | Sorting (- untuk desc) | `?sort=-created_at` |
| `page` | Halaman | `?page=2` |
| `limit` | Jumlah per halaman | `?limit=20` |
| `fields` | Select fields | `?fields=id,name,email` |
| `search` | Full-text search | `?search=john` |

### Request Body (POST/PUT/PATCH)
```json
{
  "name": "John Doe",
  "email": "john@example.com",
  "role": "admin"
}
```

---

## 5. Response Format

### Struktur Standar - Single Resource
```json
{
  "data": {
    "id": 123,
    "name": "John Doe",
    "email": "john@example.com",
    "created_at": "2025-12-08T10:30:00Z",
    "updated_at": "2025-12-08T10:30:00Z"
  }
}
```

### Struktur Standar - Collection
```json
{
  "data": [
    { "id": 1, "name": "John" },
    { "id": 2, "name": "Jane" }
  ],
  "meta": {
    "current_page": 1,
    "per_page": 20,
    "total_pages": 5,
    "total_count": 100
  }
}
```

### Struktur Error
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validation failed",
    "details": [
      {
        "field": "email",
        "message": "Email format is invalid"
      },
      {
        "field": "password",
        "message": "Password must be at least 8 characters"
      }
    ]
  }
}
```

---

## 6. Naming Convention

### Field Names
- Gunakan **snake_case** untuk JSON fields
- Konsisten di seluruh API

```json
{
  "user_id": 123,
  "first_name": "John",
  "last_name": "Doe",
  "created_at": "2025-12-08T10:30:00Z"
}
```

### Date & Time
- Gunakan format **ISO 8601** dengan timezone UTC
- Format: `YYYY-MM-DDTHH:mm:ssZ`

```json
{
  "created_at": "2025-12-08T10:30:00Z",
  "expires_at": "2025-12-15T23:59:59Z"
}
```

### Boolean Fields
Gunakan prefix yang jelas:
```json
{
  "is_active": true,
  "has_verified_email": false,
  "can_edit": true
}
```

### ID Fields
```json
{
  "id": 123,
  "user_id": 456,
  "order_id": 789
}
```

---

## 7. Pagination

### Offset-based Pagination
```
GET /users?page=2&limit=20
```

Response:
```json
{
  "data": [...],
  "meta": {
    "current_page": 2,
    "per_page": 20,
    "total_pages": 10,
    "total_count": 200
  },
  "links": {
    "first": "/users?page=1&limit=20",
    "prev": "/users?page=1&limit=20",
    "next": "/users?page=3&limit=20",
    "last": "/users?page=10&limit=20"
  }
}
```

### Cursor-based Pagination (untuk large datasets)
```
GET /users?cursor=eyJpZCI6MTAwfQ&limit=20
```

Response:
```json
{
  "data": [...],
  "meta": {
    "has_more": true,
    "next_cursor": "eyJpZCI6MTIwfQ"
  }
}
```

---

## 8. Filtering & Sorting

### Filtering
```
GET /products?category=electronics&price_min=100&price_max=500
GET /users?status=active,pending
GET /orders?created_after=2025-01-01
```

### Sorting
```
GET /users?sort=name              # ASC by name
GET /users?sort=-created_at       # DESC by created_at
GET /users?sort=-created_at,name  # Multiple sort
```

---

## 9. Authentication & Authorization

### Bearer Token (JWT)
```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

### API Key
```http
X-API-Key: your-api-key-here
```

### Response untuk Auth Error
```json
// 401 Unauthorized
{
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Invalid or expired token"
  }
}

// 403 Forbidden
{
  "error": {
    "code": "FORBIDDEN",
    "message": "You don't have permission to access this resource"
  }
}
```

---

## 10. Rate Limiting

### Response Headers
```http
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1702036800
```

### Response saat Limit Exceeded (429)
```json
{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Too many requests",
    "retry_after": 60
  }
}
```

---

## 11. HATEOAS (Hypermedia)

Sertakan links untuk navigasi:
```json
{
  "data": {
    "id": 123,
    "name": "John Doe"
  },
  "links": {
    "self": "/users/123",
    "orders": "/users/123/orders",
    "profile": "/users/123/profile"
  }
}
```

---

## 12. Bulk Operations

### Batch Create
```http
POST /users/batch
```
```json
{
  "items": [
    { "name": "John", "email": "john@example.com" },
    { "name": "Jane", "email": "jane@example.com" }
  ]
}
```

### Batch Update
```http
PATCH /users/batch
```
```json
{
  "items": [
    { "id": 1, "status": "active" },
    { "id": 2, "status": "inactive" }
  ]
}
```

### Batch Delete
```http
DELETE /users/batch
```
```json
{
  "ids": [1, 2, 3, 4, 5]
}
```

---

## 13. File Upload

### Single File
```http
POST /files
Content-Type: multipart/form-data
```

### Response
```json
{
  "data": {
    "id": "abc123",
    "filename": "document.pdf",
    "size": 1024000,
    "mime_type": "application/pdf",
    "url": "https://cdn.example.com/files/abc123.pdf"
  }
}
```

---

## 14. Caching

### Response Headers
```http
Cache-Control: public, max-age=3600
ETag: "abc123"
Last-Modified: Sun, 08 Dec 2025 10:30:00 GMT
```

### Conditional Requests
```http
If-None-Match: "abc123"
If-Modified-Since: Sun, 08 Dec 2025 10:30:00 GMT
```

Response `304 Not Modified` jika tidak ada perubahan.

---

## 15. Security Best Practices

1. **Selalu gunakan HTTPS**
2. **Validasi semua input** di server-side
3. **Sanitize output** untuk mencegah XSS
4. **Implementasi rate limiting**
5. **Gunakan parameterized queries** untuk mencegah SQL injection
6. **Jangan expose sensitive data** di response (password, internal IDs)
7. **Log semua request** untuk audit trail
8. **Implementasi CORS** dengan benar
9. **Gunakan security headers**:
   ```http
   X-Content-Type-Options: nosniff
   X-Frame-Options: DENY
   X-XSS-Protection: 1; mode=block
   ```

---

## 16. API Documentation

Setiap endpoint harus didokumentasikan dengan:
- URL dan HTTP method
- Deskripsi singkat
- Request parameters (query, path, body)
- Request/response examples
- Possible error codes
- Authentication requirements

Gunakan tools seperti:
- OpenAPI/Swagger
- Postman Collections
- API Blueprint

---

## Quick Reference

```
URL:        lowercase, kebab-case, plural nouns
Fields:     snake_case
Dates:      ISO 8601 (2025-12-08T10:30:00Z)
Success:    200, 201, 204
Errors:     400, 401, 403, 404, 422, 500
Auth:       Bearer token / API key
Pagination: page, limit, cursor
Sorting:    sort=-field (- for DESC)
```
