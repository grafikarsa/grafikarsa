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
  "success": true,
  "data": { ... },
  "meta": { ... }
}
```

**Error Response:**
```json
{
  "success": false,
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
7. [Series](#7-series)
8. [File Upload (MinIO)](#8-file-upload-minio)
9. [Social (Follow)](#9-social-follow)
10. [Likes](#10-likes)
11. [Search](#11-search)
12. [Feed](#12-feed)
13. [Admin - Jurusan](#13-admin---jurusan)
14. [Admin - Tahun Ajaran](#14-admin---tahun-ajaran)
15. [Admin - Kelas](#15-admin---kelas)
16. [Admin - Users](#16-admin---users)
17. [Admin - Tags](#17-admin---tags)
18. [Admin - Series](#18-admin---series)
19. [Admin - Moderasi](#19-admin---moderasi)
20. [Admin - Dashboard](#20-admin---dashboard)
21. [Public - Jurusan & Kelas](#21-public---jurusan--kelas)
22. [Feedback](#22-feedback)
23. [Admin - Assessment Metrics](#23-admin---assessment-metrics)
24. [Admin - Portfolio Assessments](#24-admin---portfolio-assessments)
25. [Notifications](#25-notifications)
26. [Admin - Special Roles](#26-admin---special-roles)
27. [Changelog](#27-changelog)

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
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "token_type": "Bearer",
    "expires_in": 900,
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "username": "john_doe",
      "nama": "John Doe",
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
  "success": false,
  "error": {
    "code": "INVALID_CREDENTIALS",
    "message": "Username atau password salah"
  }
}
```

`403 Forbidden` - Akun nonaktif:
```json
{
  "success": false,
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
  "success": true,
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
  "success": false,
  "error": {
    "code": "TOKEN_EXPIRED",
    "message": "Refresh token telah expired. Silakan login ulang."
  }
}
```

`401 Unauthorized` - Token reuse detected (security alert):
```json
{
  "success": false,
  "error": {
    "code": "TOKEN_REUSE_DETECTED",
    "message": "Aktivitas mencurigakan terdeteksi. Semua sesi telah diakhiri. Silakan login ulang."
  }
}
```

---

### POST /auth/logout

Logout dari sesi saat ini.

**Authentication:** Required

**Request Headers:**
```
Authorization: Bearer <access_token>
```

**Success Response (200):**
```json
{
  "success": true,
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
  "success": true,
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
  "success": true,
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440001",
      "device_info": {
        "user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64)...",
        "device_type": "desktop",
        "browser": "Chrome",
        "os": "Windows 10"
      },
      "ip_address": "192.168.1.1",
      "created_at": "2025-12-09T10:00:00Z",
      "last_used_at": "2025-12-09T14:30:00Z",
      "is_current": true
    },
    {
      "id": "550e8400-e29b-41d4-a716-446655440002",
      "device_info": {
        "user_agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 15_0)...",
        "device_type": "mobile",
        "browser": "Safari",
        "os": "iOS 15"
      },
      "ip_address": "192.168.1.2",
      "created_at": "2025-12-08T08:00:00Z",
      "is_current": false
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
  "success": true,
  "message": "Sesi berhasil dihapus"
}
```

**Error Response:**

`404 Not Found`:
```json
{
  "success": false,
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
| jurusan_id | UUID | Filter berdasarkan jurusan | `?jurusan_id=xxx` |
| kelas_id | UUID | Filter berdasarkan kelas | `?kelas_id=xxx` |
| role | string | Filter berdasarkan role | `?role=student` |
| page | integer | Halaman (default: 1) | `?page=2` |
| limit | integer | Jumlah per halaman (default: 20, max: 50) | `?limit=20` |

**Success Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "username": "john_doe",
      "nama": "John Doe",
      "avatar_url": "https://cdn.grafikarsa.com/avatars/john.jpg",
      "role": "student",
      "kelas": {
        "id": "660e8400-e29b-41d4-a716-446655440000",
        "nama": "XII-RPL-A"
      },
      "jurusan": {
        "id": "770e8400-e29b-41d4-a716-446655440000",
        "nama": "Rekayasa Perangkat Lunak"
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
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "john_doe",
    "nama": "John Doe",
    "bio": "Siswa RPL yang suka coding dan desain",
    "avatar_url": "https://cdn.grafikarsa.com/avatars/john.jpg",
    "banner_url": "https://cdn.grafikarsa.com/banners/john.jpg",
    "role": "student",
    "tahun_masuk": 2023,
    "tahun_lulus": 2026,
    "kelas": {
      "id": "660e8400-e29b-41d4-a716-446655440000",
      "nama": "XII-RPL-A"
    },
    "jurusan": {
      "id": "770e8400-e29b-41d4-a716-446655440000",
      "nama": "Rekayasa Perangkat Lunak"
    },
    "class_history": [
      {
        "kelas_nama": "X-RPL-A",
        "tahun_ajaran": 2023
      },
      {
        "kelas_nama": "XI-RPL-A",
        "tahun_ajaran": 2024
      },
      {
        "kelas_nama": "XII-RPL-A",
        "tahun_ajaran": 2025
      }
    ],
    "social_links": [
      {
        "platform": "github",
        "url": "https://github.com/johndoe"
      },
      {
        "platform": "instagram",
        "url": "https://instagram.com/johndoe"
      }
    ],
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
  "success": false,
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
  "success": true,
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440001",
      "username": "jane_doe",
      "nama": "Jane Doe",
      "avatar_url": "https://cdn.grafikarsa.com/avatars/jane.jpg",
      "role": "student",
      "kelas_nama": "XI-MM-B",
      "is_following": true,
      "followed_at": "2025-12-01T10:00:00Z"
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
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "john_doe",
    "email": "john@example.com",
    "nama": "John Doe",
    "bio": "Siswa RPL yang suka coding dan desain",
    "avatar_url": "https://cdn.grafikarsa.com/avatars/john.jpg",
    "banner_url": "https://cdn.grafikarsa.com/banners/john.jpg",
    "role": "student",
    "nisn": "0098115881",
    "nis": "25491/02000.0411",
    "tahun_masuk": 2023,
    "tahun_lulus": 2026,
    "kelas": {
      "id": "660e8400-e29b-41d4-a716-446655440000",
      "nama": "XII-RPL-A"
    },
    "jurusan": {
      "id": "770e8400-e29b-41d4-a716-446655440000",
      "nama": "Rekayasa Perangkat Lunak"
    },
    "social_links": [
      {
        "platform": "github",
        "url": "https://github.com/johndoe"
      }
    ],
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
  "nama": "John Doe Updated",
  "username": "john_doe_new",
  "bio": "Updated bio",
  "email": "newemail@example.com"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "john_doe_new",
    "email": "newemail@example.com",
    "nama": "John Doe Updated",
    "bio": "Updated bio"
  },
  "message": "Profil berhasil diperbarui"
}
```

**Error Responses:**

`409 Conflict` - Username sudah dipakai:
```json
{
  "success": false,
  "error": {
    "code": "USERNAME_TAKEN",
    "message": "Username sudah digunakan"
  }
}
```

`422 Unprocessable Entity` - Validasi gagal:
```json
{
  "success": false,
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
  "success": true,
  "message": "Password berhasil diubah"
}
```

**Error Responses:**

`400 Bad Request` - Password lama salah:
```json
{
  "success": false,
  "error": {
    "code": "INVALID_PASSWORD",
    "message": "Password lama tidak sesuai"
  }
}
```

`422 Unprocessable Entity`:
```json
{
  "success": false,
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

### Upload Avatar

Untuk upload avatar, gunakan [File Upload (MinIO)](#7-file-upload-minio) dengan `upload_type: "avatar"`.

**Flow:**
1. `POST /uploads/presign` dengan `upload_type: "avatar"`
2. Upload file ke MinIO menggunakan presigned URL
3. `POST /uploads/confirm` untuk update avatar_url di database

---

### Upload Banner

Untuk upload banner, gunakan [File Upload (MinIO)](#7-file-upload-minio) dengan `upload_type: "banner"`.

**Flow:**
1. `POST /uploads/presign` dengan `upload_type: "banner"`
2. Upload file ke MinIO menggunakan presigned URL
3. `POST /uploads/confirm` untuk update banner_url di database

---

### PUT /me/social-links

Update semua social links sekaligus.

**Authentication:** Required

**Request Body:**
```json
{
  "social_links": [
    {
      "platform": "github",
      "url": "https://github.com/johndoe"
    },
    {
      "platform": "instagram",
      "url": "https://instagram.com/johndoe"
    },
    {
      "platform": "linkedin",
      "url": "https://linkedin.com/in/johndoe"
    }
  ]
}
```

**Valid Platforms:**
`facebook`, `instagram`, `github`, `linkedin`, `twitter`, `personal_website`, `tiktok`, `youtube`, `behance`, `dribbble`, `threads`, `bluesky`, `medium`, `gitlab`

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "social_links": [
      {
        "platform": "github",
        "url": "https://github.com/johndoe"
      },
      {
        "platform": "instagram",
        "url": "https://instagram.com/johndoe"
      },
      {
        "platform": "linkedin",
        "url": "https://linkedin.com/in/johndoe"
      }
    ]
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
  "success": true,
  "data": {
    "username": "new_username",
    "available": true
  }
}
```

```json
{
  "success": true,
  "data": {
    "username": "existing_user",
    "available": false
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
| jurusan_id | UUID | Filter berdasarkan jurusan pembuat | `?jurusan_id=xxx` |
| kelas_id | UUID | Filter berdasarkan kelas pembuat | `?kelas_id=xxx` |
| user_id | UUID | Filter berdasarkan user | `?user_id=xxx` |
| sort | string | Sorting: `-published_at`, `-like_count`, `judul` | `?sort=-published_at` |
| page | integer | Halaman | `?page=1` |
| limit | integer | Jumlah per halaman | `?limit=20` |

**Success Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "880e8400-e29b-41d4-a716-446655440000",
      "judul": "Website Portfolio Pribadi",
      "slug": "website-portfolio-pribadi",
      "thumbnail_url": "https://cdn.grafikarsa.com/thumbnails/portfolio1.jpg",
      "published_at": "2025-12-01T10:00:00Z",
      "like_count": 45,
      "user": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "username": "john_doe",
        "nama": "John Doe",
        "avatar_url": "https://cdn.grafikarsa.com/avatars/john.jpg",
        "role": "student",
        "kelas_nama": "XII-RPL-A"
      },
      "tags": [
        { "id": "tag-uuid-1", "nama": "Web Development" },
        { "id": "tag-uuid-2", "nama": "UI/UX Design" }
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

### GET /portfolios/{slug}

Detail portfolio berdasarkan slug.

**Authentication:** Optional

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| slug | string | Slug portfolio |

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| username | string | Username pemilik (untuk unique slug per user) |

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "id": "880e8400-e29b-41d4-a716-446655440000",
    "judul": "Website Portfolio Pribadi",
    "slug": "website-portfolio-pribadi",
    "thumbnail_url": "https://cdn.grafikarsa.com/thumbnails/portfolio1.jpg",
    "status": "published",
    "published_at": "2025-12-01T10:00:00Z",
    "created_at": "2025-11-25T08:00:00Z",
    "updated_at": "2025-12-01T09:30:00Z",
    "like_count": 45,
    "is_liked": false,
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "username": "john_doe",
      "nama": "John Doe",
      "avatar_url": "https://cdn.grafikarsa.com/avatars/john.jpg",
      "role": "student",
      "kelas_nama": "XII-RPL-A"
    },
    "tags": [
      { "id": "tag-uuid-1", "nama": "Web Development" },
      { "id": "tag-uuid-2", "nama": "UI/UX Design" }
    ],
    "series": {
      "id": "series-uuid-1",
      "nama": "PJBL Semester 1 2024"
    },
    "content_blocks": [
      {
        "id": "block-uuid-1",
        "block_type": "text",
        "block_order": 0,
        "payload": {
          "content": "<p>Ini adalah portfolio website pribadi saya...</p>"
        },
        "series_instruksi": "Judul dan deskripsi singkat proyek"
      },
      {
        "id": "block-uuid-2",
        "block_type": "image",
        "block_order": 1,
        "payload": {
          "url": "https://cdn.grafikarsa.com/images/screenshot1.jpg",
          "caption": "Tampilan homepage"
        },
        "series_instruksi": "Thumbnail/cover proyek (rasio 16:9 recommended)"
      },
      {
        "id": "block-uuid-3",
        "block_type": "youtube",
        "block_order": 2,
        "payload": {
          "video_id": "dQw4w9WgXcQ",
          "title": "Demo Video"
        },
        "series_instruksi": "Video dokumentasi presentasi PJBL"
      }
    ]
  }
}
```

**Error Response:**

`404 Not Found`:
```json
{
  "success": false,
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
| status | string | Filter status: `draft`, `pending_review`, `rejected`, `published`, `archived` |
| page | integer | Halaman |
| limit | integer | Jumlah per halaman |

**Success Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "880e8400-e29b-41d4-a716-446655440000",
      "judul": "Website Portfolio Pribadi",
      "slug": "website-portfolio-pribadi",
      "thumbnail_url": "https://cdn.grafikarsa.com/thumbnails/portfolio1.jpg",
      "status": "published",
      "created_at": "2025-11-25T08:00:00Z",
      "updated_at": "2025-12-01T09:30:00Z",
      "like_count": 45
    },
    {
      "id": "880e8400-e29b-41d4-a716-446655440001",
      "judul": "Desain Logo Keren",
      "slug": "desain-logo-keren",
      "thumbnail_url": null,
      "status": "draft",
      "created_at": "2025-12-05T08:00:00Z",
      "updated_at": "2025-12-05T08:00:00Z",
      "like_count": 0
    },
    {
      "id": "880e8400-e29b-41d4-a716-446655440002",
      "judul": "Aplikasi Mobile",
      "slug": "aplikasi-mobile",
      "thumbnail_url": "https://cdn.grafikarsa.com/thumbnails/portfolio3.jpg",
      "status": "rejected",
      "admin_review_note": "Konten tidak sesuai dengan ketentuan. Mohon perbaiki bagian X.",
      "created_at": "2025-12-03T08:00:00Z",
      "updated_at": "2025-12-03T08:00:00Z",
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
  "judul": "Website Portfolio Pribadi",
  "tag_ids": ["tag-uuid-1", "tag-uuid-2"],
  "series_id": "series-uuid-1"
}
```

**Note:** `series_id` adalah optional. Jika disertakan, portfolio akan menggunakan template blocks dari series tersebut dan blocks akan di-generate otomatis berdasarkan template.

**Success Response (201):**
```json
{
  "success": true,
  "data": {
    "id": "880e8400-e29b-41d4-a716-446655440000",
    "judul": "Website Portfolio Pribadi",
    "slug": "website-portfolio-pribadi",
    "status": "draft",
    "thumbnail_url": null,
    "tags": [
      { "id": "tag-uuid-1", "nama": "Web Development" },
      { "id": "tag-uuid-2", "nama": "UI/UX Design" }
    ],
    "series": {
      "id": "series-uuid-1",
      "nama": "PJBL Semester 1 2024"
    },
    "content_blocks": [
      {
        "id": "block-uuid-1",
        "block_type": "text",
        "block_order": 0,
        "payload": {},
        "series_instruksi": "Judul dan deskripsi singkat proyek"
      },
      {
        "id": "block-uuid-2",
        "block_type": "image",
        "block_order": 1,
        "payload": {},
        "series_instruksi": "Thumbnail/cover proyek (rasio 16:9 recommended)"
      }
    ],
    "created_at": "2025-12-09T10:00:00Z"
  },
  "message": "Portfolio berhasil dibuat"
}
```

**Error Response:**

`422 Unprocessable Entity`:
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validasi gagal",
    "details": [
      {
        "field": "judul",
        "message": "Judul wajib diisi"
      }
    ]
  }
}
```

---

### GET /portfolios/id/{id}

Detail portfolio berdasarkan ID (untuk edit).

**Authentication:** Required (owner atau admin)

**Success Response (200):** Sama dengan GET /portfolios/{slug}

---

### PATCH /portfolios/{id}

Update portfolio.

**Authentication:** Required (owner atau admin)

**Request Body:**
```json
{
  "judul": "Website Portfolio Pribadi - Updated",
  "thumbnail_url": "https://cdn.grafikarsa.com/thumbnails/portfolio1.jpg",
  "tag_ids": ["tag-uuid-1", "tag-uuid-3"],
  "series_id": "series-uuid-1"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "id": "880e8400-e29b-41d4-a716-446655440000",
    "judul": "Website Portfolio Pribadi - Updated",
    "slug": "website-portfolio-pribadi-updated",
    "status": "draft",
    "updated_at": "2025-12-09T11:00:00Z"
  },
  "message": "Portfolio berhasil diperbarui"
}
```

**Error Responses:**

`403 Forbidden`:
```json
{
  "success": false,
  "error": {
    "code": "FORBIDDEN",
    "message": "Anda tidak memiliki akses untuk mengedit portfolio ini"
  }
}
```

`404 Not Found`:
```json
{
  "success": false,
  "error": {
    "code": "PORTFOLIO_NOT_FOUND",
    "message": "Portfolio tidak ditemukan"
  }
}
```

---

### Upload portfoliio Thumbnail

Untuk upload thumbnail portfolio gunakan [File Upload (MinIO)](#7-file-upload-minio) dengan `upload_type: "thumbnail"`.

**Flow:**
1. `POST /uploads/presign` dengan `upload_type: "thumbnail"` dan `portfolio_id`
2. Upload file ke MinIO menggunakan presigned URL
3. `POST /uploads/confirm` untuk update thumbnail_url di database

---

### POST /portfolios/{id}/submit

Submit portfolio untuk review (ubah status dari draft ke pending_review).

**Authentication:** Required (owner)

**Request Body:** None

**Success Response (200):**
```json
{
  "success": true,
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
  "success": false,
  "error": {
    "code": "INVALID_STATUS_TRANSITION",
    "message": "Portfolio hanya bisa disubmit dari status draft atau rejected"
  }
}
```

`422 Unprocessable Entity` - Portfolio belum lengkap:
```json
{
  "success": false,
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
  "success": true,
  "data": {
    "id": "880e8400-e29b-41d4-a716-446655440000",
    "status": "archived"
  },
  "message": "Portfolio berhasil diarsipkan"
}
```

---

### POST /portfolios/{id}/unarchive

Batalkan arsip (kembalikan ke status sebelumnya atau draft).

**Authentication:** Required (owner atau admin)

**Success Response (200):**
```json
{
  "success": true,
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
  "success": true,
  "message": "Portfolio berhasil dihapus"
}
```

---

## 5. Content Blocks

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
    "video_id": "dQw4w9WgXcQ",
    "title": "Demo Video"
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
    "text": "Lihat Demo",
    "url": "https://demo.example.com"
  }
}
```

**Request Body - Embed Block:**
```json
{
  "block_type": "embed",
  "block_order": 5,
  "payload": {
    "html": "<iframe src=\"https://codepen.io/...\" ...></iframe>",
    "title": "CodePen Demo"
  }
}
```

**Success Response (201):**
```json
{
  "success": true,
  "data": {
    "id": "block-uuid-1",
    "block_type": "text",
    "block_order": 0,
    "payload": {
      "content": "<p>Ini adalah paragraf pertama...</p>"
    },
    "created_at": "2025-12-09T10:00:00Z"
  },
  "message": "Content block berhasil ditambahkan"
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
  "success": true,
  "data": {
    "id": "block-uuid-1",
    "block_type": "text",
    "block_order": 0,
    "payload": {
      "content": "<p>Konten yang sudah diupdate...</p>"
    },
    "updated_at": "2025-12-09T11:00:00Z"
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
  "success": true,
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
  "success": true,
  "message": "Content block berhasil dihapus"
}
```

---

### Upload Image untuk Content Block

Untuk upload gambar di content block, gunakan [File Upload (MinIO)](#7-filenio) dengan `upload_type: "portfolio_image"`.

**Flow:**
1. `POST /uploads/presign` dengan `upload_type: "portfolio_image"`, `portfolio_id`, dan `block_id`
2. Upload file ke MinIO menggunakan presigned URL
3. `POST /uploads/confirm` untuk mendapat URL final
4. Update content block payload dengan URL gambar yang didapat

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
  "success": true,
  "data": [
    { "id": "tag-uuid-1", "nama": "Web Development" },
    { "id": "tag-uuid-2", "nama": "Mobile App" },
    { "id": "tag-uuid-3", "nama": "UI/UX Design" },
    { "id": "tag-uuid-4", "nama": "Graphic Design" },
    { "id": "tag-uuid-5", "nama": "3D Modeling" }
  ]
}
```

---

## 7. Series

Series adalah **template portofolio** yang mendefinisikan struktur block konten beserta instruksi untuk setiap block. Siswa yang memilih series akan mendapatkan template block yang sudah ditentukan dan tidak dapat diubah urutannya (locked blocks).

**Konsep:**
- Series mendefinisikan template blocks dengan instruksi per block
- Relasi portfolio ke series adalah many-to-one (banyak portfolio → 1 series, atau tanpa series)
- Siswa tidak dapat menambah/hapus/reorder blocks jika menggunakan series
- Series nonaktif hanya hilang dari dropdown, tidak mempengaruhi portfolio yang sudah ada

### GET /series

Daftar semua series yang aktif (untuk dropdown di form portfolio).

**Authentication:** None

**Success Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "series-uuid-1",
      "nama": "PJBL Semester 1 2024",
      "deskripsi": "Template untuk proyek PJBL semester 1",
      "block_count": 7
    },
    {
      "id": "series-uuid-2",
      "nama": "Ujian Praktik 2024",
      "deskripsi": "Template untuk ujian praktik kejuruan",
      "block_count": 5
    }
  ]
}
```

---

### GET /series/{id}

Detail series dengan blocks template (untuk generate blocks saat siswa memilih series).

**Authentication:** None

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | UUID | ID series |

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "id": "series-uuid-1",
    "nama": "PJBL Semester 1 2024",
    "deskripsi": "Template untuk proyek PJBL semester 1 tahun ajaran 2024/2025",
    "block_count": 7,
    "blocks": [
      {
        "id": "block-uuid-1",
        "block_type": "text",
        "block_order": 0,
        "instruksi": "Judul dan deskripsi singkat proyek"
      },
      {
        "id": "block-uuid-2",
        "block_type": "image",
        "block_order": 1,
        "instruksi": "Thumbnail/cover proyek (rasio 16:9 recommended)"
      },
      {
        "id": "block-uuid-3",
        "block_type": "youtube",
        "block_order": 2,
        "instruksi": "Video dokumentasi presentasi PJBL"
      },
      {
        "id": "block-uuid-4",
        "block_type": "text",
        "block_order": 3,
        "instruksi": "Tujuan pembuatan proyek"
      },
      {
        "id": "block-uuid-5",
        "block_type": "text",
        "block_order": 4,
        "instruksi": "Proses pengerjaan"
      },
      {
        "id": "block-uuid-6",
        "block_type": "image",
        "block_order": 5,
        "instruksi": "Screenshot hasil akhir"
      },
      {
        "id": "block-uuid-7",
        "block_type": "text",
        "block_order": 6,
        "instruksi": "Kesimpulan dan pembelajaran"
      }
    ]
  }
}
```

**Error Response:**

`404 Not Found`:
```json
{
  "success": false,
  "error": {
    "code": "SERIES_NOT_FOUND",
    "message": "Series tidak ditemukan"
  }
}
```

---

## 8. File Upload (MinIO)

Grafikarsa menggunakan MinIO sebagai object storage untuk menyimpan file (avatar, banner, thumbnail, gambar portfolio). Upload menggunakan **presigned URL** untuk performa dan keamanan optimal.

### Upload Flow

```
┌─────────┐          ┌─────────┐          ┌─────────┐
│ Client  │          │ Backend │          │  MinIO  │
└────┬────┘          └────┬────┘          └────┬────┘
     │                    │                    │
     │ 1. Request presigned URL               │
     │ POST /uploads/presign                  │
     │───────────────────>│                    │
     │                    │                    │
     │                    │ 2. Generate        │
     │                    │ presigned URL      │
     │                    │───────────────────>│
     │                    │                    │
     │ 3. Return presigned URL + object_key   │
     │<───────────────────│                    │
     │                    │                    │
     │ 4. Upload file directly to MinIO       │
     │ PUT {presigned_url}                    │
     │────────────────────────────────────────>│
     │                    │                    │
     │ 5. Upload success (200)                │
     │<────────────────────────────────────────│
     │                    │                    │
     │ 6. Confirm upload to backend           │
     │ POST /uploads/confirm                  │
     │───────────────────>│                    │
     │                    │                    │
     │                    │ 7. Verify object   │
     │                    │ exists in MinIO    │
     │                    │───────────────────>│
     │                    │                    │
     │                    │ 8. Update database │
     │                    │ (avatar_url, etc)  │
     │                    │                    │
     │ 9. Return final public URL             │
     │<───────────────────│                    │
```

### Supported Upload Types

| Type | Purpose | Max Size | Allowed Types |
|------|---------|----------|---------------|
| `avatar` | User profile picture | 2 MB | jpg, jpeg, png, webp |
| `banner` | User profile banner | 5 MB | jpg, jpeg, png, webp |
| `thumbnail` | Portfolio thumbnail | 5 MB | jpg, jpeg, png, webp |
| `portfolio_image` | Image in content block | 10 MB | jpg, jpeg, png, webp, gif |

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
| upload_type | string | Yes | Tipe upload: `avatar`, `banner`, `thumbnail`, `portfolio_image` |
| filename | string | Yes | Nama file asli (untuk extension) |
| content_type | string | Yes | MIME type file |
| file_size | integer | Yes | Ukuran file dalam bytes |
| portfolio_id | UUID | Conditional | Wajib jika `upload_type` = `thumbnail` atau `portfolio_image` |
| block_id | UUID | Conditional | Wajib jika `upload_type` = `portfolio_image` |

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

**Request Body - Portfolio Image Block:**
```json
{
  "upload_type": "portfolio_image",
  "filename": "screenshot.png",
  "content_type": "image/png",
  "file_size": 1024000,
  "portfolio_id": "880e8400-e29b-41d4-a716-446655440000",
  "block_id": "block-uuid-1"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "upload_id": "upload-uuid-123",
    "presigned_url": "https://minio.grafikarsa.com/grafikarsa/avatars/550e8400.../abc123.jpg?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=...&X-Amz-Date=...&X-Amz-Expires=900&X-Amz-SignedHeaders=host&X-Amz-Signature=...",
    "object_key": "avatars/550e8400-e29b-41d4-a716-446655440000/abc123.jpg",
    "expires_in": 900,
    "method": "PUT",
    "headers": {
      "Content-Type": "image/jpeg"
    }
  }
}
```

| Field | Description |
|-------|-------------|
| upload_id | ID unik untuk tracking upload ini |
| presigned_url | URL untuk upload langsung ke MinIO |
| object_key | Key/path object di MinIO |
| expires_in | Waktu expired presigned URL (detik) |
| method | HTTP method untuk upload (PUT) |
| headers | Headers yang harus disertakan saat upload |

**Error Responses:**

`400 Bad Request` - File size exceeds limit:
```json
{
  "success": false,
  "error": {
    "code": "FILE_TOO_LARGE",
    "message": "Ukuran file melebihi batas maksimal",
    "details": [
      {
        "field": "file_size",
        "message": "Ukuran file avatar maksimal 2MB, file Anda 5MB"
      }
    ]
  }
}
```

`400 Bad Request` - Invalid content type:
```json
{
  "success": false,
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
  "success": false,
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
    method: presignedData.method, // 'PUT'
    headers: presignedData.headers,
    body: file
  });
  
  if (!response.ok) {
    throw new Error('Upload to MinIO failed');
  }
  
  return true;
}
```

**cURL Example:**
```bash
curl -X PUT \
  -H "Content-Type: image/jpeg" \
  --data-binary @profile.jpg \
  "https://minio.grafikarsa.com/grafikarsa/avatars/550e8400.../abc123.jpg?X-Amz-Algorithm=..."
```

**MinIO Response:**
- `200 OK` - Upload berhasil
- `403 Forbidden` - Presigned URL expired atau invalid
- `400 Bad Request` - Content-Type tidak sesuai

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
  "success": true,
  "data": {
    "type": "avatar",
    "url": "https://cdn.grafikarsa.com/avatars/550e8400-e29b-41d4-a716-446655440000/abc123.jpg",
    "object_key": "avatars/550e8400-e29b-41d4-a716-446655440000/abc123.jpg"
  },
  "message": "Avatar berhasil diperbarui"
}
```

**Success Response (200) - Portfolio Thumbnail:**
```json
{
  "success": true,
  "data": {
    "type": "thumbnail",
    "url": "https://cdn.grafikarsa.com/thumbnails/880e8400.../abc123.jpg",
    "portfolio_id": "880e8400-e29b-41d4-a716-446655440000",
    "object_key": "thumbnails/880e8400-e29b-41d4-a716-446655440000/abc123.jpg"
  },
  "message": "Thumbnail portfolio berhasil diperbarui"
}
```

**Success Response (200) - Portfolio Image Block:**
```json
{
  "success": true,
  "data": {
    "type": "portfolio_image",
    "url": "https://cdn.grafikarsa.com/portfolio-images/880e8400.../abc123.jpg",
    "portfolio_id": "880e8400-e29b-41d4-a716-446655440000",
    "block_id": "block-uuid-1",
    "object_key": "portfolio-images/880e8400-e29b-41d4-a716-446655440000/abc123.jpg"
  },
  "message": "Gambar berhasil diupload"
}
```

**Error Responses:**

`404 Not Found` - Upload ID tidak ditemukan:
```json
{
  "success": false,
  "error": {
    "code": "UPLOAD_NOT_FOUND",
    "message": "Upload tidak ditemukan atau sudah expired"
  }
}
```

`400 Bad Request` - Object tidak ada di MinIO:
```json
{
  "success": false,
  "error": {
    "code": "OBJECT_NOT_FOUND",
    "message": "File tidak ditemukan di storage. Pastikan upload ke MinIO berhasil."
  }
}
```

`400 Bad Request` - Upload sudah dikonfirmasi:
```json
{
  "success": false,
  "error": {
    "code": "UPLOAD_ALREADY_CONFIRMED",
    "message": "Upload ini sudah dikonfirmasi sebelumnya"
  }
}
```

---

### DELETE /uploads/{object_key}

Hapus file dari MinIO (untuk admin atau cleanup).

**Authentication:** Required (owner atau admin)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| object_key | string | Object key di MinIO (URL encoded) |

**Example:**
```
DELETE /uploads/avatars%2F550e8400-e29b-41d4-a716-446655440000%2Fabc123.jpg
```

**Success Response (200):**
```json
{
  "success": true,
  "message": "File berhasil dihapus"
}
```

---

### GET /uploads/presign-view

Generate presigned URL untuk view/download file private (jika diperlukan).

**Authentication:** Requ

**Query Parameters:**
| Parameter | Type | Descrip
|-----------|------|-------------|
| object_key | string | Object key di MinIO |

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "url": "https://minio.grafikarsa.com/grafikarsa/...?X-Amz-...",
    "expires_in": 3600
  }
}
```

---

### Complete Upload Flow Example

**1. Request presigned URL:**
```javascript
const presignResponse = await api.post('/uploads/presign', {
  upload_type: 'avatar',
  filename: 'my-photo.jpg',
  content_type: 'image/jpeg',
  file_size: file.size
});

const { upload_id, presigned_url, headers } = presignResponse.data.data;
```

**2. Upload to MinIO:**
```javascript
await fetch(presigned_url, {
  method: 'PUT',
  headers: headers,
  body: file
});
```

**3. Confirm upload:**
```javascript
const confirmResponse = await api.post('/uploads/confirm', {
  upload_id: upload_id,
  object_key: presignResponse.data.data.object_key
});

const newAvatarUrl = confirmResponse.data.data.url;
```

---

### MinIO Bucket Structure

```
grafikarsa/
├── avatars/
│   └── {user_id}/
│       └── {uuid}.{ext}
├── banners/
│   └── {user_id}/
│       └── {uuid}.{ext}
├── thumbnails/
│   └── {portfolio_id}/
│       └── {uuid}.{ext}
└── portfolio-images/
    └── {portfolio_id}/
        └── {uuid}.{ext}
```

---

### CDN URL Mapping

File yang sudah diupload akan diakses melalui CDN:

| MinIO Path | CDN URL |
|------------|---------|
| `avatars/{user_id}/{file}` | `https://cdn.grafikarsa.com/avatars/{user_id}/{file}` |
| `banners/{user_id}/{file}` | `https://cdn.grafikarsa.com/banners/{user_id}/{file}` |
| `thumbnails/{portfolio_id}/{file}` | `https://cdn.grafikarsa.com/thumbnails/{portfolio_id}/{file}` |
| `portfolio-images/{portfolio_id}/{file}` | `https://cdn.grafikarsa.com/portfolio-images/{portfolio_id}/{file}` |

---

## 9. Social (Follow)

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
  "success": true,
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
  "success": false,
  "error": {
    "code": "CANNOT_FOLLOW_SELF",
    "message": "Tidak bisa follow diri sendiri"
  }
}
```

`409 Conflict` - Sudah follow:
```json
{
  "success": false,
  "error": {
    "code": "ALREADY_FOLLOWING",
    "message": "Anda sudah follow user ini"
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
  "success": true,
  "data": {
    "is_following": false,
    "follower_count": 150
  },
  "message": "Berhasil unfollow john_doe"
}
```

**Error Response:**

`400 Bad Request` - Belum follow:
```json
{
  "success": false,
  "error": {
    "code": "NOT_FOLLOWING",
    "message": "Anda belum follow user ini"
  }
}
```

---

## 10. Likes

### POST /portfolios/{id}/like

Like portfolio.

**Authentication:** Required

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "is_liked": true,
    "like_count": 46
  },
  "message": "Portfolio berhasil di-like"
}
```

**Error Response:**

`409 Conflict`:
```json
{
  "success": false,
  "error": {
    "code": "ALREADY_LIKED",
    "message": "Anda sudah like portfolio ini"
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
  "success": true,
  "data": {
    "is_liked": false,
    "like_count": 45
  },
  "message": "Like berhasil dihapus"
}
```

---

## 11. Search

### GET /search/users

Cari user.

**Authentication:** Optional

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| q | string | Query pencarian (nama, username, bio) |
| jurusan_id | UUID | Filter jurusan |
| kelas_id | UUID | Filter kelas |
| role | string | Filter role |
| page | integer | Halaman |
| limit | integer | Jumlah per halaman |

**Success Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "username": "john_doe",
      "nama": "John Doe",
      "avatar_url": "https://cdn.grafikarsa.com/avatars/john.jpg",
      "bio": "Siswa RPL yang suka coding",
      "role": "student",
      "kelas_nama": "XII-RPL-A",
      "jurusan_nama": "Rekayasa Perangkat Lunak"
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
| jurusan_id | UUID | Filter jurusan pembuat |
| kelas_id | UUID | Filter kelas pembuat |
| page | integer | Halaman |
| limit | integer | Jumlah per halaman |

**Success Response (200):** Sama dengan GET /portfolios

---

## 12. Feed

### GET /feed

Timeline portfolio dari user yang di-follow (untuk user yang login).

**Authentication:** Required

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| page | integer | Halaman |
| limit | integer | Jumlah per halaman |

**Success Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "880e8400-e29b-41d4-a716-446655440000",
      "judul": "Website Portfolio Pribadi",
      "slug": "website-portfolio-pribadi",
      "thumbnail_url": "https://cdn.grafikarsa.com/thumbnails/portfolio1.jpg",
      "published_at": "2025-12-09T10:00:00Z",
      "like_count": 45,
      "user": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "username": "john_doe",
        "nama": "John Doe",
        "avatar_url": "https://cdn.grafikarsa.com/avatars/john.jpg",
        "role": "student",
        "kelas_nama": "XII-RPL-A"
      },
      "tags": [
        { "id": "tag-uuid-1", "nama": "Web Development" }
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

## 13. Admin - Jurusan

### GET /admin/jurusan

Daftar semua jurusan.

**Authentication:** Required (admin only)

**Success Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "770e8400-e29b-41d4-a716-446655440000",
      "nama": "Rekayasa Perangkat Lunak",
      "kode": "rpl",
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-01-01T00:00:00Z"
    },
    {
      "id": "770e8400-e29b-41d4-a716-446655440001",
      "nama": "Teknik Komputer dan Jaringan",
      "kode": "tkj",
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-01-01T00:00:00Z"
    }
  ]
}
```

---

### POST /admin/jurusan

Buat jurusan baru.

**Authentication:** Required (admin only)

**Request Body:**
```json
{
  "nama": "Desain Komunikasi Visual",
  "kode": "dkv"
}
```

**Success Response (201):**
```json
{
  "success": true,
  "data": {
    "id": "770e8400-e29b-41d4-a716-446655440002",
    "nama": "Desain Komunikasi Visual",
    "kode": "dkv",
    "created_at": "2025-12-09T10:00:00Z"
  },
  "message": "Jurusan berhasil dibuat"
}
```

**Error Responses:**

`409 Conflict`:
```json
{
  "success": false,
  "error": {
    "code": "DUPLICATE_KODE",
    "message": "Kode jurusan sudah digunakan"
  }
}
```

`422 Unprocessable Entity`:
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validasi gagal",
    "details": [
      {
        "field": "kode",
        "message": "Kode hanya boleh berisi huruf lowercase"
      }
    ]
  }
}
```

---

### PATCH /admin/jurusan/{id}

Update jurusan.

**Authentication:** Required (admin only)

**Request Body:**
```json
{
  "nama": "Desain Komunikasi Visual - Updated"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "id": "770e8400-e29b-41d4-a716-446655440002",
    "nama": "Desain Komunikasi Visual - Updated",
    "kode": "dkv",
    "updated_at": "2025-12-09T11:00:00Z"
  },
  "message": "Jurusan berhasil diperbarui"
}
```

---

### DELETE /admin/jurusan/{id}

Hapus jurusan (soft delete).

**Authentication:** Required (admin only)

**Success Response (200):**
```json
{
  "success": true,
  "message": "Jurusan berhasil dihapus"
}
```

**Error Response:**

`409 Conflict` - Masih ada kelas yang menggunakan:
```json
{
  "success": false,
  "error": {
    "code": "JURUSAN_IN_USE",
    "message": "Jurusan tidak bisa dihapus karena masih digunakan oleh kelas"
  }
}
```

---

## 14. Admin - Tahun Ajaran

### GET /admin/tahun-ajaran

Daftar semua tahun ajaran.

**Authentication:** Required (admin only)

**Success Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "990e8400-e29b-41d4-a716-446655440000",
      "tahun_mulai": 2025,
      "is_active": true,
      "promotion_month": 7,
      "promotion_day": 1,
      "created_at": "2025-01-01T00:00:00Z"
    },
    {
      "id": "990e8400-e29b-41d4-a716-446655440001",
      "tahun_mulai": 2024,
      "is_active": false,
      "promotion_month": 7,
      "promotion_day": 1,
      "created_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

---

### POST /admin/tahun-ajaran

Buat tahun ajaran baru.

**Authentication:** Required (admin only)

**Request Body:**
```json
{
  "tahun_mulai": 2026,
  "is_active": false,
  "promotion_month": 7,
  "promotion_day": 1
}
```

**Success Response (201):**
```json
{
  "success": true,
  "data": {
    "id": "990e8400-e29b-41d4-a716-446655440002",
    "tahun_mulai": 2026,
    "is_active": false,
    "promotion_month": 7,
    "promotion_day": 1,
    "created_at": "2025-12-09T10:00:00Z"
  },
  "message": "Tahun ajaran berhasil dibuat"
}
```

**Error Response:**

`409 Conflict`:
```json
{
  "success": false,
  "error": {
    "code": "DUPLICATE_TAHUN",
    "message": "Tahun ajaran sudah ada"
  }
}
```

---

### PATCH /admin/tahun-ajaran/{id}

Update tahun ajaran.

**Authentication:** Required (admin only)

**Request Body:**
```json
{
  "is_active": true,
  "promotion_month": 6,
  "promotion_day": 15
}
```

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "id": "990e8400-e29b-41d4-a716-446655440002",
    "tahun_mulai": 2026,
    "is_active": true,
    "promotion_month": 6,
    "promotion_day": 15,
    "updated_at": "2025-12-09T11:00:00Z"
  },
  "message": "Tahun ajaran berhasil diperbarui"
}
```

*Note: Jika `is_active` diset `true`, tahun ajaran lain akan otomatis menjadi `false`.*

---

### DELETE /admin/tahun-ajaran/{id}

Hapus tahun ajaran.

**Authentication:** Required (admin only)

**Error Response:**

`409 Conflict`:
```json
{
  "success": false,
  "error": {
    "code": "TAHUN_AJARAN_IN_USE",
    "message": "Tahun ajaran tidak bisa dihapus karena masih digunakan oleh kelas"
  }
}
```

---

## 15. Admin - Kelas

### GET /admin/kelas

Daftar semua kelas.

**Authentication:** Required (admin only)

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| tahun_ajaran_id | UUID | Filter berdasarkan tahun ajaran |
| jurusan_id | UUID | Filter berdasarkan jurusan |
| tingkat | integer | Filter berdasarkan tingkat (10, 11, 12) |

**Success Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "660e8400-e29b-41d4-a716-446655440000",
      "nama": "XII-RPL-A",
      "tingkat": 12,
      "rombel": "A",
      "tahun_ajaran": {
        "id": "990e8400-e29b-41d4-a716-446655440000",
        "tahun_mulai": 2025,
        "is_active": true
      },
      "jurusan": {
        "id": "770e8400-e29b-41d4-a716-446655440000",
        "nama": "Rekayasa Perangkat Lunak",
        "kode": "rpl"
      },
      "student_count": 32,
      "created_at": "2025-01-01T00:00:00Z"
    }
  ],
  "meta": {
    "current_page": 1,
    "per_page": 50,
    "total_pages": 2,
    "total_count": 60
  }
}
```

---

### POST /admin/kelas

Buat kelas baru.

**Authentication:** Required (admin only)

**Request Body:**
```json
{
  "tahun_ajaran_id": "990e8400-e29b-41d4-a716-446655440000",
  "jurusan_id": "770e8400-e29b-41d4-a716-446655440000",
  "tingkat": 10,
  "rombel": "A"
}
```

**Success Response (201):**
```json
{
  "success": true,
  "data": {
    "id": "660e8400-e29b-41d4-a716-446655440001",
    "nama": "X-RPL-A",
    "tingkat": 10,
    "rombel": "A",
    "tahun_ajaran": {
      "id": "990e8400-e29b-41d4-a716-446655440000",
      "tahun_mulai": 2025
    },
    "jurusan": {
      "id": "770e8400-e29b-41d4-a716-446655440000",
      "nama": "Rekayasa Perangkat Lunak",
      "kode": "rpl"
    },
    "created_at": "2025-12-09T10:00:00Z"
  },
  "message": "Kelas berhasil dibuat"
}
```

*Note: Nama kelas (`X-RPL-A`) di-generate otomatis dari tingkat + kode jurusan + rombel.*

**Error Responses:**

`409 Conflict`:
```json
{
  "success": false,
  "error": {
    "code": "DUPLICATE_KELAS",
    "message": "Kelas dengan kombinasi tahun ajaran, jurusan, tingkat, dan rombel yang sama sudah ada"
  }
}
```

`422 Unprocessable Entity`:
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validasi gagal",
    "details": [
      {
        "field": "tingkat",
        "message": "Tingkat harus 10, 11, atau 12"
      },
      {
        "field": "rombel",
        "message": "Rombel harus satu huruf A-Z"
      }
    ]
  }
}
```

---

### PATCH /admin/kelas/{id}

Update kelas.

**Authentication:** Required (admin only)

**Request Body:**
```json
{
  "rombel": "B"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "id": "660e8400-e29b-41d4-a716-446655440001",
    "nama": "X-RPL-B",
    "tingkat": 10,
    "rombel": "B",
    "updated_at": "2025-12-09T11:00:00Z"
  },
  "message": "Kelas berhasil diperbarui"
}
```

---

### DELETE /admin/kelas/{id}

Hapus kelas.

**Authentication:** Required (admin only)

**Error Response:**

`409 Conflict`:
```json
{
  "success": false,
  "error": {
    "code": "KELAS_IN_USE",
    "message": "Kelas tidak bisa dihapus karena masih memiliki siswa"
  }
}
```

---

### GET /admin/kelas/{id}/students

Daftar siswa dalam kelas.

**Authentication:** Required (admin only)

**Success Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "username": "john_doe",
      "nama": "John Doe",
      "nisn": "0098115881",
      "nis": "25491/02000.0411",
      "avatar_url": "https://cdn.grafikarsa.com/avatars/john.jpg"
    }
  ],
  "meta": {
    "total_count": 32
  }
}
```

---

## 16. Admin - Users

### GET /admin/users

Daftar semua user (untuk admin).

**Authentication:** Required (admin only)

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| search | string | Cari nama, username, email |
| role | string | Filter role |
| kelas_id | UUID | Filter kelas |
| jurusan_id | UUID | Filter jurusan |
| is_active | boolean | Filter status aktif |
| page | integer | Halaman |
| limit | integer | Jumlah per halaman |

**Success Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "username": "john_doe",
      "email": "john@example.com",
      "nama": "John Doe",
      "avatar_url": "https://cdn.grafikarsa.com/avatars/john.jpg",
      "role": "student",
      "nisn": "0098115881",
      "nis": "25491/02000.0411",
      "kelas": {
        "id": "660e8400-e29b-41d4-a716-446655440000",
        "nama": "XII-RPL-A"
      },
      "jurusan": {
        "id": "770e8400-e29b-41d4-a716-446655440000",
        "nama": "Rekayasa Perangkat Lunak"
      },
      "tahun_masuk": 2023,
      "tahun_lulus": 2026,
      "is_active": true,
      "last_login_at": "2025-12-09T08:00:00Z",
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

**Authentication:** Required (admin only)

**Request Body:**
```json
{
  "username": "new_student",
  "email": "newstudent@example.com",
  "password": "initialpassword123",
  "nama": "New Student",
  "role": "student",
  "nisn": "0098115882",
  "nis": "25491/02000.0412",
  "kelas_id": "660e8400-e29b-41d4-a716-446655440000",
  "tahun_masuk": 2025
}
```

**Success Response (201):**
```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440001",
    "username": "new_student",
    "email": "newstudent@example.com",
    "nama": "New Student",
    "role": "student",
    "nisn": "0098115882",
    "nis": "25491/02000.0412",
    "kelas": {
      "id": "660e8400-e29b-41d4-a716-446655440000",
      "nama": "XII-RPL-A"
    },
    "tahun_masuk": 2025,
    "is_active": true,
    "created_at": "2025-12-09T10:00:00Z"
  },
  "message": "User berhasil dibuat"
}
```

**Error Responses:**

`409 Conflict`:
```json
{
  "success": false,
  "error": {
    "code": "DUPLICATE_USERNAME",
    "message": "Username sudah digunakan"
  }
}
```

```json
{
  "success": false,
  "error": {
    "code": "DUPLICATE_EMAIL",
    "message": "Email sudah digunakan"
  }
}
```

---

### GET /admin/users/{id}

Detail user.

**Authentication:** Required (admin only)

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "john_doe",
    "email": "john@example.com",
    "nama": "John Doe",
    "bio": "Siswa RPL yang suka coding",
    "avatar_url": "https://cdn.grafikarsa.com/avatars/john.jpg",
    "banner_url": "https://cdn.grafikarsa.com/banners/john.jpg",
    "role": "student",
    "nisn": "0098115881",
    "nis": "25491/02000.0411",
    "kelas": {
      "id": "660e8400-e29b-41d4-a716-446655440000",
      "nama": "XII-RPL-A"
    },
    "jurusan": {
      "id": "770e8400-e29b-41d4-a716-446655440000",
      "nama": "Rekayasa Perangkat Lunak"
    },
    "tahun_masuk": 2023,
    "tahun_lulus": 2026,
    "class_history": [
      {
        "kelas_nama": "X-RPL-A",
        "tahun_ajaran": 2023
      },
      {
        "kelas_nama": "XI-RPL-A",
        "tahun_ajaran": 2024
      },
      {
        "kelas_nama": "XII-RPL-A",
        "tahun_ajaran": 2025
      }
    ],
    "social_links": [
      {
        "platform": "github",
        "url": "https://github.com/johndoe"
      }
    ],
    "is_active": true,
    "last_login_at": "2025-12-09T08:00:00Z",
    "created_at": "2023-07-15T08:00:00Z",
    "updated_at": "2025-12-09T08:00:00Z"
  }
}
```

---

### PATCH /admin/users/{id}

Update user.

**Authentication:** Required (admin only)

**Request Body:**
```json
{
  "nama": "John Doe Updated",
  "role": "alumni",
  "kelas_id": null,
  "tahun_lulus": 2025,
  "is_active": true
}
```

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "nama": "John Doe Updated",
    "role": "alumni",
    "tahun_lulus": 2025,
    "updated_at": "2025-12-09T11:00:00Z"
  },
  "message": "User berhasil diperbarui"
}
```

---

### PATCH /admin/users/{id}/password

Reset password user.

**Authentication:** Required (admin only)

**Request Body:**
```json
{
  "new_password": "newpassword123"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "message": "Password user berhasil direset"
}
```

---

### DELETE /admin/users/{id}

Hapus user (soft delete).

**Authentication:** Required (admin only)

**Success Response (200):**
```json
{
  "success": true,
  "message": "User berhasil dihapus"
}
```

---

### POST /admin/users/{id}/deactivate

Nonaktifkan user.

**Authentication:** Required (admin only)

**Success Response (200):**
```json
{
  "success": true,
  "message": "User berhasil dinonaktifkan"
}
```

---

### POST /admin/users/{id}/activate

Aktifkan user.

**Authentication:** Required (admin only)

**Success Response (200):**
```json
{
  "success": true,
  "message": "User berhasil diaktifkan"
}
```

---

## 17. Admin - Tags

### GET /admin/tags

Daftar semua tags.

**Authentication:** Required (admin only)

**Success Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "tag-uuid-1",
      "nama": "Web Development",
      "portfolio_count": 45,
      "created_at": "2025-01-01T00:00:00Z"
    },
    {
      "id": "tag-uuid-2",
      "nama": "Mobile App",
      "portfolio_count": 32,
      "created_at": "2025-01-01T00:00:00Z"
    }
  ]
}
```

---

### POST /admin/tags

Buat tag baru.

**Authentication:** Required (admin only)

**Request Body:**
```json
{
  "nama": "Machine Learning"
}
```

**Success Response (201):**
```json
{
  "success": true,
  "data": {
    "id": "tag-uuid-11",
    "nama": "Machine Learning",
    "created_at": "2025-12-09T10:00:00Z"
  },
  "message": "Tag berhasil dibuat"
}
```

**Error Response:**

`409 Conflict`:
```json
{
  "success": false,
  "error": {
    "code": "DUPLICATE_TAG",
    "message": "Tag dengan nama tersebut sudah ada"
  }
}
```

---

### PATCH /admin/tags/{id}

Update tag.

**Authentication:** Required (admin only)

**Request Body:**
```json
{
  "nama": "Machine Learning & AI"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "id": "tag-uuid-11",
    "nama": "Machine Learning & AI",
    "updated_at": "2025-12-09T11:00:00Z"
  },
  "message": "Tag berhasil diperbarui"
}
```

---

### DELETE /admin/tags/{id}

Hapus tag.

**Authentication:** Required (admin only)

**Success Response (200):**
```json
{
  "success": true,
  "message": "Tag berhasil dihapus"
}
```

---

## 18. Admin - Series

Series adalah **template portofolio** yang mendefinisikan struktur block konten beserta instruksi untuk setiap block. Admin dapat mengelola series template dengan CRUD operations termasuk block builder.

**Konsep:**
- Series mendefinisikan template blocks dengan instruksi per block
- Relasi portfolio ke series adalah many-to-one (banyak portfolio → 1 series)
- Series nonaktif hanya hilang dari dropdown siswa, tidak mempengaruhi portfolio yang sudah ada
- Setiap series harus memiliki minimal 1 block

### GET /admin/series

Daftar semua series (termasuk yang tidak aktif).

**Authentication:** Required (admin only)

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| search | string | Cari berdasarkan nama series |
| page | integer | Halaman |
| limit | integer | Jumlah per halaman |

**Success Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "series-uuid-1",
      "nama": "PJBL Semester 1 2024",
      "deskripsi": "Template untuk proyek PJBL semester 1",
      "is_active": true,
      "block_count": 7,
      "portfolio_count": 25,
      "created_at": "2025-12-01T10:00:00Z"
    },
    {
      "id": "series-uuid-2",
      "nama": "Ujian Praktik 2024",
      "deskripsi": "Template untuk ujian praktik kejuruan",
      "is_active": false,
      "block_count": 5,
      "portfolio_count": 120,
      "created_at": "2025-12-01T10:00:00Z"
    }
  ],
  "meta": {
    "current_page": 1,
    "per_page": 20,
    "total_pages": 1,
    "total_count": 2
  }
}
```

---

### GET /admin/series/{id}

Detail series dengan blocks template.

**Authentication:** Required (admin only)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | UUID | ID series |

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "id": "series-uuid-1",
    "nama": "PJBL Semester 1 2024",
    "deskripsi": "Template untuk proyek PJBL semester 1 tahun ajaran 2024/2025",
    "is_active": true,
    "blocks": [
      {
        "id": "block-uuid-1",
        "block_type": "text",
        "block_order": 0,
        "instruksi": "Judul dan deskripsi singkat proyek"
      },
      {
        "id": "block-uuid-2",
        "block_type": "image",
        "block_order": 1,
        "instruksi": "Thumbnail/cover proyek (rasio 16:9 recommended)"
      },
      {
        "id": "block-uuid-3",
        "block_type": "youtube",
        "block_order": 2,
        "instruksi": "Video dokumentasi presentasi PJBL"
      }
    ],
    "portfolio_count": 25,
    "created_at": "2025-12-01T10:00:00Z"
  }
}
```

**Error Response:**

`404 Not Found`:
```json
{
  "success": false,
  "error": {
    "code": "SERIES_NOT_FOUND",
    "message": "Series tidak ditemukan"
  }
}
```

---

### POST /admin/series

Buat series baru dengan blocks template.

**Authentication:** Required (admin only)

**Request Body:**
```json
{
  "nama": "PJBL Semester 2 2024",
  "deskripsi": "Template untuk proyek PJBL semester 2 tahun ajaran 2024/2025",
  "is_active": true,
  "blocks": [
    {
      "block_type": "text",
      "instruksi": "Judul dan deskripsi singkat proyek"
    },
    {
      "block_type": "image",
      "instruksi": "Thumbnail/cover proyek (rasio 16:9 recommended)"
    },
    {
      "block_type": "youtube",
      "instruksi": "Video dokumentasi presentasi PJBL"
    },
    {
      "block_type": "text",
      "instruksi": "Tujuan pembuatan proyek"
    },
    {
      "block_type": "text",
      "instruksi": "Proses pengerjaan"
    },
    {
      "block_type": "image",
      "instruksi": "Screenshot hasil akhir"
    },
    {
      "block_type": "text",
      "instruksi": "Kesimpulan dan pembelajaran"
    }
  ]
}
```

**Valid Block Types:**
`text`, `image`, `youtube`, `table`, `button`, `embed`

**Success Response (201):**
```json
{
  "success": true,
  "data": {
    "id": "series-uuid-3",
    "nama": "PJBL Semester 2 2024",
    "deskripsi": "Template untuk proyek PJBL semester 2 tahun ajaran 2024/2025",
    "is_active": true,
    "blocks": [
      {
        "id": "block-uuid-1",
        "block_type": "text",
        "block_order": 0,
        "instruksi": "Judul dan deskripsi singkat proyek"
      },
      {
        "id": "block-uuid-2",
        "block_type": "image",
        "block_order": 1,
        "instruksi": "Thumbnail/cover proyek (rasio 16:9 recommended)"
      }
    ],
    "portfolio_count": 0,
    "created_at": "2025-12-09T10:00:00Z"
  },
  "message": "Series berhasil dibuat"
}
```

**Error Responses:**

`409 Conflict`:
```json
{
  "success": false,
  "error": {
    "code": "DUPLICATE_ERROR",
    "message": "Series dengan nama tersebut sudah ada"
  }
}
```

`422 Unprocessable Entity`:
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validasi gagal",
    "details": [
      {
        "field": "blocks",
        "message": "Series harus memiliki minimal 1 block"
      }
    ]
  }
}
```

---

### PATCH /admin/series/{id}

Update series (nama, deskripsi, status aktif, dan/atau blocks).

**Authentication:** Required (admin only)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | UUID | ID series |

**Request Body:**
```json
{
  "nama": "PJBL Semester 2 2024 - Updated",
  "deskripsi": "Deskripsi yang diperbarui",
  "is_active": false,
  "blocks": [
    {
      "block_type": "text",
      "instruksi": "Judul dan deskripsi singkat proyek (updated)"
    },
    {
      "block_type": "image",
      "instruksi": "Thumbnail/cover proyek"
    }
  ]
}
```

**Note:** Jika `blocks` disertakan, semua blocks lama akan dihapus dan diganti dengan blocks baru. Jika `blocks` tidak disertakan, blocks tidak berubah.

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "id": "series-uuid-3",
    "nama": "PJBL Semester 2 2024 - Updated",
    "deskripsi": "Deskripsi yang diperbarui",
    "is_active": false,
    "blocks": [
      {
        "id": "block-uuid-new-1",
        "block_type": "text",
        "block_order": 0,
        "instruksi": "Judul dan deskripsi singkat proyek (updated)"
      },
      {
        "id": "block-uuid-new-2",
        "block_type": "image",
        "block_order": 1,
        "instruksi": "Thumbnail/cover proyek"
      }
    ],
    "portfolio_count": 0,
    "created_at": "2025-12-09T10:00:00Z"
  },
  "message": "Series berhasil diperbarui"
}
```

---

### DELETE /admin/series/{id}

Hapus series (soft delete). Portfolio yang menggunakan series ini akan tetap ada dengan `series_id = NULL`.

**Authentication:** Required (admin only)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | UUID | ID series |

**Success Response (200):**
```json
{
  "success": true,
  "message": "Series berhasil dihapus"
}
```

---

## 19. Admin - Moderasi

### GET /admin/portfolios/pending

Daftar portfolio yang menunggu review.

**Authentication:** Required (admin only)

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| search | string | Cari judul atau nama user |
| jurusan_id | UUID | Filter jurusan pembuat |
| sort | string | Sorting: `-created_at`, `created_at` |
| page | integer | Halaman |
| limit | integer | Jumlah per halaman |

**Success Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "880e8400-e29b-41d4-a716-446655440000",
      "judul": "Website Portfolio Pribadi",
      "slug": "website-portfolio-pribadi",
      "thumbnail_url": "https://cdn.grafikarsa.com/thumbnails/portfolio1.jpg",
      "status": "pending_review",
      "created_at": "2025-12-08T10:00:00Z",
      "user": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "username": "john_doe",
        "nama": "John Doe",
        "avatar_url": "https://cdn.grafikarsa.com/avatars/john.jpg",
        "kelas_nama": "XII-RPL-A",
        "jurusan_nama": "Rekayasa Perangkat Lunak"
      }
    }
  ],
  "meta": {
    "current_page": 1,
    "per_page": 20,
    "total_pages": 2,
    "total_count": 25
  }
}
```

