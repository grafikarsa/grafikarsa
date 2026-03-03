# JWT Implementation Guide - Grafikarsa

## Overview

Grafikarsa menggunakan **dual-token authentication** dengan access token + refresh token untuk keamanan optimal. Implementasi ini mendukung:
- Token rotation untuk mencegah token theft
- Device tracking untuk multi-session management
- Token family untuk deteksi reuse attack
- Secure token storage best practices

---

## Token Architecture

### Access Token
- **Tujuan**: Autentikasi request ke protected endpoints
- **Lifetime**: 15 menit (short-lived)
- **Storage**: Memory (JavaScript variable) - JANGAN di localStorage
- **Format**: JWT signed dengan RS256 atau HS256

### Refresh Token
- **Tujuan**: Mendapatkan access token baru tanpa re-login
- **Lifetime**: 7 hari (configurable)
- **Storage**: HttpOnly cookie dengan Secure flag
- **Format**: Opaque string (random bytes), di-hash sebelum disimpan di DB

---

## Token Payload Structure

### Access Token Claims

```json
{
  "sub": "uuid-user-id",
  "jti": "unique-token-id",
  "role": "student|alumni|admin",
  "iat": 1702123456,
  "exp": 1702124356,
  "iss": "grafikarsa",
  "aud": "grafikarsa-api"
}
```

| Claim | Deskripsi |
|-------|-----------|
| `sub` | User ID (UUID) |
| `jti` | JWT ID untuk blacklisting |
| `role` | Role user untuk authorization |
| `iat` | Issued at timestamp |
| `exp` | Expiration timestamp |
| `iss` | Issuer identifier |
| `aud` | Audience identifier |

---

## Authentication Flow

### 1. Login Flow

```
┌─────────┐          ┌─────────┐          ┌──────────┐
│ Client  │          │   API   │          │    DB    │
└────┬────┘          └────┬────┘          └────┬─────┘
     │                    │                    │
     │ POST /auth/login   │                    │
     │ {username,password}│                    │
     │───────────────────>│                    │
     │                    │                    │
     │                    │ Verify credentials │
     │                    │───────────────────>│
     │                    │                    │
     │                    │ Generate tokens    │
     │                    │                    │
     │                    │ Store refresh_token│
     │                    │ (hashed + family)  │
     │                    │───────────────────>│
     │                    │                    │
     │ Set-Cookie:        │                    │
     │ refresh_token      │                    │
     │ (HttpOnly,Secure)  │                    │
     │<───────────────────│                    │
     │                    │                    │
     │ {access_token,     │                    │
     │  expires_in}       │                    │
     │<───────────────────│                    │
```

### 2. Token Refresh Flow (dengan Rotation)

```
┌─────────┐          ┌─────────┐          ┌──────────┐
│ Client  │          │   API   │          │    DB    │
└────┬────┘          └────┬────┘          └────┬─────┘
     │                    │                    │
     │ POST /auth/refresh │                    │
     │ Cookie:refresh_tok │                    │
     │───────────────────>│                    │
     │                    │                    │
     │                    │ Lookup token hash  │
     │                    │───────────────────>│
     │                    │                    │
     │                    │ Validate:          │
     │                    │ - not revoked      │
     │                    │ - not expired      │
     │                    │ - family valid     │
     │                    │                    │
     │                    │ Revoke old token   │
     │                    │───────────────────>│
     │                    │                    │
     │                    │ Create new refresh │
     │                    │ (same family_id)   │
     │                    │───────────────────>│
     │                    │                    │
     │ Set-Cookie:        │                    │
     │ NEW refresh_token  │                    │
     │<───────────────────│                    │
     │                    │                    │
     │ {access_token,     │                    │
     │  expires_in}       │                    │
     │<───────────────────│                    │
```

### 3. Token Reuse Attack Detection

Jika refresh token yang sudah di-rotate digunakan lagi:

