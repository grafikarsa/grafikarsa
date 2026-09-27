# 🚀 Deployment — Ubuntu 24 VPS (Single-Domain + Caddy)

Deploy Grafikarsa ke VPS Ubuntu 24.04 dengan **1 domain saja, tanpa Nginx host, tanpa certbot manual**.

---

## 📋 Yang Dibutuhkan

| Item | Keterangan |
|------|-----------|
| VPS Ubuntu 24.04 | Minimal 2GB RAM, 20GB disk |
| Domain (opsional) | Contoh: `grafikarsa.com`. Belum punya? Pakai IP dulu (HTTP) |
| Docker Hub account | Untuk push/pull images |
| GitHub repo | Sudah ada CI/CD workflow |

**Hanya 1 domain, 1 DNS record:**

| Type | Name | Content |
|------|------|---------|
| A | `@` | `IP_VPS` |

Tidak ada `api.*` / `storage.*`. Semuanya nunut domain utama:

- `https://grafikarsa.com/` → frontend
- `https://grafikarsa.com/api/*` → backend (internal only)
- `https://grafikarsa.com/storage/*` → MinIO (publik read)

---

## 📋 Arsitektur

```
                  1 DNS record (A @)
                          │
                          ▼
┌──────────────────────────────────────────────────────┐
│                      UBUNTU VPS                       │
│  ┌────────────────────────────────────────────────┐  │
│  │  DOCKER (docker-compose.deploy.yml)             │  │
│  │                                                 │  │
│  │  internet --80/443--> proxy (Caddy, auto HTTPS) │  │
│  │    /          -> web:3100                       │  │
│  │    /api/*     -> backend:8080 (internal only)   │  │
│  │    /storage/* -> minio:9000  (strip /storage)   │  │
│  │                                                 │  │
│  │  + db (postgres, internal) + redis (internal)   │  │
│  └────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────┘
```

Tidak ada port host selain `80/443` — tidak akan tabrakan dengan aplikasi lain.

---

## Step 1: Persiapan VPS

```bash
ssh root@IP_VPS
apt update && apt upgrade -y

# User deploy (jangan pakai root untuk harian)
adduser deploy
usermod -aG sudo deploy
mkdir -p /home/deploy/.ssh
cp ~/.ssh/authorized_keys /home/deploy/.ssh/
chown -R deploy:deploy /home/deploy/.ssh
chmod 700 /home/deploy/.ssh && chmod 600 /home/deploy/.ssh/authorized_keys

# Firewall: cukup SSH + HTTP + HTTPS
ufw allow OpenSSH
ufw allow 80/tcp
ufw allow 443/tcp
ufw enable && ufw status
```

> Jangan expose 8080/9000/6379 — semua internal via Caddy.

---

## Step 2: Install Docker

```bash
ssh deploy@IP_VPS
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER
exit
ssh deploy@IP_VPS
docker --version && docker compose version
```

---

## Step 3: Setup Project (sekali jalan)

**Opsi A — script otomatis (disarankan):**

```bash
# Dari repo lokal, copy 4 file ke server:
scp docker-compose.deploy.yml Caddyfile .env.example deploy@IP_VPS:~/
scp -r db/ deploy@IP_VPS:~/
scp scripts/server-init.sh deploy@IP_VPS:~/

# Di server:
chmod +x server-init.sh
DOMAIN=grafikarsa.com ./server-init.sh
# Belum ada domain?  DOMAIN=IP_VPS ./server-init.sh   (jalan via HTTP)
```

Script ini: siapkan `/opt/grafikarsa`, generate password + JWT secret random,
turunkan semua URL dari `DOMAIN`, `up -d`, import DB jika kosong, buat bucket public-read.

**Opsi B — manual:**

```bash
sudo mkdir -p /opt/grafikarsa && sudo chown -R deploy:deploy /opt/grafikarsa
cd /opt/grafikarsa
# copy: docker-compose.deploy.yml, Caddyfile, db/, .env.example -> .env
cp .env.example .env && nano .env
```

Isi penting di `.env` (ganti `grafikarsa.com` dengan domain/IP kamu):

```env
DOMAIN=grafikarsa.com

DB_USER=grafikarsa
DB_PASSWORD=<random: openssl rand -base64 32>
DB_NAME=grafikarsa

MINIO_ACCESS_KEY=grafikarsa_admin
MINIO_SECRET_KEY=<random>
MINIO_ENDPOINT=minio:9000
MINIO_PRESIGN_HOST=grafikarsa.com
MINIO_PRESIGN_USE_SSL=true
MINIO_PRESIGN_PATH_PREFIX=/storage
STORAGE_PUBLIC_URL=https://grafikarsa.com/storage/grafikarsa

JWT_ACCESS_SECRET=<random>
JWT_REFRESH_SECRET=<random>

CORS_ORIGINS=https://grafikarsa.com
NEXT_PUBLIC_API_URL=https://grafikarsa.com/api/v1
NEXT_PUBLIC_APP_URL=https://grafikarsa.com
NEXT_PUBLIC_STORAGE_URL=https://grafikarsa.com/storage/grafikarsa

DOCKERHUB_USERNAME=username_kamu
IMAGE_TAG=latest
```

Tanpa domain (`DOMAIN=1.2.3.4`): semua URL pakai `http://`, `MINIO_PRESIGN_USE_SSL=false`.

---

## Step 4: Start