---

### GET /admin/portfolios/{id}

Detail portfolio untuk review (termasuk content blocks).

**Authentication:** Required (admin only)

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "id": "880e8400-e29b-41d4-a716-446655440000",
    "judul": "Website Portfolio Pribadi",
    "slug": "website-portfolio-pribadi",
    "thumbnail_url": "https://cdn.grafikarsa.com/thumbnails/portfolio1.jpg",
    "status": "pending_review",
    "admin_review_note": null,
    "reviewed_by": null,
    "reviewed_at": null,
    "created_at": "2025-12-08T10:00:00Z",
    "updated_at": "2025-12-08T10:00:00Z",
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "username": "john_doe",
      "nama": "John Doe",
      "avatar_url": "https://cdn.grafikarsa.com/avatars/john.jpg",
      "role": "student",
      "kelas_nama": "XII-RPL-A",
      "jurusan_nama": "Rekayasa Perangkat Lunak"
    },
    "tags": [
      { "id": "tag-uuid-1", "nama": "Web Development" }
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
      }
    ]
  }
}
```

---

### POST /admin/portfolios/{id}/approve

Setujui portfolio (ubah status ke published).

**Authentication:** Required (admin only)

**Request Body (optional):**
```json
{
  "note": "Portfolio bagus, disetujui!"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "id": "880e8400-e29b-41d4-a716-446655440000",
    "status": "published",
    "admin_review_note": "Portfolio bagus, disetujui!",
    "reviewed_at": "2025-12-09T10:00:00Z",
    "published_at": "2025-12-09T10:00:00Z"
  },
  "message": "Portfolio berhasil disetujui dan dipublish"
}
```

---

### POST /admin/portfolios/{id}/reject

Tolak portfolio.

**Authentication:** Required (admin only)

**Request Body:**
```json
{
  "note": "Konten tidak sesuai dengan ketentuan. Mohon perbaiki bagian X dan Y."
}
```

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "id": "880e8400-e29b-41d4-a716-446655440000",
    "status": "rejected",
    "admin_review_note": "Konten tidak sesuai dengan ketentuan. Mohon perbaiki bagian X dan Y.",
    "reviewed_at": "2025-12-09T10:00:00Z"
  },
  "message": "Portfolio ditolak"
}
```