```
┌─────────┐          ┌─────────┐          ┌──────────┐
│Attacker │          │   API   │          │    DB    │
└────┬────┘          └────┬────┘          └────┬─────┘
     │                    │                    │
     │ POST /auth/refresh │                    │
     │ (stolen old token) │                    │
     │───────────────────>│                    │
     │                    │                    │
     │                    │ Lookup token       │
     │                    │───────────────────>│
     │                    │                    │
     │                    │ Token found but    │
     │                    │ is_revoked = true  │
     │                    │                    │
     │                    │ ⚠️ REUSE DETECTED! │
     │                    │                    │
     │                    │ Revoke ALL tokens  │
     │                    │ in this family     │
     │                    │───────────────────>│
     │                    │                    │
     │ 401 Unauthorized   │                    │
     │ "Token reuse       │                    │
     │  detected"         │                    │
     │<───────────────────│                    │
```

---

## API Endpoints

### POST /api/auth/login

Login user dan dapatkan tokens.

**Request:**
```json
{
  "username": "john_doe",
  "password": "securepassword123"
}
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "token_type": "Bearer",
    "expires_in": 900
  }
}
```

**Headers:**
```
Set-Cookie: refresh_token=abc123...; HttpOnly; Secure; SameSite=Strict; Path=/api/auth; Max-Age=604800
```

---

### POST /api/auth/refresh

Refresh access token (token rotation).

**Request:**
- Cookie `refresh_token` dikirim otomatis

**Response (200):**
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "token_type": "Bearer",
    "expires_in": 900
  }
}
```

**Response (401) - Token Reuse Detected:**
```json
{
  "success": false,
  "error": {
    "code": "TOKEN_REUSE_DETECTED",
    "message": "Security alert: token reuse detected. All sessions terminated."
  }
}
```

---

### POST /api/auth/logout

Logout dan revoke current session.

**Request:**
- Header: `Authorization: Bearer <access_token>`
- Cookie: `refresh_token`

**Response (200):**
```json
{
  "success": true,
  "message": "Logged out successfully"
}
```

---

### POST /api/auth/logout-all

Logout dari semua device/session.

**Request:**
- Header: `Authorization: Bearer <access_token>`

**Response (200):**
```json
{
  "success": true,
  "message": "Logged out from all devices",
  "data": {
    "sessions_terminated": 3
  }
}
```

---

### GET /api/auth/sessions

List semua active sessions user.

**Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "session-uuid",
      "device_info": {
        "user_agent": "Mozilla/5.0...",
        "device_type": "desktop"
      },
      "ip_address": "192.168.1.1",
      "created_at": "2024-12-09T10:00:00Z",
      "last_used_at": "2024-12-09T14:30:00Z",
      "is_current": true
    }
  ]
}
```

---

### DELETE /api/auth/sessions/:id

Revoke specific session.

**Response (200):**
```json
{
  "success": true,
  "message": "Session terminated"
}
```

---

## Implementation Code Examples

### Token Generation (Backend)

```typescript
// lib/auth/tokens.ts
import jwt from 'jsonwebtoken';
import crypto from 'crypto';
import { db } from '@/lib/db';

const ACCESS_TOKEN_SECRET = process.env.JWT_ACCESS_SECRET!;
const ACCESS_TOKEN_EXPIRY = '15m';
const REFRESH_TOKEN_EXPIRY_DAYS = 7;

interface TokenPayload {
  sub: string;
  role: string;
  jti: string;
}

// Generate Access Token
export function generateAccessToken(userId: string, role: string): string {
  const jti = crypto.randomUUID();
  
  return jwt.sign(
    { sub: userId, role, jti },
    ACCESS_TOKEN_SECRET,
    {
      expiresIn: ACCESS_TOKEN_EXPIRY,
      issuer: 'grafikarsa',
      audience: 'grafikarsa-api'
    }
  );
}

// Generate Refresh Token (opaque)
export function generateRefreshToken(): string {
  return crypto.randomBytes(32).toString('base64url');
}

// Hash refresh token untuk storage
export function hashToken(token: string): string {
  return crypto.createHash('sha256').update(token).digest('hex');
}

// Store refresh token di database
export async function storeRefreshToken(
  userId: string,
  token: string,
  familyId: string,
  deviceInfo?: object,
  ipAddress?: string
) {
  const tokenHash = hashToken(token);
  const expiresAt = new Date();
  expiresAt.setDate(expiresAt.getDate() + REFRESH_TOKEN_EXPIRY_DAYS);

  await db.refreshToken.create({
    data: {
      user_id: userId,
      token_hash: tokenHash,
      family_id: familyId,
      device_info: deviceInfo,
      ip_address: ipAddress,
      expires_at: expiresAt
    }
  });
}

// Verify dan rotate refresh token
export async function rotateRefreshToken(oldToken: string) {
  const tokenHash = hashToken(oldToken);
  
  const storedToken = await db.refreshToken.findUnique({
    where: { token_hash: tokenHash }
  });

  if (!storedToken) {
    return { valid: false, error: 'TOKEN_NOT_FOUND' };
  }

  // Check if already revoked (potential reuse attack!)
  if (storedToken.is_revoked) {
    // Revoke entire token family
    await db.refreshToken.updateMany({
      where: { family_id: storedToken.family_id },
      data: { 
        is_revoked: true, 
        revoked_at: new Date(),
        revoked_reason: 'token_reuse_detected'
      }
    });
    return { valid: false, error: 'TOKEN_REUSE_DETECTED' };
  }

  // Check expiration
  if (new Date() > storedToken.expires_at) {
    return { valid: false, error: 'TOKEN_EXPIRED' };
  }

  // Revoke old token
  await db.refreshToken.update({
    where: { id: storedToken.id },
    data: { 
      is_revoked: true, 
      revoked_at: new Date(),
      revoked_reason: 'rotated'
    }
  });

  // Generate new refresh token (same family)
  const newToken = generateRefreshToken();
  await storeRefreshToken(
    storedToken.user_id,
    newToken,
    storedToken.family_id, // Keep same family
    storedToken.device_info,
    storedToken.ip_address
  );

  return { 
    valid: true, 
    userId: storedToken.user_id,
    newRefreshToken: newToken 
  };
}
```

