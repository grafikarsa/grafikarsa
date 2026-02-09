# Grafikarsa API Documentation

Base URL: `https://grafikarsa.com/api/v1`

## Overview

API untuk platform Grafikarsa - Katalog Portofolio & Social Network Warga SMKN 4 Malang.

### Authentication

Semua endpoint yang memerlukan autentikasi menggunakan JWT Bearer Token:
```
Authorization: Bearer <access_token>
```

### Response Format

Semua response menggunakan format JSON dengan struktur konsisten:

**Success Response:**
```json
{
  "data": { ... },
  "meta": { ... }
}
```

**Error Response:**
```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable message",
    "details": [ ... ]
  }
}
```

---

## Table of Contents

1. [Authentication](#1-authentication)
2. [Users](#2-users)
3. [Profiles](#3-profiles)
4. [Portfolios](#4-portfolios)
5. [Content Blocks](#5-content-blocks)
6. [Tags](#6-tags)
7. [File Upload (MinIO Presigned URL)](#7-file-upload-minio-presigned-url)
8. [Social (Follow)](#8-social-follow)
9. [Likes](#9-likes)
10. [Search](#10-search)
11. [Feed](#11-feed)
12. [Admin - Jurusan](#12-admin---jurusan)
13. [Admin - Tahun Ajaran](#13-admin---tahun-ajaran)
14. [Admin - Kelas](#14-admin---kelas)
15. [Admin - Users](#15-admin---users)
16. [Admin - Tags](#16-admin---tags)
17. [Admin - Portfolios](#17-admin---portfolios)
18. [Admin - Moderasi](#18-admin---moderasi)
19. [Public - Jurusan & Kelas](#19-public---jurusan--kelas)

---

## 1. Authentication

### POST /auth/login

Login user dan dapatkan access token.

**Authentication:** None

**Request Body:**
```json
{
  "username": "john_doe",
  "password": "securepassword123"
}
```

**Success Response (200):**
```json
{
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "token_type": "Bearer",
    "expires_in": 900,
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "username": "john_doe",
      "name": "John Doe",
      "role": "student",
      "avatar_url": "https://cdn.grafikarsa.com/avatars/john.jpg"
    }
  }
}
```

**Response Headers:**
```
Set-Cookie: refresh_token=abc123...; HttpOnly; Secure; SameSite=Strict; Path=/api/v1/auth; Max-Age=604800
```

**Error Responses:**

`401 Unauthorized` - Kredensial salah:
```json
{
  "error": {
    "code": "INVALID_CREDENTIALS",
    "message": "Username atau password salah"
  }
}
```

`403 Forbidden` - Akun nonaktif:
```json
{
  "error": {
    "code": "ACCOUNT_DISABLED",
    "message": "Akun Anda telah dinonaktifkan. Hubungi admin."
  }
}
```

---

### POST /auth/refresh

Refresh access token menggunakan refresh token dari cookie.

**Authentication:** None (menggunakan HttpOnly cookie)

**Request:** Cookie `refresh_token` dikirim otomatis oleh browser.

**Success Response (200):**
```json
{
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "token_type": "Bearer",
    "expires_in": 900
  }
}
```

**Error Responses:**

`401 Unauthorized` - Token expired/invalid:
```json
{
  "error": {
    "code": "TOKEN_EXPIRED",
    "message": "Refresh token telah expired. Silakan login ulang."
  }
}
```

`401 Unauthorized` - Token reuse detected:
```json
{
  "error": {
    "code": "TOKEN_REUSE_DETECTED",
    "message": "Aktivitas mencurigakan terdeteksi. Semua sesi telah diakhiri."
  }
}
```

---

### POST /auth/logout

Logout dari sesi saat ini.

**Authentication:** Required

**Success Response (200):**
```json
{
  "message": "Berhasil logout"
}
```

**Response Headers:**
```
Set-Cookie: refresh_token=; HttpOnly; Secure; SameSite=Strict; Path=/api/v1/auth; Max-Age=0
```

---

### POST /auth/logout-all

Logout dari semua perangkat/sesi.

**Authentication:** Required

**Success Response (200):**
```json
{
  "message": "Berhasil logout dari semua perangkat",
  "data": {
    "sessions_terminated": 3
  }
}
```

---

### GET /auth/sessions

Lihat semua sesi aktif user.

**Authentication:** Required

**Success Response (200):**
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440001",
      "user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64)...",
      "ip_address": "192.168.1.1",
      "created_at": "2026-02-09T10:00:00Z",
      "last_used_at": "2026-02-09T14:30:00Z",
      "is_current": true
    }
  ]
}
```

---

### DELETE /auth/sessions/{session_id}

Hapus/revoke sesi tertentu.

**Authentication:** Required

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| session_id | UUID | ID sesi yang akan dihapus |

**Success Response (200):**
```json
{
  "message": "Sesi berhasil dihapus"
}
```

**Error Response:**

`404 Not Found`:
```json
{
  "error": {
    "code": "SESSION_NOT_FOUND",
    "message": "Sesi tidak ditemukan"
  }
}
```

---

## 2. Users

### GET /users

Daftar semua user (publik). Untuk halaman "Siswa & Alumni".

**Authentication:** Optional

**Query Parameters:**
| Parameter | Type | Description | Example |
|-----------|------|-------------|---------|
| search | string | Cari berdasarkan nama, username, bio | `?search=john` |
| major_id | UUID | Filter berdasarkan jurusan | `?major_id=xxx` |
| class_id | UUID | Filter berdasarkan kelas | `?class_id=xxx` |
| role | string | Filter berdasarkan role | `?role=student` |
| page | integer | Halaman (default: 1) | `?page=2` |
| limit | integer | Jumlah per halaman (default: 20, max: 50) | `?limit=20` |

**Success Response (200):**
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "username": "john_doe",
      "name": "John Doe",
      "avatar_url": "https://cdn.grafikarsa.com/avatars/john.jpg",
      "role": "student",
      "class": {
        "id": "660e8400-e29b-41d4-a716-446655440000",
        "name": "XII-RPL-A"
      },
      "major": {
        "id": "770e8400-e29b-41d4-a716-446655440000",
        "name": "Rekayasa Perangkat Lunak"
      }
    }
  ],
  "meta": {
    "current_page": 1,
    "per_page": 20,
    "total_pages": 5,
    "total_count": 100
  }
}
```

---

### GET /users/{username}

Detail profil user berdasarkan username.

**Authentication:** Optional

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| username | string | Username user |

**Success Response (200):**
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "john_doe",
    "name": "John Doe",
    "bio": "Siswa RPL yang suka coding dan desain",
    "avatar_url": "https://cdn.grafikarsa.com/avatars/john.jpg",
    "banner_url": "https://cdn.grafikarsa.com/banners/john.jpg",
    "role": "student",
    "status": "active",
    "entry_year": 2023,
    "graduation_year": null,
    "class": {
      "id": "660e8400-e29b-41d4-a716-446655440000",
      "name": "XII-RPL-A"
    },
    "major": {
      "id": "770e8400-e29b-41d4-a716-446655440000",
      "name": "Rekayasa Perangkat Lunak"
    },
    "class_history": [
      {
        "class_name": "X-RPL-A",
        "academic_year": 2023,
        "assigned_at": "2023-07-15T00:00:00Z"
      },
      {
        "class_name": "XI-RPL-A",
        "academic_year": 2024,
        "assigned_at": "2024-07-01T00:00:00Z"
      }
    ],
    "social_links": {
      "github": "https://github.com/johndoe",
      "instagram": "https://instagram.com/johndoe"
    },
    "follower_count": 150,
    "following_count": 75,
    "portfolio_count": 12,
    "is_following": false,
    "created_at": "2023-07-15T08:00:00Z"
  }
}
```

**Error Response:**

`404 Not Found`:
```json
{
  "error": {
    "code": "USER_NOT_FOUND",
    "message": "User tidak ditemukan"
  }
}
```

---

### GET /users/{username}/followers

Daftar follower user.

**Authentication:** Optional

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| search | string | Cari berdasarkan nama/username |
| page | integer | Halaman |
| limit | integer | Jumlah per halaman |

**Success Response (200):**
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440001",
      "username": "jane_doe",
      "name": "Jane Doe",
      "avatar_url": "https://cdn.grafikarsa.com/avatars/jane.jpg",
      "role": "student",
      "class_name": "XI-MM-B",
      "is_following": true,
      "followed_at": "2026-01-01T10:00:00Z"
    }
  ],
  "meta": {
    "current_page": 1,
    "per_page": 20,
    "total_pages": 8,
    "total_count": 150
  }
}
```

---

### GET /users/{username}/following

Daftar user yang di-follow.

**Authentication:** Optional

**Query Parameters:** Sama dengan `/followers`

**Success Response (200):** Sama dengan `/followers`

---

## 3. Profiles

### GET /me

Profil user yang sedang login.

**Authentication:** Required

**Success Response (200):**
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "john_doe",
    "email": "john@example.com",
    "name": "John Doe",
    "bio": "Siswa RPL yang suka coding dan desain",
    "avatar_url": "https://cdn.grafikarsa.com/avatars/john.jpg",
    "banner_url": "https://cdn.grafikarsa.com/banners/john.jpg",
    "role": "student",
    "status": "active",
    "nisn": "0098115881",
    "nis": "25491/02000.0411",
    "entry_year": 2023,
    "graduation_year": null,
    "class": {
      "id": "660e8400-e29b-41d4-a716-446655440000",
      "name": "XII-RPL-A"
    },
    "major": {
      "id": "770e8400-e29b-41d4-a716-446655440000",
      "name": "Rekayasa Perangkat Lunak"
    },
    "social_links": {
      "github": "https://github.com/johndoe"
    },
    "follower_count": 150,
    "following_count": 75,
    "created_at": "2023-07-15T08:00:00Z"
  }
}
```

---

### PATCH /me

Update profil user yang sedang login.

**Authentication:** Required

**Request Body:**
```json
{
  "name": "John Doe Updated",
  "username": "john_doe_new",
  "bio": "Updated bio",
  "email": "newemail@example.com"
}
```

**Success Response (200):**
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "john_doe_new",
    "email": "newemail@example.com",
    "name": "John Doe Updated",
    "bio": "Updated bio"
  },
  "message": "Profil berhasil diperbarui"
}
```

**Error Responses:**

`409 Conflict` - Username sudah dipakai:
```json
{
  "error": {
    "code": "USERNAME_TAKEN",
    "message": "Username sudah digunakan"
  }
}
```

`422 Unprocessable Entity` - Validasi gagal:
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validasi gagal",
    "details": [
      {
        "field": "username",
        "message": "Username minimal 3 karakter"
      },
      {
        "field": "email",
        "message": "Format email tidak valid"
      }
    ]
  }
}
```

---

### PATCH /me/password

Ubah password.

**Authentication:** Required

**Request Body:**
```json
{
  "current_password": "oldpassword123",
  "new_password": "newpassword456",
  "new_password_confirmation": "newpassword456"
}
```

**Success Response (200):**
```json
{
  "message": "Password berhasil diubah"
}
```

**Error Responses:**

`400 Bad Request` - Password lama salah:
```json
{
  "error": {
    "code": "INVALID_PASSWORD",
    "message": "Password lama tidak sesuai"
  }
}
```

`422 Unprocessable Entity`:
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validasi gagal",
    "details": [
      {
        "field": "new_password",
        "message": "Password minimal 8 karakter"
      }
    ]
  }
}
```

---

### PUT /me/social-links

Update semua social links sekaligus.

**Authentication:** Required

**Request Body:**
```json
{
  "social_links": {
    "github": "https://github.com/johndoe",
    "instagram": "https://instagram.com/johndoe",
    "linkedin": "https://linkedin.com/in/johndoe"
  }
}
```

**Valid Platforms:**
`facebook`, `instagram`, `github`, `linkedin`, `twitter`, `website`, `tiktok`, `youtube`, `behance`, `dribbble`, `threads`, `bluesky`, `medium`, `gitlab`

**Success Response (200):**
```json
{
  "data": {
    "social_links": {
      "github": "https://github.com/johndoe",
      "instagram": "https://instagram.com/johndoe",
      "linkedin": "https://linkedin.com/in/johndoe"
    }
  },
  "message": "Social links berhasil diperbarui"
}
```

---

### GET /me/check-username

Cek ketersediaan username.

**Authentication:** Required

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| username | string | Username yang ingin dicek |

**Success Response (200):**
```json
{
  "data": {
    "username": "new_username",
    "available": true
  }
}
```

---

## 4. Portfolios

### GET /portfolios

Daftar semua portfolio yang published (publik).

**Authentication:** Optional

**Query Parameters:**
| Parameter | Type | Description | Example |
|-----------|------|-------------|---------|
| search | string | Cari berdasarkan judul atau nama user | `?search=website` |
| tag_ids | string | Filter berdasarkan tag (comma-separated) | `?tag_ids=uuid1,uuid2` |
| major_id | UUID | Filter berdasarkan jurusan pembuat | `?major_id=xxx` |
| class_id | UUID | Filter berdasarkan kelas pembuat | `?class_id=xxx` |
| user_id | UUID | Filter berdasarkan user | `?user_id=xxx` |
| sort | string | Sorting: `-published_at`, `-like_count`, `title` | `?sort=-published_at` |
| page | integer | Halaman | `?page=1` |
| limit | integer | Jumlah per halaman | `?limit=20` |

**Success Response (200):**
```json
{
  "data": [
    {
      "id": "880e8400-e29b-41d4-a716-446655440000",
      "title": "Website Portfolio Pribadi",
      "slug": "website-portfolio-pribadi",
      "thumbnail_url": "https://cdn.grafikarsa.com/thumbnails/portfolio1.jpg",
      "published_at": "2026-02-01T10:00:00Z",
      "like_count": 45,
      "user": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "username": "john_doe",
        "name": "John Doe",
        "avatar_url": "https://cdn.grafikarsa.com/avatars/john.jpg",
        "role": "student",
        "class_name": "XII-RPL-A"
      },
      "tags": [
        { "id": "tag-uuid-1", "name": "Web Development" },
        { "id": "tag-uuid-2", "name": "UI/UX Design" }
      ]
    }
  ],
  "meta": {
    "current_page": 1,
    "per_page": 20,
    "total_pages": 10,
    "total_count": 200
  }
}
```

---

### GET /portfolios/{username}/{slug}

Detail portfolio berdasarkan username dan slug.

**Authentication:** Optional

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| username | string | Username pemilik portfolio |
| slug | string | Slug portfolio |

**Success Response (200):**
```json
{
  "data": {
    "id": "880e8400-e29b-41d4-a716-446655440000",
    "title": "Website Portfolio Pribadi",
    "slug": "website-portfolio-pribadi",
    "thumbnail_url": "https://cdn.grafikarsa.com/thumbnails/portfolio1.jpg",
    "status": "published",
    "published_at": "2026-02-01T10:00:00Z",
    "created_at": "2026-01-25T08:00:00Z",
    "updated_at": "2026-02-01T09:30:00Z",
    "like_count": 45,
    "is_liked": false,
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "username": "john_doe",
      "name": "John Doe",
      "avatar_url": "https://cdn.grafikarsa.com/avatars/john.jpg",
      "role": "student",
      "class_name": "XII-RPL-A"
    },
    "tags": [
      { "id": "tag-uuid-1", "name": "Web Development" },
      { "id": "tag-uuid-2", "name": "UI/UX Design" }
    ],
    "content_blocks": [
      {
        "id": "block-uuid-1",
        "block_type": "text",
        "block_order": 0,
        "payload": {
          "content": "<p>Ini adalah portfolio website pribadi saya...</p>"
        }
      },
      {
        "id": "block-uuid-2",
        "block_type": "image",
        "block_order": 1,
        "payload": {
          "url": "https://cdn.grafikarsa.com/images/screenshot1.jpg",
          "caption": "Tampilan homepage"
        }
      },
      {
        "id": "block-uuid-3",
        "block_type": "youtube",
        "block_order": 2,
        "payload": {
          "video_id": "dQw4w9WgXcQ"
        }
      }
    ]
  }
}
```

**Error Response:**

`404 Not Found`:
```json
{
  "error": {
    "code": "PORTFOLIO_NOT_FOUND",
    "message": "Portfolio tidak ditemukan"
  }
}
```

---

### GET /me/portfolios

Daftar semua portfolio milik user yang login (termasuk draft, pending, rejected, archived).

**Authentication:** Required

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| status | string | Filter: `draft`, `pending_review`, `rejected`, `published`, `archived` |
| page | integer | Halaman |
| limit | integer | Jumlah per halaman |

**Success Response (200):**
```json
{
  "data": [
    {
      "id": "880e8400-e29b-41d4-a716-446655440000",
      "title": "Website Portfolio Pribadi",
      "slug": "website-portfolio-pribadi",
      "thumbnail_url": "https://cdn.grafikarsa.com/thumbnails/portfolio1.jpg",
      "status": "published",
      "created_at": "2026-01-25T08:00:00Z",
      "updated_at": "2026-02-01T09:30:00Z",
      "like_count": 45
    },
    {
      "id": "880e8400-e29b-41d4-a716-446655440001",
      "title": "Desain Logo Keren",
      "slug": "desain-logo-keren",
      "thumbnail_url": null,
      "status": "draft",
      "created_at": "2026-02-05T08:00:00Z",
      "updated_at": "2026-02-05T08:00:00Z",
      "like_count": 0
    },
    {
      "id": "880e8400-e29b-41d4-a716-446655440002",
      "title": "Aplikasi Mobile",
      "slug": "aplikasi-mobile",
      "thumbnail_url": "https://cdn.grafikarsa.com/thumbnails/portfolio3.jpg",
      "status": "rejected",
      "admin_review_note": "Konten tidak sesuai dengan ketentuan. Mohon perbaiki bagian X.",
      "created_at": "2026-02-03T08:00:00Z",
      "updated_at": "2026-02-03T08:00:00Z",
      "like_count": 0
    }
  ],
  "meta": {
    "current_page": 1,
    "per_page": 20,
    "total_pages": 1,
    "total_count": 3
  }
}
```

---

### POST /portfolios

Buat portfolio baru (status default: draft).

**Authentication:** Required

**Request Body:**
```json
{
  "title": "Website Portfolio Pribadi",
  "tag_ids": ["tag-uuid-1", "tag-uuid-2"]
}
```

**Success Response (201):**
```json
{
  "data": {
    "id": "880e8400-e29b-41d4-a716-446655440000",
    "title": "Website Portfolio Pribadi",
    "slug": "website-portfolio-pribadi",
    "status": "draft",
    "thumbnail_url": null,
    "tags": [
      { "id": "tag-uuid-1", "name": "Web Development" },
      { "id": "tag-uuid-2", "name": "UI/UX Design" }
    ],
    "content_blocks": [],
    "created_at": "2026-02-09T10:00:00Z"
  },
  "message": "Portfolio berhasil dibuat"
}
```

**Error Responses:**

`422 Unprocessable Entity`:
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validasi gagal",
    "details": [
      {
        "field": "title",
        "message": "Judul wajib diisi"
      }
    ]
  }
}
```

`429 Too Many Requests` - Rate limit (max 10 portfolio/hari):
```json
{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Anda sudah mencapai batas maksimal pembuatan portfolio hari ini (10/hari)"
  }
}
```

---

### GET /portfolios/id/{id}

Detail portfolio berdasarkan ID (untuk edit).

**Authentication:** Required (owner atau admin)

**Success Response (200):** Sama dengan GET /portfolios/{username}/{slug}

---

### PATCH /portfolios/{id}

Update portfolio.

**Authentication:** Required (owner atau admin)

**Request Body:**
```json
{
  "title": "Website Portfolio Pribadi - Updated",
  "thumbnail_url": "https://cdn.grafikarsa.com/thumbnails/portfolio1.jpg",
  "tag_ids": ["tag-uuid-1", "tag-uuid-3"]
}
```

**Success Response (200):**
```json
{
  "data": {
    "id": "880e8400-e29b-41d4-a716-446655440000",
    "title": "Website Portfolio Pribadi - Updated",
    "slug": "website-portfolio-pribadi-updated",
    "status": "draft",
    "updated_at": "2026-02-09T11:00:00Z"
  },
  "message": "Portfolio berhasil diperbarui"
}
```

**Error Responses:**

`403 Forbidden`:
```json
{
  "error": {
    "code": "FORBIDDEN",
    "message": "Anda tidak memiliki akses untuk mengedit portfolio ini"
  }
}
```

`404 Not Found`:
```json
{
  "error": {
    "code": "PORTFOLIO_NOT_FOUND",
    "message": "Portfolio tidak ditemukan"
  }
}
```

---

### POST /portfolios/{id}/submit

Submit portfolio untuk review (ubah status dari draft ke pending_review).

**Authentication:** Required (owner)

**Success Response (200):**
```json
{
  "data": {
    "id": "880e8400-e29b-41d4-a716-446655440000",
    "status": "pending_review"
  },
  "message": "Portfolio berhasil diajukan untuk review"
}
```

**Error Responses:**

`400 Bad Request` - Status tidak valid:
```json
{
  "error": {
    "code": "INVALID_STATUS_TRANSITION",
    "message": "Portfolio hanya bisa disubmit dari status draft atau rejected"
  }
}
```

`422 Unprocessable Entity` - Portfolio belum lengkap:
```json
{
  "error": {
    "code": "INCOMPLETE_PORTFOLIO",
    "message": "Portfolio belum lengkap",
    "details": [
      {
        "field": "thumbnail",
        "message": "Thumbnail wajib diisi sebelum submit"
      },
      {
        "field": "content_blocks",
        "message": "Portfolio harus memiliki minimal 1 content block"
      }
    ]
  }
}
```

---

### POST /portfolios/{id}/archive

Arsipkan portfolio (sembunyikan dari publik).

**Authentication:** Required (owner atau admin)

**Success Response (200):**
```json
{
  "data": {
    "id": "880e8400-e29b-41d4-a716-446655440000",
    "status": "archived"
  },
  "message": "Portfolio berhasil diarsipkan"
}
```

---

### POST /portfolios/{id}/unarchive

Batalkan arsip (kembalikan ke status draft).

**Authentication:** Required (owner atau admin)

**Success Response (200):**
```json
{
  "data": {
    "id": "880e8400-e29b-41d4-a716-446655440000",
    "status": "draft"
  },
  "message": "Portfolio berhasil dikembalikan"
}
```

---

### DELETE /portfolios/{id}

Hapus portfolio (soft delete).

**Authentication:** Required (owner atau admin)

**Success Response (200):**
```json
{
  "message": "Portfolio berhasil dihapus"
}
```

---

## 5. Content Blocks

### Content Block Types

| Type | Description | Payload Structure |
|------|-------------|-------------------|
| `text` | Rich text / paragraf | `{ "content": "<p>HTML content</p>" }` |
| `image` | Gambar dengan caption | `{ "url": "https://...", "caption": "Optional" }` |
| `table` | Tabel dengan header & row | `{ "headers": [...], "rows": [[...], [...]] }` |
| `youtube` | Video YouTube embed | `{ "video_id": "dQw4w9WgXcQ" }` |
| `button` | Tombol custom dengan link | `{ "label": "Click Me", "url": "https://..." }` |

---

### POST /portfolios/{portfolio_id}/blocks

Tambah content block ke portfolio.

**Authentication:** Required (owner atau admin)

**Request Body - Text Block:**
```json
{
  "block_type": "text",
  "block_order": 0,
  "payload": {
    "content": "<p>Ini adalah paragraf pertama...</p><p>Paragraf kedua...</p>"
  }
}
```

**Request Body - Image Block:**
```json
{
  "block_type": "image",
  "block_order": 1,
  "payload": {
    "url": "https://cdn.grafikarsa.com/images/screenshot.jpg",
    "caption": "Screenshot aplikasi"
  }
}
```

**Request Body - YouTube Block:**
```json
{
  "block_type": "youtube",
  "block_order": 2,
  "payload": {
    "video_id": "dQw4w9WgXcQ"
  }
}
```

**Request Body - Table Block:**
```json
{
  "block_type": "table",
  "block_order": 3,
  "payload": {
    "headers": ["Fitur", "Deskripsi"],
    "rows": [
      ["Login", "Autentikasi user"],
      ["Dashboard", "Halaman utama"]
    ]
  }
}
```

**Request Body - Button Block:**
```json
{
  "block_type": "button",
  "block_order": 4,
  "payload": {
    "label": "Lihat Demo",
    "url": "https://demo.example.com"
  }
}
```

**Success Response (201):**
```json
{
  "data": {
    "id": "block-uuid-1",
    "block_type": "text",
    "block_order": 0,
    "payload": {
      "content": "<p>Ini adalah paragraf pertama...</p>"
    },
    "created_at": "2026-02-09T10:00:00Z"
  },
  "message": "Content block berhasil ditambahkan"
}
```

**Error Responses:**

`403 Forbidden`:
```json
{
  "error": {
    "code": "FORBIDDEN",
    "message": "Anda tidak memiliki akses ke portfolio ini"
  }
}
```

`422 Unprocessable Entity`:
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validasi gagal",
    "details": [
      {
        "field": "block_type",
        "message": "Block type tidak valid"
      }
    ]
  }
}
```

---

### PATCH /portfolios/{portfolio_id}/blocks/{block_id}

Update content block.

**Authentication:** Required (owner atau admin)

**Request Body:**
```json
{
  "payload": {
    "content": "<p>Konten yang sudah diupdate...</p>"
  }
}
```

**Success Response (200):**
```json
{
  "data": {
    "id": "block-uuid-1",
    "block_type": "text",
    "block_order": 0,
    "payload": {
      "content": "<p>Konten yang sudah diupdate...</p>"
    },
    "updated_at": "2026-02-09T11:00:00Z"
  },
  "message": "Content block berhasil diperbarui"
}
```

---

### PUT /portfolios/{portfolio_id}/blocks/reorder

Ubah urutan content blocks.

**Authentication:** Required (owner atau admin)

**Request Body:**
```json
{
  "block_orders": [
    { "id": "block-uuid-3", "order": 0 },
    { "id": "block-uuid-1", "order": 1 },
    { "id": "block-uuid-2", "order": 2 }
  ]
}
```

**Success Response (200):**
```json
{
  "message": "Urutan content blocks berhasil diperbarui"
}
```

---

### DELETE /portfolios/{portfolio_id}/blocks/{block_id}

Hapus content block.

**Authentication:** Required (owner atau admin)

**Success Response (200):**
```json
{
  "message": "Content block berhasil dihapus"
}
```

---

## 6. Tags

### GET /tags

Daftar semua tags.

**Authentication:** None

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| search | string | Cari berdasarkan nama tag |

**Success Response (200):**
```json
{
  "data": [
    { "id": "tag-uuid-1", "name": "Web Development" },
    { "id": "tag-uuid-2", "name": "Mobile App" },
    { "id": "tag-uuid-3", "name": "UI/UX Design" },
    { "id": "tag-uuid-4", "name": "Graphic Design" },
    { "id": "tag-uuid-5", "name": "3D Modeling" }
  ]
}
```

---

## 7. File Upload (MinIO Presigned URL)

Grafikarsa menggunakan MinIO sebagai object storage dengan strategi **Presigned URL** untuk upload file secara efisien.

### Upload Flow

```
┌─────────┐          ┌─────────┐          ┌─────────┐
│ Client  │          │ Backend │          │  MinIO  │
└────┬────┘          └────┬────┘          └────┬────┘
     │ 1. Request presigned URL               │
     │───────────────────>│                    │
     │                    │ 2. Generate URL    │
     │                    │───────────────────>│
     │ 3. Return presigned URL                │
     │<───────────────────│                    │
     │ 4. Upload file directly                │
     │────────────────────────────────────────>│
     │ 5. Upload success                      │
     │<────────────────────────────────────────│
     │ 6. Confirm upload                      │
     │───────────────────>│                    │
     │                    │ 7. Verify & update │
     │ 8. Return final URL                    │
     │<───────────────────│                    │
```

### Supported Upload Types

| Type | Purpose | Max Size | Allowed Types |
|------|---------|----------|---------------|
| `avatar` | User profile picture | 2 MB | jpg, jpeg, png, webp |
| `banner` | User profile banner | 2 MB | jpg, jpeg, png, webp |
| `thumbnail` | Portfolio thumbnail | 5 MB | jpg, jpeg, png, webp |
| `portfolio_image` | Image in content block | 5 MB | jpg, jpeg, png, webp, gif |

---

### POST /uploads/presign

Request presigned URL untuk upload file ke MinIO.

**Authentication:** Required

**Request Body:**
```json
{
  "upload_type": "avatar",
  "filename": "profile.jpg",
  "content_type": "image/jpeg",
  "file_size": 102400
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| upload_type | string | Yes | `avatar`, `banner`, `thumbnail`, `portfolio_image` |
| filename | string | Yes | Nama file asli |
| content_type | string | Yes | MIME type file |
| file_size | integer | Yes | Ukuran file dalam bytes |
| portfolio_id | UUID | Conditional | Wajib jika `thumbnail` atau `portfolio_image` |

**Request Body - Portfolio Thumbnail:**
```json
{
  "upload_type": "thumbnail",
  "filename": "thumbnail.png",
  "content_type": "image/png",
  "file_size": 512000,
  "portfolio_id": "880e8400-e29b-41d4-a716-446655440000"
}
```

**Success Response (200):**
```json
{
  "data": {
    "upload_id": "upload-uuid-123",
    "presigned_url": "https://minio.grafikarsa.com/grafikarsa/avatars/550e8400.../abc123.jpg?X-Amz-Algorithm=...",
    "object_key": "avatars/550e8400-e29b-41d4-a716-446655440000/abc123.jpg",
    "expires_in": 900,
    "method": "PUT",
    "headers": {
      "Content-Type": "image/jpeg"
    }
  }
}
```

**Error Responses:**

`400 Bad Request` - File size exceeds limit:
```json
{
  "error": {
    "code": "FILE_TOO_LARGE",
    "message": "Ukuran file melebihi batas maksimal",
    "details": [
      {
        "field": "file_size",
        "message": "Ukuran file avatar maksimal 2MB"
      }
    ]
  }
}
```

`400 Bad Request` - Invalid content type:
```json
{
  "error": {
    "code": "INVALID_CONTENT_TYPE",
    "message": "Tipe file tidak diizinkan",
    "details": [
      {
        "field": "content_type",
        "message": "Tipe file yang diizinkan: image/jpeg, image/png, image/webp"
      }
    ]
  }
}
```

`403 Forbidden` - Not owner of portfolio:
```json
{
  "error": {
    "code": "FORBIDDEN",
    "message": "Anda tidak memiliki akses untuk upload ke portfolio ini"
  }
}
```

---

### Client-Side Upload to MinIO

Setelah mendapat presigned URL, client upload langsung ke MinIO:

**JavaScript Example:**
```javascript
async function uploadToMinIO(presignedData, file) {
  const response = await fetch(presignedData.presigned_url, {
    method: presignedData.method,
    headers: presignedData.headers,
    body: file
  });
  if (!response.ok) throw new Error('Upload failed');
  return true;
}
```

---

### POST /uploads/confirm

Konfirmasi upload selesai dan update database.

**Authentication:** Required

**Request Body:**
```json
{
  "upload_id": "upload-uuid-123",
  "object_key": "avatars/550e8400-e29b-41d4-a716-446655440000/abc123.jpg"
}
```

**Success Response (200) - Avatar:**
```json
{
  "data": {
    "type": "avatar",
    "url": "https://cdn.grafikarsa.com/avatars/550e8400.../abc123.jpg",
    "object_key": "avatars/550e8400-e29b-41d4-a716-446655440000/abc123.jpg"
  },
  "message": "Avatar berhasil diperbarui"
}
```

**Success Response (200) - Portfolio Thumbnail:**
```json
{
  "data": {
    "type": "thumbnail",
    "url": "https://cdn.grafikarsa.com/thumbnails/880e8400.../abc123.jpg",
    "portfolio_id": "880e8400-e29b-41d4-a716-446655440000",
    "object_key": "thumbnails/880e8400-e29b-41d4-a716-446655440000/abc123.jpg"
  },
  "message": "Thumbnail portfolio berhasil diperbarui"
}
```

**Error Responses:**

`404 Not Found`:
```json
{
  "error": {
    "code": "UPLOAD_NOT_FOUND",
    "message": "Upload tidak ditemukan atau sudah expired"
  }
}
```

`400 Bad Request`:
```json
{
  "error": {
    "code": "OBJECT_NOT_FOUND",
    "message": "File tidak ditemukan di storage. Pastikan upload berhasil."
  }
}
```

---

### MinIO Bucket Structure

```
grafikarsa/
├── avatars/{user_id}/{uuid}.{ext}
├── banners/{user_id}/{uuid}.{ext}
├── thumbnails/{portfolio_id}/{uuid}.{ext}
└── portfolio-images/{portfolio_id}/{uuid}.{ext}
```

---

## 8. Social (Follow)

### POST /users/{username}/follow

Follow user.

**Authentication:** Required

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| username | string | Username user yang akan di-follow |

**Success Response (200):**
```json
{
  "data": {
    "is_following": true,
    "follower_count": 151
  },
  "message": "Berhasil follow john_doe"
}
```

**Error Responses:**

`400 Bad Request` - Tidak bisa follow diri sendiri:
```json
{
  "error": {
    "code": "CANNOT_FOLLOW_SELF",
    "message": "Tidak bisa follow diri sendiri"
  }
}
```

`409 Conflict` - Sudah follow:
```json
{
  "error": {
    "code": "ALREADY_FOLLOWING",
    "message": "Anda sudah follow user ini"
  }
}
```

`404 Not Found`:
```json
{
  "error": {
    "code": "USER_NOT_FOUND",
    "message": "User tidak ditemukan"
  }
}
```

---

### DELETE /users/{username}/follow

Unfollow user.

**Authentication:** Required

**Success Response (200):**
```json
{
  "data": {
    "is_following": false,
    "follower_count": 150
  },
  "message": "Berhasil unfollow john_doe"
}
```

**Error Response:**

`400 Bad Request`:
```json
{
  "error": {
    "code": "NOT_FOLLOWING",
    "message": "Anda belum follow user ini"
  }
}
```

---

## 9. Likes

### POST /portfolios/{id}/like

Like portfolio.

**Authentication:** Required

**Success Response (200):**
```json
{
  "data": {
    "is_liked": true,
    "like_count": 46
  },
  "message": "Portfolio berhasil di-like"
}
```

**Error Responses:**

`409 Conflict`:
```json
{
  "error": {
    "code": "ALREADY_LIKED",
    "message": "Anda sudah like portfolio ini"
  }
}
```

`404 Not Found`:
```json
{
  "error": {
    "code": "PORTFOLIO_NOT_FOUND",
    "message": "Portfolio tidak ditemukan"
  }
}
```

---

### DELETE /portfolios/{id}/like

Unlike portfolio.

**Authentication:** Required

**Success Response (200):**
```json
{
  "data": {
    "is_liked": false,
    "like_count": 45
  },
  "message": "Like berhasil dihapus"
}
```

---

## 10. Search

### GET /search/users

Cari user.

**Authentication:** Optional

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| q | string | Query pencarian (nama, username, bio) |
| major_id | UUID | Filter jurusan |
| class_id | UUID | Filter kelas |
| role | string | Filter role |
| page | integer | Halaman |
| limit | integer | Jumlah per halaman |

**Success Response (200):**
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "username": "john_doe",
      "name": "John Doe",
      "avatar_url": "https://cdn.grafikarsa.com/avatars/john.jpg",
      "bio": "Siswa RPL yang suka coding",
      "role": "student",
      "class_name": "XII-RPL-A",
      "major_name": "Rekayasa Perangkat Lunak"
    }
  ],
  "meta": {
    "current_page": 1,
    "per_page": 20,
    "total_pages": 1,
    "total_count": 5
  }
}
```

---

### GET /search/portfolios

Cari portfolio.

**Authentication:** Optional

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| q | string | Query pencarian (judul, nama user) |
| tag_ids | string | Filter tags (comma-separated) |
| major_id | UUID | Filter jurusan pembuat |
| class_id | UUID | Filter kelas pembuat |
| page | integer | Halaman |
| limit | integer | Jumlah per halaman |

**Success Response (200):** Sama dengan GET /portfolios

---

## 11. Feed

### GET /feed

Timeline portfolio dari user yang di-follow.

**Authentication:** Required

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| page | integer | Halaman |
| limit | integer | Jumlah per halaman |

**Success Response (200):**
```json
{
  "data": [
    {
      "id": "880e8400-e29b-41d4-a716-446655440000",
      "title": "Website Portfolio Pribadi",
      "slug": "website-portfolio-pribadi",
      "thumbnail_url": "https://cdn.grafikarsa.com/thumbnails/portfolio1.jpg",
      "published_at": "2026-02-09T10:00:00Z",
      "like_count": 45,
      "user": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "username": "john_doe",
        "name": "John Doe",
        "avatar_url": "https://cdn.grafikarsa.com/avatars/john.jpg",
        "role": "student",
        "class_name": "XII-RPL-A"
      },
      "tags": [
        { "id": "tag-uuid-1", "name": "Web Development" }
      ]
    }
  ],
  "meta": {
    "current_page": 1,
    "per_page": 20,
    "total_pages": 5,
    "total_count": 100
  }
}
```

---

## 12. Admin - Jurusan

> **Note:** Semua endpoint admin memerlukan role `admin`.

### GET /admin/majors

Daftar semua jurusan.

**Authentication:** Required (admin)

**Success Response (200):**
```json
{
  "data": [
    {
      "id": "770e8400-e29b-41d4-a716-446655440000",
      "name": "Rekayasa Perangkat Lunak",
      "code": "rpl",
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-01-01T00:00:00Z"
    },
    {
      "id": "770e8400-e29b-41d4-a716-446655440001",
      "name": "Teknik Komputer dan Jaringan",
      "code": "tkj",
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-01-01T00:00:00Z"
    }
  ]
}
```

---

### POST /admin/majors

Buat jurusan baru.

**Authentication:** Required (admin)

**Request Body:**
```json
{
  "name": "Desain Komunikasi Visual",
  "code": "dkv"
}
```

**Success Response (201):**
```json
{
  "data": {
    "id": "770e8400-e29b-41d4-a716-446655440002",
    "name": "Desain Komunikasi Visual",
    "code": "dkv",
    "created_at": "2026-02-09T10:00:00Z"
  },
  "message": "Jurusan berhasil dibuat"
}
```

**Error Responses:**

`409 Conflict`:
```json
{
  "error": {
    "code": "DUPLICATE_CODE",
    "message": "Kode jurusan sudah digunakan"
  }
}
```

`422 Unprocessable Entity`:
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validasi gagal",
    "details": [
      {
        "field": "code",
        "message": "Kode hanya boleh berisi huruf lowercase"
      }
    ]
  }
}
```

---

### PATCH /admin/majors/{id}

Update jurusan.

**Authentication:** Required (admin)

**Request Body:**
```json
{
  "name": "Desain Komunikasi Visual - Updated"
}
```

**Success Response (200):**
```json
{
  "data": {
    "id": "770e8400-e29b-41d4-a716-446655440002",
    "name": "Desain Komunikasi Visual - Updated",
    "code": "dkv",
    "updated_at": "2026-02-09T11:00:00Z"
  },
  "message": "Jurusan berhasil diperbarui"
}
```

---

### DELETE /admin/majors/{id}

Hapus jurusan (soft delete).

**Authentication:** Required (admin)

**Success Response (200):**
```json
{
  "message": "Jurusan berhasil dihapus"
}
```

**Error Response:**

`409 Conflict`:
```json
{
  "error": {
    "code": "MAJOR_IN_USE",
    "message": "Jurusan tidak bisa dihapus karena masih digunakan oleh kelas"
  }
}
```

---

## 13. Admin - Tahun Ajaran

### GET /admin/academic-years

Daftar semua tahun ajaran.

**Authentication:** Required (admin)

**Success Response (200):**
```json
{
  "data": [
    {
      "id": "990e8400-e29b-41d4-a716-446655440000",
      "year_start": 2025,
      "is_active": true,
      "promotion_month": 7,
      "promotion_day": 1,
      "created_at": "2025-01-01T00:00:00Z"
    },
    {
      "id": "990e8400-e29b-41d4-a716-446655440001",
      "year_start": 2024,
      "is_active": false,
      "promotion_month": 7,
      "promotion_day": 1,
      "created_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

---

### POST /admin/academic-years

Buat tahun ajaran baru.

**Authentication:** Required (admin)

**Request Body:**
```json
{
  "year_start": 2026,
  "is_active": false,
  "promotion_month": 7,
  "promotion_day": 1
}
```

**Success Response (201):**
```json
{
  "data": {
    "id": "990e8400-e29b-41d4-a716-446655440002",
    "year_start": 2026,
    "is_active": false,
    "promotion_month": 7,
    "promotion_day": 1,
    "created_at": "2026-02-09T10:00:00Z"
  },
  "message": "Tahun ajaran berhasil dibuat"
}
```

**Error Response:**

`409 Conflict`:
```json
{
  "error": {
    "code": "DUPLICATE_YEAR",
    "message": "Tahun ajaran sudah ada"
  }
}
```

---

### PATCH /admin/academic-years/{id}

Update tahun ajaran.

**Authentication:** Required (admin)

**Request Body:**
```json
{
  "is_active": true,
  "promotion_month": 7,
  "promotion_day": 15
}
```

> **Note:** Hanya satu tahun ajaran yang bisa aktif. Mengaktifkan satu akan menonaktifkan yang lain.

**Success Response (200):**
```json
{
  "data": {
    "id": "990e8400-e29b-41d4-a716-446655440002",
    "year_start": 2026,
    "is_active": true,
    "promotion_month": 7,
    "promotion_day": 15,
    "updated_at": "2026-02-09T11:00:00Z"
  },
  "message": "Tahun ajaran berhasil diperbarui"
}
```

---

### DELETE /admin/academic-years/{id}

Hapus tahun ajaran.

**Authentication:** Required (admin)

**Success Response (200):**
```json
{
  "message": "Tahun ajaran berhasil dihapus"
}
```

**Error Response:**

`409 Conflict`:
```json
{
  "error": {
    "code": "ACADEMIC_YEAR_IN_USE",
    "message": "Tahun ajaran tidak bisa dihapus karena masih memiliki kelas"
  }
}
```

---

## 14. Admin - Kelas

### GET /admin/classes

Daftar semua kelas.

**Authentication:** Required (admin)

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| academic_year_id | UUID | Filter berdasarkan tahun ajaran |
| major_id | UUID | Filter berdasarkan jurusan |
| grade_level | string | Filter berdasarkan tingkat (10, 11, 12) |

**Success Response (200):**
```json
{
  "data": [
    {
      "id": "660e8400-e29b-41d4-a716-446655440000",
      "name": "XII-RPL-A",
      "grade_level": "12",
      "group_letter": "A",
      "academic_year": {
        "id": "990e8400-e29b-41d4-a716-446655440000",
        "year_start": 2025
      },
      "major": {
        "id": "770e8400-e29b-41d4-a716-446655440000",
        "name": "Rekayasa Perangkat Lunak",
        "code": "rpl"
      },
      "student_count": 32,
      "created_at": "2025-01-01T00:00:00Z"
    }
  ],
  "meta": {
    "current_page": 1,
    "per_page": 50,
    "total_pages": 2,
    "total_count": 75
  }
}
```

---

### POST /admin/classes

Buat kelas baru. Nama kelas akan di-generate otomatis.

**Authentication:** Required (admin)

**Request Body:**
```json
{
  "academic_year_id": "990e8400-e29b-41d4-a716-446655440000",
  "major_id": "770e8400-e29b-41d4-a716-446655440000",
  "grade_level": "10",
  "group_letter": "A"
}
```

**Success Response (201):**
```json
{
  "data": {
    "id": "660e8400-e29b-41d4-a716-446655440001",
    "name": "X-RPL-A",
    "grade_level": "10",
    "group_letter": "A",
    "academic_year": {
      "id": "990e8400-e29b-41d4-a716-446655440000",
      "year_start": 2025
    },
    "major": {
      "id": "770e8400-e29b-41d4-a716-446655440000",
      "name": "Rekayasa Perangkat Lunak",
      "code": "rpl"
    },
    "created_at": "2026-02-09T10:00:00Z"
  },
  "message": "Kelas berhasil dibuat"
}
```

**Error Responses:**

`409 Conflict`:
```json
{
  "error": {
    "code": "DUPLICATE_CLASS",
    "message": "Kelas dengan kombinasi tahun ajaran, jurusan, tingkat, dan rombel ini sudah ada"
  }
}
```

`422 Unprocessable Entity`:
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validasi gagal",
    "details": [
      {
        "field": "group_letter",
        "message": "Rombel hanya boleh satu huruf A-Z"
      }
    ]
  }
}
```

---

### PATCH /admin/classes/{id}

Update kelas.

**Authentication:** Required (admin)

**Request Body:**
```json
{
  "group_letter": "B"
}
```

**Success Response (200):**
```json
{
  "data": {
    "id": "660e8400-e29b-41d4-a716-446655440001",
    "name": "X-RPL-B",
    "grade_level": "10",
    "group_letter": "B",
    "updated_at": "2026-02-09T11:00:00Z"
  },
  "message": "Kelas berhasil diperbarui"
}
```

---

### DELETE /admin/classes/{id}

Hapus kelas (soft delete).

**Authentication:** Required (admin)

**Success Response (200):**
```json
{
  "message": "Kelas berhasil dihapus"
}
```

**Error Response:**

`409 Conflict`:
```json
{
  "error": {
    "code": "CLASS_HAS_STUDENTS",
    "message": "Kelas tidak bisa dihapus karena masih memiliki siswa"
  }
}
```

---

## 15. Admin - Users

### GET /admin/users

Daftar semua user.

**Authentication:** Required (admin)

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| search | string | Cari berdasarkan nama, username, email |
| role | string | Filter role: `admin`, `student`, `alumni` |
| status | string | Filter status: `active`, `graduated`, `dropped_out`, `inactive` |
| class_id | UUID | Filter berdasarkan kelas |
| major_id | UUID | Filter berdasarkan jurusan |
| page | integer | Halaman |
| limit | integer | Jumlah per halaman |

**Success Response (200):**
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "username": "john_doe",
      "email": "john@example.com",
      "name": "John Doe",
      "role": "student",
      "status": "active",
      "avatar_url": "https://cdn.grafikarsa.com/avatars/john.jpg",
      "class": {
        "id": "660e8400-e29b-41d4-a716-446655440000",
        "name": "XII-RPL-A"
      },
      "major": {
        "id": "770e8400-e29b-41d4-a716-446655440000",
        "name": "Rekayasa Perangkat Lunak"
      },
      "entry_year": 2023,
      "created_at": "2023-07-15T08:00:00Z"
    }
  ],
  "meta": {
    "current_page": 1,
    "per_page": 20,
    "total_pages": 50,
    "total_count": 1000
  }
}
```

---

### POST /admin/users

Buat user baru.

**Authentication:** Required (admin)

**Request Body:**
```json
{
  "username": "new_student",
  "email": "newstudent@example.com",
  "password": "securepassword123",
  "name": "New Student",
  "role": "student",
  "status": "active",
  "nisn": "0098115882",
  "nis": "25491/02000.0412",
  "current_class_id": "660e8400-e29b-41d4-a716-446655440000",
  "entry_year": 2024
}
```

**Success Response (201):**
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440001",
    "username": "new_student",
    "email": "newstudent@example.com",
    "name": "New Student",
    "role": "student",
    "status": "active",
    "class": {
      "id": "660e8400-e29b-41d4-a716-446655440000",
      "name": "XII-RPL-A"
    },
    "created_at": "2026-02-09T10:00:00Z"
  },
  "message": "User berhasil dibuat"
}
```

**Error Responses:**

`409 Conflict`:
```json
{
  "error": {
    "code": "USERNAME_TAKEN",
    "message": "Username sudah digunakan"
  }
}
```

`409 Conflict`:
```json
{
  "error": {
    "code": "EMAIL_TAKEN",
    "message": "Email sudah digunakan"
  }
}
```

`422 Unprocessable Entity`:
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validasi gagal",
    "details": [
      {
        "field": "username",
        "message": "Username tidak boleh menggunakan reserved words"
      }
    ]
  }
}
```

---

### GET /admin/users/{id}

Detail user.

**Authentication:** Required (admin)

**Success Response (200):**
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "john_doe",
    "email": "john@example.com",
    "name": "John Doe",
    "bio": "Siswa RPL yang suka coding",
    "avatar_url": "https://cdn.grafikarsa.com/avatars/john.jpg",
    "banner_url": "https://cdn.grafikarsa.com/banners/john.jpg",
    "role": "student",
    "status": "active",
    "nisn": "0098115881",
    "nis": "25491/02000.0411",
    "entry_year": 2023,
    "graduation_year": null,
    "class": {
      "id": "660e8400-e29b-41d4-a716-446655440000",
      "name": "XII-RPL-A"
    },
    "major": {
      "id": "770e8400-e29b-41d4-a716-446655440000",
      "name": "Rekayasa Perangkat Lunak"
    },
    "social_links": {
      "github": "https://github.com/johndoe"
    },
    "class_history": [
      {
        "class_name": "X-RPL-A",
        "academic_year": 2023,
        "assigned_at": "2023-07-15T00:00:00Z"
      }
    ],
    "portfolio_count": 12,
    "follower_count": 150,
    "following_count": 75,
    "created_at": "2023-07-15T08:00:00Z",
    "updated_at": "2026-02-01T10:00:00Z"
  }
}
```

---

### PATCH /admin/users/{id}

Update user.

**Authentication:** Required (admin)

**Request Body:**
```json
{
  "name": "John Doe Updated",
  "role": "alumni",
  "status": "graduated",
  "graduation_year": 2026
}
```

**Success Response (200):**
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "john_doe",
    "name": "John Doe Updated",
    "role": "alumni",
    "status": "graduated",
    "graduation_year": 2026,
    "updated_at": "2026-02-09T11:00:00Z"
  },
  "message": "User berhasil diperbarui"
}
```

---

### PATCH /admin/users/{id}/password

Reset password user.

**Authentication:** Required (admin)

**Request Body:**
```json
{
  "new_password": "newpassword123"
}
```

**Success Response (200):**
```json
{
  "message": "Password berhasil direset"
}
```

---

### DELETE /admin/users/{id}

Hapus user (soft delete).

**Authentication:** Required (admin)

**Success Response (200):**
```json
{
  "message": "User berhasil dihapus"
}
```

---

## 16. Admin - Tags

### GET /admin/tags

Daftar semua tags.

**Authentication:** Required (admin)

**Success Response (200):**
```json
{
  "data": [
    {
      "id": "tag-uuid-1",
      "name": "Web Development",
      "portfolio_count": 45,
      "created_at": "2025-01-01T00:00:00Z"
    },
    {
      "id": "tag-uuid-2",
      "name": "Mobile App",
      "portfolio_count": 32,
      "created_at": "2025-01-01T00:00:00Z"
    }
  ]
}
```

---

### POST /admin/tags

Buat tag baru.

**Authentication:** Required (admin)

**Request Body:**
```json
{
  "name": "Machine Learning"
}
```

**Success Response (201):**
```json
{
  "data": {
    "id": "tag-uuid-new",
    "name": "Machine Learning",
    "created_at": "2026-02-09T10:00:00Z"
  },
  "message": "Tag berhasil dibuat"
}
```

**Error Response:**

`409 Conflict`:
```json
{
  "error": {
    "code": "DUPLICATE_TAG",
    "message": "Tag dengan nama ini sudah ada"
  }
}
```

---

### PATCH /admin/tags/{id}

Update tag.

**Authentication:** Required (admin)

**Request Body:**
```json
{
  "name": "Machine Learning & AI"
}
```

**Success Response (200):**
```json
{
  "data": {
    "id": "tag-uuid-new",
    "name": "Machine Learning & AI",
    "updated_at": "2026-02-09T11:00:00Z"
  },
  "message": "Tag berhasil diperbarui"
}
```

---

### DELETE /admin/tags/{id}

Hapus tag (soft delete).

**Authentication:** Required (admin)

**Success Response (200):**
```json
{
  "message": "Tag berhasil dihapus"
}
```

---

## 17. Admin - Portfolios

### GET /admin/portfolios

Daftar semua portfolio (semua status).

**Authentication:** Required (admin)

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| search | string | Cari berdasarkan judul, nama user |
| status | string | Filter status |
| user_id | UUID | Filter berdasarkan user |
| major_id | UUID | Filter berdasarkan jurusan |
| page | integer | Halaman |
| limit | integer | Jumlah per halaman |

**Success Response (200):**
```json
{
  "data": [
    {
      "id": "880e8400-e29b-41d4-a716-446655440000",
      "title": "Website Portfolio Pribadi",
      "slug": "website-portfolio-pribadi",
      "thumbnail_url": "https://cdn.grafikarsa.com/thumbnails/portfolio1.jpg",
      "status": "published",
      "like_count": 45,
      "user": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "username": "john_doe",
        "name": "John Doe",
        "class_name": "XII-RPL-A"
      },
      "created_at": "2026-01-25T08:00:00Z",
      "published_at": "2026-02-01T10:00:00Z"
    }
  ],
  "meta": {
    "current_page": 1,
    "per_page": 20,
    "total_pages": 10,
    "total_count": 200
  }
}
```

---

### GET /admin/portfolios/{id}

Detail portfolio (untuk admin review).

**Authentication:** Required (admin)

**Success Response (200):** Sama dengan GET /portfolios/{username}/{slug} dengan tambahan `admin_review_note`

---

### PATCH /admin/portfolios/{id}

Update portfolio (admin).

**Authentication:** Required (admin)

**Request Body:**
```json
{
  "title": "Updated Title by Admin",
  "tag_ids": ["tag-uuid-1", "tag-uuid-2"],
  "admin_review_note": "Catatan dari admin"
}
```

**Success Response (200):**
```json
{
  "data": {
    "id": "880e8400-e29b-41d4-a716-446655440000",
    "title": "Updated Title by Admin",
    "updated_at": "2026-02-09T11:00:00Z"
  },
  "message": "Portfolio berhasil diperbarui"
}
```

---

### DELETE /admin/portfolios/{id}

Hapus portfolio (soft delete).

**Authentication:** Required (admin)

**Success Response (200):**
```json
{
  "message": "Portfolio berhasil dihapus"
}
```

---

## 18. Admin - Moderasi

### GET /admin/moderation/queue

Daftar portfolio dengan status `pending_review`.

**Authentication:** Required (admin)

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| search | string | Cari berdasarkan judul, nama user |
| major_id | UUID | Filter berdasarkan jurusan |
| class_id | UUID | Filter berdasarkan kelas |
| sort | string | Sorting: `-created_at`, `created_at` |
| page | integer | Halaman |
| limit | integer | Jumlah per halaman |

**Success Response (200):**
```json
{
  "data": [
    {
      "id": "880e8400-e29b-41d4-a716-446655440000",
      "title": "Portfolio Baru",
      "slug": "portfolio-baru",
      "thumbnail_url": "https://cdn.grafikarsa.com/thumbnails/portfolio.jpg",
      "status": "pending_review",
      "submitted_at": "2026-02-09T08:00:00Z",
      "user": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "username": "john_doe",
        "name": "John Doe",
        "avatar_url": "https://cdn.grafikarsa.com/avatars/john.jpg",
        "class_name": "XII-RPL-A",
        "major_name": "Rekayasa Perangkat Lunak"
      },
      "tags": [
        { "id": "tag-uuid-1", "name": "Web Development" }
      ],
      "block_count": 5
    }
  ],
  "meta": {
    "current_page": 1,
    "per_page": 20,
    "total_pages": 1,
    "total_count": 8
  }
}
```

---

### POST /admin/moderation/{portfolio_id}/approve

Setujui portfolio (ubah status ke `published`).

**Authentication:** Required (admin)

**Request Body (optional):**
```json
{
  "admin_review_note": "Portfolio bagus!"
}
```

**Success Response (200):**
```json
{
  "data": {
    "id": "880e8400-e29b-41d4-a716-446655440000",
    "status": "published",
    "published_at": "2026-02-09T10:00:00Z"
  },
  "message": "Portfolio berhasil disetujui dan dipublish"
}
```

**Error Response:**

`400 Bad Request`:
```json
{
  "error": {
    "code": "INVALID_STATUS",
    "message": "Hanya portfolio dengan status pending_review yang bisa disetujui"
  }
}
```

---

### POST /admin/moderation/{portfolio_id}/reject

Tolak portfolio (ubah status ke `rejected`).

**Authentication:** Required (admin)

**Request Body:**
```json
{
  "admin_review_note": "Konten tidak sesuai ketentuan. Mohon perbaiki bagian X."
}
```

**Success Response (200):**
```json
{
  "data": {
    "id": "880e8400-e29b-41d4-a716-446655440000",
    "status": "rejected",
    "admin_review_note": "Konten tidak sesuai ketentuan. Mohon perbaiki bagian X."
  },
  "message": "Portfolio ditolak"
}
```

**Error Responses:**

`400 Bad Request`:
```json
{
  "error": {
    "code": "INVALID_STATUS",
    "message": "Hanya portfolio dengan status pending_review yang bisa ditolak"
  }
}
```

`422 Unprocessable Entity`:
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validasi gagal",
    "details": [
      {
        "field": "admin_review_note",
        "message": "Alasan penolakan wajib diisi"
      }
    ]
  }
}
```

---

## 19. Public - Jurusan & Kelas

### GET /majors

Daftar semua jurusan (untuk dropdown/filter publik).

**Authentication:** None

**Success Response (200):**
```json
{
  "data": [
    {
      "id": "770e8400-e29b-41d4-a716-446655440000",
      "name": "Rekayasa Perangkat Lunak",
      "code": "rpl"
    },
    {
      "id": "770e8400-e29b-41d4-a716-446655440001",
      "name": "Teknik Komputer dan Jaringan",
      "code": "tkj"
    },
    {
      "id": "770e8400-e29b-41d4-a716-446655440002",
      "name": "Desain Komunikasi Visual",
      "code": "dkv"
    },
    {
      "id": "770e8400-e29b-41d4-a716-446655440003",
      "name": "Animasi",
      "code": "ani"
    }
  ]
}
```

---

### GET /classes

Daftar kelas dari tahun ajaran aktif (untuk dropdown/filter publik).

**Authentication:** None

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| major_id | UUID | Filter berdasarkan jurusan |
| grade_level | string | Filter berdasarkan tingkat (10, 11, 12) |

**Success Response (200):**
```json
{
  "data": [
    {
      "id": "660e8400-e29b-41d4-a716-446655440000",
      "name": "XII-RPL-A",
      "grade_level": "12",
      "major": {
        "id": "770e8400-e29b-41d4-a716-446655440000",
        "name": "Rekayasa Perangkat Lunak"
      }
    },
    {
      "id": "660e8400-e29b-41d4-a716-446655440001",
      "name": "XII-RPL-B",
      "grade_level": "12",
      "major": {
        "id": "770e8400-e29b-41d4-a716-446655440000",
        "name": "Rekayasa Perangkat Lunak"
      }
    }
  ]
}
```

---

## Common Error Codes

Berikut adalah daftar error code yang umum digunakan di seluruh API:

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `INVALID_CREDENTIALS` | 401 | Username atau password salah |
| `TOKEN_EXPIRED` | 401 | Token sudah expired |
| `TOKEN_REUSE_DETECTED` | 401 | Refresh token reuse attack terdeteksi |
| `UNAUTHORIZED` | 401 | Token tidak valid atau tidak ada |
| `FORBIDDEN` | 403 | Tidak punya akses ke resource |
| `ACCOUNT_DISABLED` | 403 | Akun dinonaktifkan |
| `NOT_FOUND` | 404 | Resource tidak ditemukan |
| `USER_NOT_FOUND` | 404 | User tidak ditemukan |
| `PORTFOLIO_NOT_FOUND` | 404 | Portfolio tidak ditemukan |
| `SESSION_NOT_FOUND` | 404 | Session tidak ditemukan |
| `ALREADY_FOLLOWING` | 409 | Sudah follow user |
| `ALREADY_LIKED` | 409 | Sudah like portfolio |
| `USERNAME_TAKEN` | 409 | Username sudah digunakan |
| `EMAIL_TAKEN` | 409 | Email sudah digunakan |
| `DUPLICATE_CODE` | 409 | Kode sudah ada (jurusan) |
| `DUPLICATE_TAG` | 409 | Tag sudah ada |
| `DUPLICATE_YEAR` | 409 | Tahun ajaran sudah ada |
| `DUPLICATE_CLASS` | 409 | Kelas sudah ada |
| `VALIDATION_ERROR` | 422 | Validasi input gagal |
| `INCOMPLETE_PORTFOLIO` | 422 | Portfolio belum lengkap |
| `INVALID_STATUS_TRANSITION` | 400 | Transisi status tidak valid |
| `FILE_TOO_LARGE` | 400 | Ukuran file melebihi batas |
| `INVALID_CONTENT_TYPE` | 400 | Tipe file tidak diizinkan |
| `RATE_LIMIT_EXCEEDED` | 429 | Rate limit terlampaui |
| `INTERNAL_ERROR` | 500 | Server error |

---

## Rate Limiting

API menggunakan rate limiting untuk mencegah abuse:

| Scope | Limit |
|-------|-------|
| General API | 100 requests/minute per IP |
| Auth endpoints | 10 requests/minute per IP |
| Portfolio creation | 10 portfolios/day per user |

**Response Headers:**
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1707465600
```

---

## Reserved Usernames

Berikut adalah username yang tidak boleh digunakan:

`admin`, `dashboard`, `login`, `register`, `api`, `feed`, `explore`, `search`, `settings`, `profile`, `logout`, `grafikarsa`

---