**Error Response:**

`422 Unprocessable Entity`:
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validasi gagal",
    "details": [
      {
        "field": "note",
        "message": "Alasan penolakan wajib diisi"
      }
    ]
  }
}
```

---

### GET /admin/portfolios

Daftar semua portfolio (untuk admin).

**Authentication:** Required (admin only)

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| search | string | Cari judul atau nama user |
| status | string | Filter status |
| user_id | UUID | Filter user |
| jurusan_id | UUID | Filter jurusan |
| page | integer | Halaman |
| limit | integer | Jumlah per halaman |

**Success Response (200):** Sama dengan GET /admin/portfolios/pending tapi dengan semua status.

---

### PATCH /admin/portfolios/{id}

Update portfolio (admin).

**Authentication:** Required (admin only)

**Request Body:**
```json
{
  "judul": "Updated Title",
  "status": "published",
  "tag_ids": ["tag-uuid-1", "tag-uuid-2"]
}
```

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "id": "880e8400-e29b-41d4-a716-446655440000",
    "judul": "Updated Title",
    "status": "published",
    "updated_at": "2025-12-09T11:00:00Z"
  },
  "message": "Portfolio berhasil diperbarui"
}
```

---

### DELETE /admin/portfolios/{id}

Hapus portfolio (admin).