```bash
cd /opt/grafikarsa
docker compose -f docker-compose.deploy.yml pull
docker compose -f docker-compose.deploy.yml up -d
docker compose -f docker-compose.deploy.yml ps

# Cek via proxy (backend/minio TIDAK expose port host, jadi cek lewat proxy):
docker exec grafikarsa-proxy wget -q -O /dev/null http://backend:8080/api/v1/health && echo "API OK"
curl -s http://localhost/api/v1/health -H "Host: grafikarsa.com"
```

Buka di browser: `https://grafikarsa.com/` (atau `http://IP/` jika tanpa domain).
HTTPS terbit otomatis via Let's Encrypt (butuh port 80/443 terbuka + DNS sudah mengarah).

---

## Step 5: DNS (cukup 1 record)

| Type | Name | Content | Proxy |
|------|------|---------|-------|
| A | `@` | `IP_VPS` | DNS-only **atau** Proxied |

- Tanpa Cloudflare: langsung bisa, Caddy urus sertifikat.
- Dengan Cloudflare: boleh proxied, tapi SSL mode harus **Full (Strict)** — JANGAN Flexible (Caddy sudah TLS sendiri).

---

## Step 6: CI/CD (GitHub Actions)

Secrets yang dibutuhkan (URL diturunkan otomatis dari `DOMAIN`):

| Secret | Value |
|--------|-------|
| `DOCKERHUB_USERNAME` / `DOCKERHUB_TOKEN` | Docker Hub |
| `DOMAIN` | `grafikarsa.com` (atau IP) |
| `SSH_HOST` / `SSH_PORT` / `SSH_USERNAME` / `SSH_PRIVATE_KEY` | Akses deploy |

Push ke `main` → build 2 image → copy `docker-compose.deploy.yml,Caddyfile,db/` → `pull + up -d` → health check via proxy.

---

## Migrasi dari setup lama (subdomain)

Jika DB masih berisi `https://storage.grafikarsa.com/grafikarsa/...`:

```bash
# Backup dulu! Lalu:
docker exec -i grafikarsa-db psql -U grafikarsa -d grafikarsa \
  -v old_host='https://storage.grafikarsa.com/grafikarsa' \
  -v new_base='https://grafikarsa.com/storage/grafikarsa' \
  -f db/migrate-to-single-domain.sql
```

Hapus Nginx host lama jika ada:

```bash
sudo rm -f /etc/nginx/sites-enabled/grafikarsa*
sudo systemctl reload nginx   # atau uninstall nginx sekalian
```

---

## Lampiran: shared host (port 80/443 dipakai app lain)

Jika satu VPS menampung banyak aplikasi (Nginx host milik app lain, mis. SIPODI),
jangan rebut port 80/443. Caddy jalan HTTP-only di belakang Nginx host:

`.env` (hanya 3 baris tambahan):

```env
CADDYFILE=./Caddyfile.http-only
PROXY_HTTP_PORT=8081
PROXY_HTTPS_PORT=8443
```

Vhost Nginx host (`/etc/nginx/sites-available/grafikarsa`, lalu `certbot --nginx`):

```nginx
server {
    listen 80;
    server_name grafikarsa.com www.grafikarsa.com;

    location / {
        proxy_pass http://127.0.0.1:8081;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }
}
```

 Alurnya: browser `https://` (Nginx host) → Caddy `:8081` (HTTP) → web/backend/minio.
 `MINIO_PRESIGN_USE_SSL` tetap `true` karena browser melihat HTTPS.

---

## 🔧 Maintenance

```bash
cd /opt/grafikarsa
docker compose -f docker-compose.deploy.yml logs -f            # semua
docker logs grafikarsa-proxy -f --tail=100                     # Caddy (TLS?)
docker logs grafikarsa-backend -f --tail=100
docker compose -f docker-compose.deploy.yml pull && \
docker compose -f docker-compose.deploy.yml up -d && \
docker image prune -f                                          # update manual
docker exec grafikarsa-db pg_dump -U grafikarsa grafikarsa > backup_$(date +%Y%m%d).sql
```

## 🐛 Troubleshooting

| Gejala | Penyebab umum | Fix |
|---|---|---|
| `grafikarsa-web` exit `address already in use` | Port host dipakai app lain | Sudah tidak mungkin: web tidak publish port host lagi. Jika masih terjadi, compose lama — `pull` ulang |
| `EAI_AGAIN api...` di log web | SSR pakai hostname publik | Sudah di-fix via `INTERNAL_API_URL=http://backend:8080/api/v1` — pastikan var ini ada |
| Redis `Restarting (139)` loop | AOF corrupt / OOM | `docker volume rm grafikarsa_redis_data` (isinya cuma cache), compose auto-buat baru |
| HTTPS tidak terbit | Port 80 tertutup / DNS belum mengarah | `ufw allow 80/tcp`, cek `dig +short grafikarsa.com`, `docker logs grafikarsa-proxy` |
| Upload gagal | Caddy `/storage` tidak proxy | `curl -I https://domain/storage/grafikarsa/` harus `Server: MinIO` |

---

## ✅ Checklist

- [ ] UFW: 22, 80, 443
- [ ] Docker terinstall
- [ ] `/opt/grafikarsa` berisi compose + Caddyfile + `.env` + `db/`
- [ ] `docker compose ps` semua Up (proxy, web, backend, db, minio, redis)
- [ ] 1 DNS A record mengarah ke IP
- [ ] `https://DOMAIN/` buka frontend, `/api/v1/health` OK
- [ ] GitHub Secrets (`DOMAIN` dkk) terisi, push `main` hijau