### Auth Middleware

```typescript
// middleware/auth.ts
import { NextRequest, NextResponse } from 'next/server';
import jwt from 'jsonwebtoken';
import { db } from '@/lib/db';

export async function authMiddleware(req: NextRequest) {
  const authHeader = req.headers.get('authorization');
  
  if (!authHeader?.startsWith('Bearer ')) {
    return NextResponse.json(
      { success: false, error: { code: 'UNAUTHORIZED', message: 'Missing token' } },
      { status: 401 }
    );
  }

  const token = authHeader.substring(7);

  try {
    const payload = jwt.verify(token, process.env.JWT_ACCESS_SECRET!, {
      issuer: 'grafikarsa',
      audience: 'grafikarsa-api'
    }) as { sub: string; role: string; jti: string };

    // Check if token is blacklisted
    const blacklisted = await db.tokenBlacklist.findUnique({
      where: { jti: payload.jti }
    });

    if (blacklisted) {
      return NextResponse.json(
        { success: false, error: { code: 'TOKEN_REVOKED', message: 'Token has been revoked' } },
        { status: 401 }
      );
    }

    // Attach user to request
    return { userId: payload.sub, role: payload.role };
    
  } catch (error) {
    if (error instanceof jwt.TokenExpiredError) {
      return NextResponse.json(
        { success: false, error: { code: 'TOKEN_EXPIRED', message: 'Token expired' } },
        { status: 401 }
      );
    }
    return NextResponse.json(
      { success: false, error: { code: 'INVALID_TOKEN', message: 'Invalid token' } },
      { status: 401 }
    );
  }
}
```

### Cookie Configuration

```typescript
// lib/auth/cookies.ts
import { ResponseCookie } from 'next/dist/compiled/@edge-runtime/cookies';

export const REFRESH_TOKEN_COOKIE_OPTIONS: Partial<ResponseCookie> = {
  httpOnly: true,
  secure: process.env.NODE_ENV === 'production',
  sameSite: 'strict',
  path: '/api/auth',
  maxAge: 7 * 24 * 60 * 60 // 7 days in seconds
};

export function setRefreshTokenCookie(response: Response, token: string) {
  response.headers.append(
    'Set-Cookie',
    `refresh_token=${token}; HttpOnly; Secure; SameSite=Strict; Path=/api/auth; Max-Age=${7 * 24 * 60 * 60}`
  );
}

export function clearRefreshTokenCookie(response: Response) {
  response.headers.append(
    'Set-Cookie',
    'refresh_token=; HttpOnly; Secure; SameSite=Strict; Path=/api/auth; Max-Age=0'
  );
}
```

---

## Frontend Token Management

### Token Storage Strategy