**Authentication:** Required (admin only)

**Success Response (200):**
```json
{
  "success": true,
  "message": "Portfolio berhasil dihapus"
}
```

---

## 20. Admin - Dashboard

### GET /admin/dashboard/stats

Statistik dashboard admin.

**Authentication:** Required (admin only)

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "users": {
      "total": 1000,
      "students": 950,
      "alumni": 45,
      "admins": 5,
      "new_this_month": 50
    },
    "portfolios": {
      "total": 500,
      "published": 400,
      "pending_review": 25,
      "draft": 60,
      "rejected": 10,
      "archived": 5,
      "new_this_month": 30
    },
    "jurusan": {
      "total": 5
    },
    "kelas": {
      "total": 60,
      "active_tahun_ajaran": 20
    }
  }
}
```

---

## 21. Public - Jurusan & Kelas

### GET /jurusan

Daftar jurusan (publik).

**Authentication:** None

**Success Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "770e8400-e29b-41d4-a716-446655440000",
      "nama": "Rekayasa Perangkat Lunak",
      "kode": "rpl"
    },
    {
      "id": "770e8400-e29b-41d4-a716-446655440001",
      "nama": "Teknik Komputer dan Jaringan",
      "kode": "tkj"
    }
  ]
}
```

---

### GET /kelas

