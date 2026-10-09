# Grafikarsa v2 — API (Fase 0)

Base URL (via BFF): `/api/v1` → `http://api:8080/v1` internal.

## Envelope

Sukses: `{ success: true, data, request_id }`
Gagal: `{ success: false, error: { code, message }, request_id }`

## Endpoint Fase 0

| Method | Path | Auth | Deskripsi |
|---|---|---|---|
| GET | `/health` | — | Liveness |
| GET | `/ready` | — | Readiness: `{ db, dragonfly, storage }` |
| GET | `/v1/health` | — | Liveness versi API |

Kontrak lengkap: `packages/contracts/openapi.yaml`.