```typescript
// stores/auth.ts
import { create } from 'zustand';

interface AuthState {
  accessToken: string | null;
  isAuthenticated: boolean;
  setAccessToken: (token: string | null) => void;
  logout: () => void;
}

// Access token HANYA di memory, bukan localStorage!
export const useAuthStore = create<AuthState>((set) => ({
  accessToken: null,
  isAuthenticated: false,
  
  setAccessToken: (token) => set({ 
    accessToken: token, 
    isAuthenticated: !!token 
  }),
  
  logout: () => set({ 
    accessToken: null, 
    isAuthenticated: false 
  })
}));
```

### Auto Refresh dengan Axios Interceptor

```typescript
// lib/api/client.ts
import axios from 'axios';
import { useAuthStore } from '@/stores/auth';

const api = axios.create({
  baseURL: '/api',
  withCredentials: true // Penting untuk cookies
});

let isRefreshing = false;
let failedQueue: Array<{
  resolve: (token: string) => void;
  reject: (error: any) => void;
}> = [];

const processQueue = (error: any, token: string | null = null) => {
  failedQueue.forEach((prom) => {
    if (error) {
      prom.reject(error);
    } else {
      prom.resolve(token!);
    }
  });
  failedQueue = [];
};

// Request interceptor - attach access token
api.interceptors.request.use((config) => {
  const token = useAuthStore.getState().accessToken;
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Response interceptor - handle 401 & auto refresh
api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;

    if (error.response?.status === 401 && !originalRequest._retry) {
      if (isRefreshing) {
        // Queue request while refreshing
        return new Promise((resolve, reject) => {
          failedQueue.push({ resolve, reject });
        }).then((token) => {
          originalRequest.headers.Authorization = `Bearer ${token}`;
          return api(originalRequest);
        });
      }

      originalRequest._retry = true;
      isRefreshing = true;

      try {
        const { data } = await axios.post('/api/auth/refresh', {}, {
          withCredentials: true
        });
        
        const newToken = data.data.access_token;
        useAuthStore.getState().setAccessToken(newToken);
        
        processQueue(null, newToken);
        
        originalRequest.headers.Authorization = `Bearer ${newToken}`;
        return api(originalRequest);
        
      } catch (refreshError) {
        processQueue(refreshError, null);
        useAuthStore.getState().logout();
        window.location.href = '/login';
        return Promise.reject(refreshError);
      } finally {
        isRefreshing = false;
      }
    }

    return Promise.reject(error);
  }
);

export default api;
```

---

## Security Checklist

### Token Security
- [x] Access token short-lived (15 menit)
- [x] Refresh token di HttpOnly cookie
- [x] Refresh token di-hash sebelum disimpan
- [x] Token rotation setiap refresh
- [x] Token family tracking untuk reuse detection
- [x] Blacklist untuk access token revocation

### Cookie Security
- [x] `HttpOnly` - Tidak bisa diakses JavaScript
- [x] `Secure` - Hanya HTTPS (production)
- [x] `SameSite=Strict` - CSRF protection
- [x] `Path=/api/auth` - Scope terbatas

### Additional Security
- [x] Device/IP tracking per session
- [x] Logout all devices feature
- [x] Automatic expired token cleanup
- [x] Rate limiting pada auth endpoints
- [x] Password hashing dengan bcrypt

---

## Environment Variables

```env
# JWT Configuration
JWT_ACCESS_SECRET=your-256-bit-secret-key-here
JWT_REFRESH_SECRET=another-256-bit-secret-key

# Token Expiry
ACCESS_TOKEN_EXPIRY=15m
REFRESH_TOKEN_EXPIRY_DAYS=7

# Cookie
COOKIE_DOMAIN=grafikarsa.com
```

---

## Database Cleanup (Cron Job)

Jalankan secara berkala untuk membersihkan expired tokens:

```sql
-- Cleanup expired tokens (jalankan daily)
SELECT cleanup_expired_tokens();
```

Atau via aplikasi:

```typescript
// cron/cleanup-tokens.ts
import { db } from '@/lib/db';

export async function cleanupExpiredTokens() {
  const result = await db.$executeRaw`
    DELETE FROM refresh_tokens WHERE expires_at < NOW();
    DELETE FROM token_blacklist WHERE expires_at < NOW();
  `;
  console.log(`Cleaned up expired tokens`);
}
```