Daftar kelas aktif (publik).

**Authentication:** None

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| jurusan_id | UUID | Filter jurusan |
| tingkat | integer | Filter tingkat |

**Success Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "660e8400-e29b-41d4-a716-446655440000",
      "nama": "XII-RPL-A",
      "tingkat": 12,
      "jurusan": {
        "id": "770e8400-e29b-41d4-a716-446655440000",
        "nama": "Rekayasa Perangkat Lunak"
      }
    }
  ]
}
```

---

## 22. Feedback

### POST /feedback

Kirim feedback untuk platform (auth optional).

**Authentication:** Optional (Bearer Token)

**Request Body:**
```json
{
  "kategori": "saran",
  "pesan": "Saran untuk menambahkan fitur dark mode"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| kategori | string | Yes | `bug`, `saran`, atau `lainnya` |
| pesan | string | Yes | Isi feedback (min 10, max 2000 karakter) |

**Success Response (201):**
```json
{
  "success": true,
  "message": "Feedback berhasil dikirim. Terima kasih!",
  "data": {
    "id": "123"
  }
}
```

---

### GET /admin/feedback

Daftar semua feedback (admin only).

**Authentication:** Required (Admin)

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| page | integer | Halaman (default: 1) |
| limit | integer | Jumlah per halaman (default: 20) |
| search | string | Cari di pesan/nama/email |
| kategori | string | Filter: `bug`, `saran`, `lainnya` |
| status | string | Filter: `pending`, `read`, `resolved` |

**Success Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "1",
      "user_id": "123",
      "user": {
        "id": "123",
        "nama": "John Doe",
        "username": "johndoe",
        "avatar_url": "https://..."
      },
      "kategori": "saran",
      "pesan": "Saran untuk menambahkan fitur dark mode",
      "status": "pending",
      "admin_notes": null,
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-01-01T00:00:00Z"
    }
  ],
  "meta": {
    "page": 1,
    "limit": 20,
    "total": 50,
    "total_pages": 3
  }
}
```

---

### GET /admin/feedback/stats

Statistik feedback (admin only).

**Authentication:** Required (Admin)

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "pending": 10,
    "read": 25,
    "resolved": 15,
    "total": 50
  }
}
```

