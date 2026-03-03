# Deploy Grafikarsa Web ke Cloudflare Pages

## Prerequisites

1. Akun Cloudflare
2. Repository sudah di-push ke GitHub
3. Backend API sudah running dan accessible

---

## Step 1: Connect Repository

1. Buka [Cloudflare Dashboard](https://dash.cloudflare.com)
2. Pilih **Workers & Pages** di sidebar
3. Klik **Create** → **Pages** → **Connect to Git**
4. Authorize GitHub dan pilih repository `grafikarsa-web`

---

## Step 2: Configure Build Settings

| Setting | Value |
|---------|-------|
| Project name | `grafikarsa` |
| Production branch | `main` |
| Framework preset | `Next.js` |
| Build command | `npm run build` |
| Build output directory | `.next` |
| Root directory | `/` (kosong, karena ini polyrepo) |

---

## Step 3: Environment Variables

Di **Settings** → **Environment Variables**, tambahkan:

### Production

```
NEXT_PUBLIC_API_URL = https://api.rafapradana.com/api/v1
NEXT_PUBLIC_APP_URL = https://grafikarsa.rafapradana.com
```

### Preview (optional)

```
NEXT_PUBLIC_API_URL = http://api.rafapradana.com:8080/api/v1
NEXT_PUBLIC_APP_URL = https://preview.grafikarsa.pages.dev
```

---

## Step 4: Custom Domain

1. Di Cloudflare Pages → **Custom domains**
2. Klik **Set up a custom domain**
3. Masukkan: `grafikarsa.rafapradana.com`
4. Cloudflare akan otomatis setup DNS record
5. SSL certificate otomatis provisioned

---

## Step 5: Deploy

Klik **Save and Deploy**. Cloudflare akan:
1. Clone repository
2. Install dependencies
3. Build Next.js app
4. Deploy ke edge network

---

## Deployment Flow

```
Push to GitHub → Cloudflare detects → Build → Deploy to Edge
     │                                              │
     └──────────────────────────────────────────────┘
                    ~2-3 minutes
```

---

## Troubleshooting

### Build Failed: Module not found

Pastikan semua dependencies ada di `package.json` dan tidak ada typo di import paths.

### Environment Variables Not Working

- Pastikan prefix `NEXT_PUBLIC_` untuk variabel yang diakses di client
- Redeploy setelah mengubah environment variables

### API Connection Failed

- Cek CORS di backend sudah include domain Cloudflare
- Pastikan API URL benar (include `/api/v1`)

### Mixed Content Error

Browser block HTTP API dari HTTPS frontend. Solusi:
- Setup HTTPS di backend (recommended)
- Atau gunakan Cloudflare proxy untuk backend juga

---

## Update Deployment

Setiap push ke branch `main` akan trigger auto-deploy.

Manual redeploy:
1. Cloudflare Pages → Deployments
2. Klik **Retry deployment** atau push commit baru

---

## Rollback

1. Cloudflare Pages → Deployments
2. Pilih deployment sebelumnya
3. Klik **Rollback to this deployment**