---

### GET /admin/feedback/:id

Detail feedback (admin only).

**Authentication:** Required (Admin)

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "id": "1",
    "user_id": "123",
    "user": { ... },
    "kategori": "bug",
    "pesan": "Ada bug di halaman login",
    "status": "read",
    "admin_notes": "Sudah diperbaiki",
    "created_at": "2025-01-01T00:00:00Z",
    "updated_at": "2025-01-01T00:00:00Z"
  }
}
```

---

### PATCH /admin/feedback/:id

Update status/notes feedback (admin only).

**Authentication:** Required (Admin)

**Request Body:**
```json
{
  "status": "resolved",
  "admin_notes": "Bug sudah diperbaiki di versi 1.2.0"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "message": "Feedback berhasil diperbarui",
  "data": { ... }
}
```

---

### DELETE /admin/feedback/:id

Hapus feedback (admin only).

**Authentication:** Required (Admin)

**Success Response (200):**
```json
{
  "success": true,
  "message": "Feedback berhasil dihapus"
}
```

---

## 23. Admin - Assessment Metrics

Endpoint untuk mengelola metrik penilaian portfolio.

### GET /admin/assessment-metrics

Daftar semua metrik penilaian.

**Authentication:** Required (Admin)

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| active_only | boolean | Hanya tampilkan metrik aktif (default: false) |

**Success Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "nama": "Kreativitas",
      "deskripsi": "Tingkat kreativitas dan orisinalitas karya",
      "urutan": 1,
      "is_active": true,
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-01-01T00:00:00Z"
    },
    {
      "id": "550e8400-e29b-41d4-a716-446655440001",
      "nama": "Teknis",
      "deskripsi": "Kualitas teknis dan implementasi",
      "urutan": 2,
      "is_active": true,
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-01-01T00:00:00Z"
    }
  ]
}
```

---

### POST /admin/assessment-metrics

Buat metrik penilaian baru.

**Authentication:** Required (Admin)

**Request Body:**
```json
{
  "nama": "Kreativitas",
  "deskripsi": "Tingkat kreativitas dan orisinalitas karya"
}
```

**Success Response (201):**
```json
{
  "success": true,
  "message": "Metrik berhasil dibuat",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "nama": "Kreativitas",
    "deskripsi": "Tingkat kreativitas dan orisinalitas karya",
    "urutan": 1,
    "is_active": true,
    "created_at": "2025-01-01T00:00:00Z",
    "updated_at": "2025-01-01T00:00:00Z"
  }
}
```

**Error Responses:**

`400 Bad Request` - Validasi gagal:
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Nama metrik minimal 2 karakter"
  }
}
```

---

### PATCH /admin/assessment-metrics/:id

Update metrik penilaian.

**Authentication:** Required (Admin)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | UUID | ID metrik |

**Request Body:**
```json
{
  "nama": "Kreativitas & Inovasi",
  "deskripsi": "Tingkat kreativitas, orisinalitas, dan inovasi karya",
  "is_active": true
}
```

**Success Response (200):**
```json
{
  "success": true,
  "message": "Metrik berhasil diupdate",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "nama": "Kreativitas & Inovasi",
    "deskripsi": "Tingkat kreativitas, orisinalitas, dan inovasi karya",
    "urutan": 1,
    "is_active": true,
    "created_at": "2025-01-01T00:00:00Z",
    "updated_at": "2025-01-01T00:00:00Z"
  }
}
```

**Error Responses:**

`404 Not Found`:
```json
{
  "success": false,
  "error": {
    "code": "NOT_FOUND",
    "message": "Metrik tidak ditemukan"
  }
}
```

---

### DELETE /admin/assessment-metrics/:id

Hapus metrik penilaian.

**Authentication:** Required (Admin)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | UUID | ID metrik |

**Success Response (200):**
```json
{
  "success": true,
  "message": "Metrik berhasil dihapus"
}
```

**Error Responses:**

`404 Not Found`:
```json
{
  "success": false,
  "error": {
    "code": "NOT_FOUND",
    "message": "Metrik tidak ditemukan"
  }
}
```

---

### PUT /admin/assessment-metrics/reorder

Ubah urutan metrik penilaian.

**Authentication:** Required (Admin)

**Request Body:**
```json
{
  "orders": [
    { "id": "550e8400-e29b-41d4-a716-446655440001", "urutan": 1 },
    { "id": "550e8400-e29b-41d4-a716-446655440000", "urutan": 2 },
    { "id": "550e8400-e29b-41d4-a716-446655440002", "urutan": 3 }
  ]
}
```

**Success Response (200):**
```json
{
  "success": true,
  "message": "Urutan metrik berhasil diubah"
}
```

**Error Responses:**

`400 Bad Request`:
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Orders tidak boleh kosong"
  }
}
```

---

## 24. Admin - Portfolio Assessments

Endpoint untuk mengelola penilaian portfolio.

### GET /admin/assessments

Daftar portfolio untuk penilaian.

**Authentication:** Required (Admin)

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| filter | string | Filter: `all`, `pending`, `assessed` (default: all) |
| search | string | Cari berdasarkan judul portfolio atau nama user |
| page | integer | Halaman (default: 1) |
| limit | integer | Jumlah per halaman (default: 20, max: 100) |

**Success Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "880e8400-e29b-41d4-a716-446655440000",
      "judul": "Website Portfolio Pribadi",
      "slug": "website-portfolio-pribadi",
      "thumbnail_url": "https://cdn.grafikarsa.com/thumbnails/portfolio1.jpg",
      "published_at": "2025-12-01T10:00:00Z",
      "user": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "username": "john_doe",
        "nama": "John Doe",
        "avatar_url": "https://cdn.grafikarsa.com/avatars/john.jpg"
      },
      "assessment": {
        "id": "990e8400-e29b-41d4-a716-446655440000",
        "total_score": 8.5,
        "assessor": {
          "id": "admin-uuid",
          "username": "admin",
          "nama": "Administrator",
          "avatar_url": null
        },
        "assessed_at": "2025-12-05T14:00:00Z"
      }
    },
    {
      "id": "880e8400-e29b-41d4-a716-446655440001",
      "judul": "Desain Logo Keren",
      "slug": "desain-logo-keren",
      "thumbnail_url": "https://cdn.grafikarsa.com/thumbnails/portfolio2.jpg",
      "published_at": "2025-12-02T08:00:00Z",
      "user": {
        "id": "550e8400-e29b-41d4-a716-446655440001",
        "username": "jane_doe",
        "nama": "Jane Doe",
        "avatar_url": "https://cdn.grafikarsa.com/avatars/jane.jpg"
      },
      "assessment": null
    }
  ],
  "meta": {
    "page": 1,
    "limit": 20,
    "total": 50,
    "total_pages": 3
  }
}
```

---

### GET /admin/assessments/:portfolio_id

Detail penilaian portfolio.

**Authentication:** Required (Admin)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| portfolio_id | UUID | ID portfolio |

**Success Response (200) - Portfolio sudah dinilai:**
```json
{
  "success": true,
  "data": {
    "portfolio": {
      "id": "880e8400-e29b-41d4-a716-446655440000",
      "judul": "Website Portfolio Pribadi",
      "slug": "website-portfolio-pribadi",
      "thumbnail_url": "https://cdn.grafikarsa.com/thumbnails/portfolio1.jpg"
    },
    "assessment": {
      "id": "990e8400-e29b-41d4-a716-446655440000",
      "portfolio_id": "880e8400-e29b-41d4-a716-446655440000",
      "assessed_by": "admin-uuid",
      "assessor": {
        "id": "admin-uuid",
        "username": "admin",
        "nama": "Administrator",
        "avatar_url": null
      },
      "scores": [
        {
          "id": "score-uuid-1",
          "metric_id": "metric-uuid-1",
          "metric": {
            "id": "metric-uuid-1",
            "nama": "Kreativitas",
            "deskripsi": "Tingkat kreativitas dan orisinalitas karya",
            "urutan": 1,
            "is_active": true
          },
          "score": 9,
          "comment": "Sangat kreatif dan original",
          "created_at": "2025-12-05T14:00:00Z",
          "updated_at": "2025-12-05T14:00:00Z"
        },
        {
          "id": "score-uuid-2",
          "metric_id": "metric-uuid-2",
          "metric": {
            "id": "metric-uuid-2",
            "nama": "Teknis",
            "deskripsi": "Kualitas teknis dan implementasi",
            "urutan": 2,
            "is_active": true
          },
          "score": 8,
          "comment": "Implementasi baik",
          "created_at": "2025-12-05T14:00:00Z",
          "updated_at": "2025-12-05T14:00:00Z"
        }
      ],
      "final_comment": "Portfolio yang sangat bagus, terus berkarya!",
      "total_score": 8.5,
      "created_at": "2025-12-05T14:00:00Z",
      "updated_at": "2025-12-05T14:00:00Z"
    }
  }
}
```

**Success Response (200) - Portfolio belum dinilai:**
```json
{
  "success": true,
  "data": {
    "portfolio": {
      "id": "880e8400-e29b-41d4-a716-446655440000",
      "judul": "Website Portfolio Pribadi",
      "slug": "website-portfolio-pribadi",
      "thumbnail_url": "https://cdn.grafikarsa.com/thumbnails/portfolio1.jpg"
    },
    "assessment": null
  }
}
```

**Error Responses:**

`404 Not Found`:
```json
{
  "success": false,
  "error": {
    "code": "NOT_FOUND",
    "message": "Portfolio tidak ditemukan"
  }
}
```

`400 Bad Request` - Portfolio belum dipublish:
```json
{
  "success": false,
  "error": {
    "code": "INVALID_STATUS",
    "message": "Portfolio belum dipublish"
  }
}
```

---

### POST /admin/assessments/:portfolio_id

Buat atau update penilaian portfolio.

**Authentication:** Required (Admin)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| portfolio_id | UUID | ID portfolio |

**Request Body:**
```json
{
  "scores": [
    {
      "metric_id": "metric-uuid-1",
      "score": 9,
      "comment": "Sangat kreatif dan original"
    },
    {
      "metric_id": "metric-uuid-2",
      "score": 8,
      "comment": "Implementasi baik"
    }
  ],
  "final_comment": "Portfolio yang sangat bagus, terus berkarya!"
}
```

**Validation Rules:**
- `scores`: Minimal 1 nilai harus diisi
- `score`: Nilai antara 1-10
- `comment`: Opsional, maksimal 500 karakter
- `final_comment`: Opsional, maksimal 2000 karakter

**Success Response (201) - Penilaian baru:**
```json
{
  "success": true,
  "message": "Penilaian berhasil disimpan",
  "data": {
    "id": "990e8400-e29b-41d4-a716-446655440000",
    "portfolio_id": "880e8400-e29b-41d4-a716-446655440000",
    "assessed_by": "admin-uuid",
    "scores": [...],
    "final_comment": "Portfolio yang sangat bagus, terus berkarya!",
    "total_score": 8.5,
    "created_at": "2025-12-05T14:00:00Z",
    "updated_at": "2025-12-05T14:00:00Z"
  }
}
```

**Success Response (200) - Update penilaian:**
```json
{
  "success": true,
  "message": "Penilaian berhasil diupdate",
  "data": { ... }
}
```

**Error Responses:**

`400 Bad Request` - Validasi gagal:
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Nilai harus antara 1-10"
  }
}
```

`400 Bad Request` - Portfolio belum dipublish:
```json
{
  "success": false,
  "error": {
    "code": "INVALID_STATUS",
    "message": "Portfolio belum dipublish"
  }
}
```

---

### DELETE /admin/assessments/:portfolio_id

Hapus penilaian portfolio.

**Authentication:** Required (Admin)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| portfolio_id | UUID | ID portfolio |

**Success Response (200):**
```json
{
  "success": true,
  "message": "Penilaian berhasil dihapus"
}
```

**Error Responses:**

`404 Not Found`:
```json
{
  "success": false,
  "error": {
    "code": "NOT_FOUND",
    "message": "Penilaian tidak ditemukan"
  }
}
```

---

## 25. Notifications

Endpoint untuk mengelola notifikasi user.

### GET /notifications

Daftar notifikasi user yang login.

**Authentication:** Required

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| page | integer | Halaman (default: 1) |
| limit | integer | Jumlah per halaman (default: 20, max: 50) |
| unread_only | boolean | Hanya tampilkan yang belum dibaca (default: false) |

**Success Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "type": "new_follower",
      "title": "Pengikut Baru",
      "message": "@jane_doe mulai mengikuti kamu",
      "data": {
        "follower_id": "uuid",
        "follower_username": "jane_doe",
        "follower_nama": "Jane Doe",
        "follower_avatar": "https://..."
      },
      "is_read": false,
      "read_at": null,
      "created_at": "2025-12-14T10:00:00Z"
    }
  ],
  "meta": {
    "page": 1,
    "limit": 20,
    "total": 50,
    "total_pages": 3,
    "unread_count": 5
  }
}
```

**Notification Types:**
- `new_follower` - Ada user baru yang follow
- `portfolio_liked` - Portfolio di-like
- `portfolio_approved` - Portfolio disetujui admin
- `portfolio_rejected` - Portfolio ditolak admin

---

### GET /notifications/count

Jumlah notifikasi yang belum dibaca.

**Authentication:** Required

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "unread_count": 5
  }
}
```

---

### PATCH /notifications/:id/read

Tandai satu notifikasi sebagai sudah dibaca.

**Authentication:** Required

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | UUID | ID notifikasi |

**Success Response (200):**
```json
{
  "success": true,
  "message": "Notifikasi ditandai sudah dibaca"
}
```

---

### POST /notifications/read-all

Tandai semua notifikasi sebagai sudah dibaca.

**Authentication:** Required

**Success Response (200):**
```json
{
  "success": true,
  "message": "Semua notifikasi ditandai sudah dibaca"
}
```

---

### DELETE /notifications/:id

Hapus satu notifikasi.

**Authentication:** Required

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | UUID | ID notifikasi |

**Success Response (200):**
```json
{
  "success": true,
  "message": "Notifikasi berhasil dihapus"
}
```

---

## 26. Admin - Special Roles

Endpoint untuk mengelola special roles (role custom dengan capabilities tertentu).

### GET /admin/special-roles

Daftar semua special roles.

**Authentication:** Required (Admin)

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| search | string | Cari berdasarkan nama |
| include_inactive | boolean | Sertakan role nonaktif (default: false) |

**Success Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "nama": "Moderator Konten",
      "description": "Dapat memoderasi dan mengelola portfolio",
      "color": "#6366f1",
      "capabilities": ["portfolios", "moderation"],
      "is_active": true,
      "user_count": 3,
      "created_at": "2025-12-14T10:00:00Z"
    }
  ]
}
```

---

### GET /admin/special-roles/active

Daftar special roles yang aktif (untuk UI assignment).

**Authentication:** Required (Admin)

**Success Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "nama": "Moderator Konten",
      "color": "#6366f1",
      "capabilities": ["portfolios", "moderation"],
      "is_active": true
    }
  ]
}
```

---

### GET /admin/special-roles/capabilities

Daftar capabilities yang tersedia.

**Authentication:** Required (Admin)

**Success Response (200):**
```json
{
  "success": true,
  "data": [
    { "key": "dashboard", "label": "Dashboard", "group": "Overview" },
    { "key": "portfolios", "label": "Kelola Portfolios", "group": "Konten" },
    { "key": "moderation", "label": "Moderasi", "group": "Konten" },
    { "key": "assessments", "label": "Penilaian", "group": "Konten" },
    { "key": "assessment_metrics", "label": "Metrik Penilaian", "group": "Konten" },
    { "key": "tags", "label": "Kelola Tags", "group": "Konten" },
    { "key": "series", "label": "Kelola Series", "group": "Konten" },
    { "key": "users", "label": "Kelola Users", "group": "Pengguna" },
    { "key": "special_roles", "label": "Kelola Special Roles", "group": "Pengguna" },
    { "key": "majors", "label": "Kelola Jurusan", "group": "Akademik" },
    { "key": "classes", "label": "Kelola Kelas", "group": "Akademik" },
    { "key": "academic_years", "label": "Tahun Ajaran", "group": "Akademik" },
    { "key": "feedback", "label": "Kelola Feedback", "group": "Lainnya" }
  ]
}
```

---

### POST /admin/special-roles

Buat special role baru.

**Authentication:** Required (Admin)

**Request Body:**
```json
{
  "nama": "Moderator Konten",
  "description": "Dapat memoderasi dan mengelola portfolio",
  "color": "#6366f1",
  "capabilities": ["portfolios", "moderation"],
  "is_active": true
}
```

**Success Response (201):**
```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "nama": "Moderator Konten",
    "description": "Dapat memoderasi dan mengelola portfolio",
    "color": "#6366f1",
    "capabilities": ["portfolios", "moderation"],
    "is_active": true,
    "created_at": "2025-12-14T10:00:00Z"
  },
  "message": "Special role berhasil dibuat"
}
```

---

### GET /admin/special-roles/:id

Detail special role dengan daftar users.

**Authentication:** Required (Admin)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | UUID | ID special role |

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "nama": "Moderator Konten",
    "description": "Dapat memoderasi dan mengelola portfolio",
    "color": "#6366f1",
    "capabilities": ["portfolios", "moderation"],
    "is_active": true,
    "user_count": 3,
    "created_at": "2025-12-14T10:00:00Z",
    "users": [
      {
        "id": "user-uuid",
        "username": "john_doe",
        "nama": "John Doe",
        "avatar_url": "https://...",
        "kelas_nama": "XII-RPL-A",
        "assigned_at": "2025-12-14T10:00:00Z",
        "assigned_by": "admin-uuid"
      }
    ]
  }
}
```

---

### PATCH /admin/special-roles/:id

Update special role.

**Authentication:** Required (Admin)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | UUID | ID special role |

**Request Body:**
```json
{
  "nama": "Super Moderator",
  "description": "Updated description",
  "color": "#3b82f6",
  "capabilities": ["portfolios", "moderation", "tags"],
  "is_active": true
}
```

**Success Response (200):**
```json
{
  "success": true,
  "data": { ... },
  "message": "Special role berhasil diperbarui"
}
```

---

### DELETE /admin/special-roles/:id

Hapus special role (soft delete).

**Authentication:** Required (Admin)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | UUID | ID special role |

**Success Response (200):**
```json
{
  "success": true,
  "message": "Special role berhasil dihapus"
}
```

---

### POST /admin/special-roles/:id/users

Assign users ke special role.

**Authentication:** Required (Admin)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | UUID | ID special role |

**Request Body:**
```json
{
  "user_ids": ["user-uuid-1", "user-uuid-2"]
}
```

**Success Response (200):**
```json
{
  "success": true,
  "message": "Users berhasil di-assign ke role"
}
```

---

### DELETE /admin/special-roles/:id/users/:userId

Hapus user dari special role.

**Authentication:** Required (Admin)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | UUID | ID special role |
| userId | UUID | ID user |

**Success Response (200):**
```json
{
  "success": true,
  "message": "User berhasil dihapus dari role"
}
```

---

### GET /admin/users/:id/special-roles

Daftar special roles yang dimiliki user.

**Authentication:** Required (Admin)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | UUID | ID user |

**Success Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "role-uuid",
      "nama": "Moderator Konten",
      "color": "#6366f1",
      "capabilities": ["portfolios", "moderation"],
      "is_active": true
    }
  ]
}
```

---

### PUT /admin/users/:id/special-roles

Update special roles user (replace all).

**Authentication:** Required (Admin)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | UUID | ID user |

**Request Body:**
```json
{
  "special_role_ids": ["role-uuid-1", "role-uuid-2"]
}
```

**Success Response (200):**
```json
{
  "success": true,
  "message": "Special roles user berhasil diperbarui"
}
```

---

## 27. Changelog

### GET /changelogs

Daftar changelog yang sudah dipublish (publik).

**Authentication:** None

**Query Parameters:**
| Parameter | Type | Description | Default |
|-----------|------|-------------|---------|
| page | integer | Halaman | 1 |
| limit | integer | Jumlah per halaman (max: 50) | 10 |

**Success Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "version": "1.2.0",
      "title": "Fitur Baru: Smart Feed",
      "description": "Penambahan algoritma smart feed untuk rekomendasi portfolio",
      "release_date": "2025-12-15",
      "is_published": true,
      "sections": [
        {
          "id": "section-uuid",
          "category": "added",
          "blocks": [
            {
              "id": "block-uuid",
              "block_type": "text",
              "payload": {
                "content": "<p>Fitur smart feed dengan algoritma rekomendasi</p>"
              }
            }
          ]
        }
      ],
      "contributors": [
        {
          "id": "contrib-uuid",
          "contribution": "Backend Development",
          "user": {
            "id": "user-uuid",
            "username": "john_doe",
            "nama": "John Doe",
            "avatar_url": "https://..."
          }
        }
      ],
      "created_by": {
        "id": "admin-uuid",
        "username": "admin",
        "nama": "Admin",
        "avatar_url": "https://..."
      },
      "created_at": "2025-12-15T10:00:00Z",
      "updated_at": "2025-12-15T10:00:00Z"
    }
  ],
  "meta": {
    "page": 1,
    "limit": 10,
    "total": 5,
    "total_pages": 1
  }
}
```

---

### GET /changelogs/:id

Detail changelog berdasarkan ID.

**Authentication:** None

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | UUID | ID changelog |

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "version": "1.2.0",
    "title": "Fitur Baru: Smart Feed",
    "description": "Penambahan algoritma smart feed",
    "release_date": "2025-12-15",
    "is_published": true,
    "sections": [...],
    "contributors": [...],
    "created_by": {...},
    "created_at": "2025-12-15T10:00:00Z",
    "updated_at": "2025-12-15T10:00:00Z"
  }
}
```

**Error Response:**

`404 Not Found`:
```json
{
  "success": false,
  "error": {
    "code": "NOT_FOUND",
    "message": "Changelog tidak ditemukan"
  }
}
```

---

### GET /changelogs/latest

Ambil changelog terbaru yang sudah dipublish.

**Authentication:** None

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "version": "1.2.0",
    "title": "Fitur Baru: Smart Feed",
    ...
  }
}
```

---

### GET /changelogs/unread-count

Jumlah changelog yang belum dibaca user.

**Authentication:** Required

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "count": 3
  }
}
```

---

### POST /changelogs/:id/mark-read

Tandai changelog sebagai sudah dibaca.

**Authentication:** Required

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | UUID | ID changelog |

**Success Response (200):**
```json
{
  "success": true,
  "message": "Berhasil ditandai sebagai dibaca"
}
```

---

### POST /changelogs/mark-all-read

Tandai semua changelog sebagai sudah dibaca.

**Authentication:** Required

**Success Response (200):**
```json
{
  "success": true,
  "message": "Semua changelog ditandai sebagai dibaca"
}
```

---

### GET /admin/changelogs

Daftar semua changelog (termasuk draft).

**Authentication:** Required (Admin)

**Query Parameters:**
| Parameter | Type | Description | Default |
|-----------|------|-------------|---------|
| page | integer | Halaman | 1 |
| limit | integer | Jumlah per halaman (max: 100) | 20 |
| search | string | Cari berdasarkan version/title | - |

**Success Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "version": "1.2.0",
      "title": "Fitur Baru: Smart Feed",
      "description": "Penambahan algoritma smart feed",
      "release_date": "2025-12-15",
      "is_published": false,
      "categories": ["added", "fixed"],
      "created_at": "2025-12-15T10:00:00Z"
    }
  ],
  "meta": {
    "page": 1,
    "limit": 20,
    "total": 10,
    "total_pages": 1
  }
}
```

---

### GET /admin/changelogs/:id

Detail changelog untuk admin.

**Authentication:** Required (Admin)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | UUID | ID changelog |

**Success Response (200):** Sama dengan GET /changelogs/:id

---

### POST /admin/changelogs

Buat changelog baru.

**Authentication:** Required (Admin)

**Request Body:**
```json
{
  "version": "1.2.0",
  "title": "Fitur Baru: Smart Feed",
  "description": "Penambahan algoritma smart feed untuk rekomendasi portfolio",
  "release_date": "2025-12-15",
  "sections": [
    {
      "category": "added",
      "blocks": [
        {
          "block_type": "text",
          "payload": {
            "content": "<p>Fitur smart feed dengan algoritma rekomendasi</p>"
          }
        }
      ]
    },
    {
      "category": "fixed",
      "blocks": [
        {
          "block_type": "text",
          "payload": {
            "content": "<p>Perbaikan bug pada halaman profil</p>"
          }
        }
      ]
    }
  ],
  "contributors": [
    {
      "user_id": "user-uuid",
      "contribution": "Backend Development"
    }
  ]
}
```

**Validation Rules:**
- `version`: Wajib diisi
- `title`: Wajib diisi
- `sections`: Minimal 1 section, category harus salah satu dari: `added`, `updated`, `removed`, `fixed`
- `contributors`: Minimal 1 contributor

**Success Response (201):**
```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "version": "1.2.0",
    "title": "Fitur Baru: Smart Feed",
    ...
  },
  "message": "Changelog berhasil dibuat"
}
```

**Error Response:**

`400 Bad Request`:
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Minimal 1 section wajib diisi"
  }
}
```

---

### PATCH /admin/changelogs/:id

Update changelog.

**Authentication:** Required (Admin)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | UUID | ID changelog |

**Request Body:**
```json
{
  "version": "1.2.1",
  "title": "Updated Title",
  "description": "Updated description",
  "release_date": "2025-12-16",
  "sections": [...],
  "contributors": [...]
}
```

**Note:** Jika `sections` atau `contributors` disertakan, data lama akan dihapus dan diganti dengan data baru.

**Success Response (200):**
```json
{
  "success": true,
  "data": {...},
  "message": "Changelog berhasil diupdate"
}
```

---

### DELETE /admin/changelogs/:id

Hapus changelog (soft delete).

**Authentication:** Required (Admin)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | UUID | ID changelog |

**Success Response (200):**
```json
{
  "success": true,
  "message": "Changelog berhasil dihapus"
}
```

---

### POST /admin/changelogs/:id/publish

Publish changelog (ubah is_published menjadi true).

**Authentication:** Required (Admin)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | UUID | ID changelog |

**Success Response (200):**
```json
{
  "success": true,
  "message": "Changelog berhasil dipublish"
}
```

**Error Response:**

`400 Bad Request`:
```json
{
  "success": false,
  "error": {
    "code": "ALREADY_PUBLISHED",
    "message": "Changelog sudah dipublish"
  }
}
```

---

### POST /admin/changelogs/:id/unpublish

Unpublish changelog (ubah is_published menjadi false).

*ion:** RequAdmin)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | UUID | ID changelog |

**Success Response (200):**
```json
{
  "success": true,
  "message": "Changelog berhasil di-unpublish"
}
```

**Error Response:**

`400 Bad Request`:
```json
{
  "success": false,
  "error": {
    "code": "NOT_PUBLISHED",
    "message": "Changelog belum dipublish"
  }
}
```

---

## Error Codes Reference

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `VALIDATION_ERROR` | 422 | Input tidak valid |
| `UNAUTHORIZED` | 401 | Token tidak ada atau tidak valid |
| `TOKEN_EXPIRED` | 401 | Token sudah expired |
| `TOKEN_REUSE_DETECTED` | 401 | Refresh token reuse attack detected |
| `FORBIDDEN` | 403 | Tidak punya akses |
| `NOT_FOUND` | 404 | Resource tidak ditemukan |
| `USER_NOT_FOUND` | 404 | User tidak ditemukan |
| `PORTFOLIO_NOT_FO ND` | 404 | Portfolio tidak ditemukan |
| `SESSION_NOT_FOUND` | 404 | Session tidak ditemukan |
| `INVALID_CREDENTIALS` | 401 | Username/password salah |
| `ACCOUNT_DISABLED` | 403 | Akun dinonaktifkan |
| `DUPLICATE_USERNAME` | 409 | Username sudah dipakai |
| `DUPLICATE_EMAIL` | 409 | Email sudah dipakai |
| `USERNAME_TAKEN` | 409 | Username sudah dipakai |
| `ALREADY_FOLLOWING` | 409 | Sudah follow user |
| `NOT_FOLLOWING` | 409 | Belum follow user |
| `ALREADY_LIKED` | 409 | Sudah like portfolio |
| `CANNOT_FOLLOW_SELF` | 400 | Tidak bisa follow diri sendiri |
| `INVALID_PASSWORD` | 400 | Password lama salah |
| `INVALID_FILE` | 422 | File tidak valid |
| `INVALID_STATUS_TRANSITION` | 400 | Transisi status tidak valid |
| `INCOMPLETE_PORTFOLIO` | 422 | Portfolio belum lengkap |
| `DUPLICATE_KODE` | 409 | Kode jurusan sudah ada |
| `DUPLICATE_TAHUN` | 409 | Tahun ajaran sudah ada |
| `DUPLICATE_KELAS` | 409 | Kelas sudah ada |
| `DUPLICATE_TAG` | 409 | Tag sudah ada |
| `JURUSAN_IN_USE` | 409 | Jurusan masih digunakan |
| `TAHUN_AJARAN_IN_USE` | 409 | Tahun ajaran masih digunakan |
| `KELAS_IN_USE` | 409 | Kelas masih digunakan |
| `METRIC_NOT_FOUND` | 404 | Metrik penilaian tidak ditemukan |
| `ASSESSMENT_NOT_FOUND` | 404 | Penilaian tidak ditemukan |
| `INVALID_STATUS` | 400 | Status portfolio tidak valid untuk operasi ini |
| `INVALID_SCORE` | 400 | Nilai tidak valid (harus 1-10) |
| `FETCH_FAILED` | 500 | Gagal mengambil data |
| `CREATE_FAILED` | 500 | Gagal membuat data |
| `UPDATE_FAILED` | 500 | Gagal mengupdate data |
| `DELETE_FAILED` | 500 | Gagal menghapus data |
| `REORDER_FAILED` | 500 | Gagal mengubah urutan |
| `INTERNAL_ERROR` | 500 | Server error |
| `ALREADY_PUBLISHED` | 400 | Changelog sudah dipublish |
| `NOT_PUBLISHED` | 400 | Changelog belum dipublish |
| `MARK_READ_FAILED` | 500 | Gagal menandai sebagai dibaca |
| `PUBLISH_FAILED` | 500 | Gagal mempublish |
| `UNPUBLISH_FAILED` | 500 | Gagal meng-unpublish |

---