# 🚀 Deployment — Ubuntu 24 VPS (Cloudflare + Nginx)

Panduan **lengkap step-by-step** untuk deploy Grafikarsa ke VPS Ubuntu 24.04 dengan Nginx reverse proxy dan Cloudflare DNS/SSL.

---

## 📋 Yang Dibutuhkan

| Item | Keterangan |
|------|-----------|
| VPS Ubuntu 24.04 | Minimal 2GB RAM, 20GB disk |
| Domain | Contoh: `grafikarsa.com` |
| Cloudflare account | Gratis |
| Docker Hub account | Untuk push/pull images |
| GitHub repo | Sudah ada CI/CD workflow |

**Domain yang akan digunakan:**
- `grafikarsa.com` — Frontend
- `api.grafikarsa.com` — Backend API
- `storage.grafikarsa.com` — MinIO (file storage)

> Ganti `grafikarsa.com` dengan domain kamu sendiri di seluruh panduan ini.

---

## Step 1: Setup VPS

### 1.1 SSH ke server

```bash
ssh root@YOUR_SERVER_IP
```

### 1.2 Update system

```bash
apt update && apt upgrade -y
```

### 1.3 Buat user deploy (jangan pakai root untuk production)

```bash
# Buat user
adduser deploy
usermod -aG sudo deploy

# Setup SSH key untuk user deploy
mkdir -p /home/deploy/.ssh
cp ~/.ssh/authorized_keys /home/deploy/.ssh/
chown -R deploy:deploy /home/deploy/.ssh
chmod 700 /home/deploy/.ssh
chmod 600 /home/deploy/.ssh/authorized_keys

# Test login di terminal baru
# ssh deploy@YOUR_SERVER_IP
```

### 1.4 Setup Firewall (UFW)

```bash
ufw allow OpenSSH
ufw allow 80/tcp
ufw allow 443/tcp
ufw enable
ufw status
```

> **Jangan** expose port 8080, 9000, 9001 ke publik. Nginx akan proxy traffic ke sana.

---

## Step 2: Install Docker

```bash
# Login sebagai deploy
ssh deploy@YOUR_SERVER_IP

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Tambah user ke docker group (supaya tidak perlu sudo)
sudo usermod -aG docker $USER

# Logout dan login lagi supaya group berlaku
exit
ssh deploy@YOUR_SERVER_IP

# Verify
docker --version
docker compose version
```

---

## Step 3: Install Nginx

```bash
sudo apt install -y nginx

# Pastikan running
sudo systemctl enable nginx
sudo systemctl start nginx
sudo systemctl status nginx
```

---

## Step 4: Setup Project di Server

### 4.1 Buat directory

```bash
sudo mkdir -p /opt/grafikarsa
sudo chown -R deploy:deploy /opt/grafikarsa
cd /opt/grafikarsa
```

### 4.2 Clone repo (atau copy file yang perlu saja)

**Opsi 1: Clone repo**
```bash
git clone https://github.com/grafikarsa/grafikarsa.git .
```

**Opsi 2: Copy file minimal** (lebih aman, tidak expose kode)
```bash
# Dari mesin lokal:
scp docker-compose.deploy.yml deploy@YOUR_SERVER_IP:/opt/grafikarsa/
scp .env.example deploy@YOUR_SERVER_IP:/opt/grafikarsa/
scp -r db/ deploy@YOUR_SERVER_IP:/opt/grafikarsa/
```

### 4.3 Konfigurasi environment production

```bash
cd /opt/grafikarsa

# Copy template
cp .env.example .env
nano .env
```

**Edit `.env` untuk production:**

```env
# APP
APP_ENV=production
NODE_ENV=production

# DATABASE — password HARUS strong!
DB_HOST=localhost
DB_PORT=5432
DB_USER=grafikarsa
DB_PASSWORD=BUAT_PASSWORD_YANG_KUAT_DISINI
DB_NAME=grafikarsa
DB_SSLMODE=disable

# MINIO — password HARUS strong!
MINIO_ENDPOINT=minio:9000
MINIO_ACCESS_KEY=grafikarsa_admin
MINIO_SECRET_KEY=BUAT_PASSWORD_YANG_KUAT_DISINI
MINIO_BUCKET=grafikarsa
MINIO_USE_SSL=false
MINIO_PRESIGN_HOST=storage.grafikarsa.com
MINIO_PRESIGN_USE_SSL=true
STORAGE_PUBLIC_URL=https://storage.grafikarsa.com/grafikarsa

# JWT — HARUS random dan kuat!
# Generate: openssl rand -base64 32
JWT_ACCESS_SECRET=GENERATE_RANDOM_STRING_DISINI
JWT_REFRESH_SECRET=GENERATE_RANDOM_STRING_LAIN_DISINI
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h

# CORS
CORS_ORIGINS=https://grafikarsa.com,https://www.grafikarsa.com

# NEXT.JS URLs
NEXT_PUBLIC_API_URL=https://api.grafikarsa.com/api/v1
NEXT_PUBLIC_APP_URL=https://grafikarsa.com
NEXT_PUBLIC_STORAGE_URL=https://storage.grafikarsa.com/grafikarsa

# ADMIN
ADMIN_LOGIN_PATH=loginadmin

# DOCKER HUB
DOCKERHUB_USERNAME=your_dockerhub_username
IMAGE_TAG=latest
```

> **Generate password kuat:**
> ```bash
> openssl rand -base64 32
> ```

---

## Step 5: Start Services

```bash
cd /opt/grafikarsa

# Pull images dari Docker Hub (jika CI/CD sudah push)
docker compose -f docker-compose.deploy.yml pull

# Start semua services
docker compose -f docker-compose.deploy.yml up -d

# Cek status
docker ps

# Cek logs
docker compose -f docker-compose.deploy.yml logs -f
```

### 5.1 Import database schema (pertama kali saja)

```bash
docker exec -i grafikarsa-db psql -U grafikarsa -d grafikarsa < db/db.sql
```

### 5.2 Setup MinIO bucket (pertama kali saja)

```bash
docker exec -it grafikarsa-minio sh
mc alias set local http://localhost:9000 grafikarsa_admin PASSWORD_MINIO_KAMU
mc mb local/grafikarsa
mc anonymous set download local/grafikarsa
exit
```

---

## Step 6: Konfigurasi Nginx

### 6.1 Frontend (grafikarsa.com)

```bash
sudo nano /etc/nginx/sites-available/grafikarsa
```

```nginx
server {
    listen 80;
    server_name grafikarsa.com www.grafikarsa.com;

    location / {
        proxy_pass http://localhost:3000;
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

### 6.2 Backend API (api.grafikarsa.com)

```bash
sudo nano /etc/nginx/sites-available/grafikarsa-api
```

```nginx
server {
    listen 80;
    server_name api.grafikarsa.com;

    client_max_body_size 50M;

    location / {
        proxy_pass http://localhost:8080;
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

### 6.3 MinIO Storage (storage.grafikarsa.com)

```bash
sudo nano /etc/nginx/sites-available/grafikarsa-storage
```

```nginx
server {
    listen 80;
    server_name storage.grafikarsa.com;

    client_max_body_size 100M;

    # Public read access to bucket
    location / {
        proxy_pass http://localhost:9000;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 6.4 Enable semua sites

```bash
# Enable configs
sudo ln -s /etc/nginx/sites-available/grafikarsa /etc/nginx/sites-enabled/
sudo ln -s /etc/nginx/sites-available/grafikarsa-api /etc/nginx/sites-enabled/
sudo ln -s /etc/nginx/sites-available/grafikarsa-storage /etc/nginx/sites-enabled/

# Hapus default site
sudo rm -f /etc/nginx/sites-enabled/default

# Test config
sudo nginx -t

# Reload
sudo systemctl reload nginx
```

---

## Step 7: Konfigurasi Cloudflare

### 7.1 Tambahkan domain ke Cloudflare

1. Login ke [Cloudflare Dashboard](https://dash.cloudflare.com)
2. Klik **Add a Site** → masukkan `grafikarsa.com`
3. Pilih plan **Free**
4. Cloudflare akan scan DNS records yang ada
5. Update nameservers di domain registrar kamu ke nameservers yang diberikan Cloudflare

### 7.2 Setup DNS Records

Tambahkan A records berikut:

| Type | Name | Content | Proxy |
|------|------|---------|-------|
| A | `@` | `YOUR_SERVER_IP` | ☁️ Proxied |
| A | `www` | `YOUR_SERVER_IP` | ☁️ Proxied |
| A | `api` | `YOUR_SERVER_IP` | ☁️ Proxied |
| A | `storage` | `YOUR_SERVER_IP` | ☁️ Proxied |

### 7.3 SSL Settings

1. **SSL/TLS** → **Overview** → Mode: **Flexible**

   > Flexible = Cloudflare ↔ server pakai HTTP (port 80), browser ↔ Cloudflare pakai HTTPS. Ini yang paling mudah karena tidak perlu SSL cert di server.

2. **SSL/TLS** → **Edge Certificates**:
   - Always Use HTTPS: ✅ **ON**
   - Automatic HTTPS Rewrites: ✅ **ON**
   - Minimum TLS Version: **1.2**

### 7.4 Cache Settings (untuk storage subdomain)

1. **Rules** → **Page Rules** → Create Page Rule:
   - URL: `storage.grafikarsa.com/*`
   - Setting: Cache Level = **Cache Everything**
   - Edge Cache TTL: **1 month**

---

## Step 8: Setup CI/CD (GitHub Actions)

### 8.1 Tambahkan GitHub Secrets

Di GitHub repo → **Settings** → **Secrets and variables** → **Actions** → **New repository secret**:

| Secret | Value |
|--------|-------|
| `DOCKERHUB_USERNAME` | Username Docker Hub kamu |
| `DOCKERHUB_TOKEN` | Docker Hub access token ([buat di sini](https://hub.docker.com/settings/security)) |
| `SSH_HOST` | IP server VPS |
| `SSH_PORT` | Port SSH (default: `22`) |
| `SSH_USERNAME` | `deploy` |
| `SSH_PRIVATE_KEY` | Isi dari `~/.ssh/id_ed25519` (private key) |
| `NEXT_PUBLIC_API_URL` | `https://api.grafikarsa.com/api/v1` |
| `NEXT_PUBLIC_APP_URL` | `https://grafikarsa.com` |

### 8.2 Generate SSH Key untuk deploy

```bash
# Di mesin lokal
ssh-keygen -t ed25519 -C "github-deploy" -f ~/.ssh/github_deploy

# Copy public key ke server
ssh-copy-id -i ~/.ssh/github_deploy.pub deploy@YOUR_SERVER_IP

# Isi dari private key ini yang ditaruh di GitHub Secret SSH_PRIVATE_KEY:
cat ~/.ssh/github_deploy
```

### 8.3 Test deploy

```bash
git add .
git commit -m "chore: setup deployment"
git push origin main
```

GitHub Actions akan otomatis:
1. Build Docker images (backend + web)
2. Push ke Docker Hub
3. SSH ke server
4. Pull images terbaru
5. Restart containers

---

## Step 9: Verifikasi

```bash
# Di server, cek semua running
docker ps

# Cek dari browser
# https://grafikarsa.com        → Frontend
# https://api.grafikarsa.com    → Backend API
# https://storage.grafikarsa.com → File storage

# Cek health endpoint
curl https://api.grafikarsa.com/health
```

---

## 🔧 Maintenance

### Update aplikasi

Push ke branch `main` → CI/CD otomatis deploy.

Manual update:
```bash
cd /opt/grafikarsa
docker compose -f docker-compose.deploy.yml pull
docker compose -f docker-compose.deploy.yml up -d
docker image prune -f
```

### Backup database

```bash
docker exec grafikarsa-db pg_dump -U grafikarsa grafikarsa > backup_$(date +%Y%m%d).sql
```

### Lihat logs

```bash
cd /opt/grafikarsa
docker compose -f docker-compose.deploy.yml logs -f

# Service tertentu
docker logs grafikarsa-backend -f --tail=100
docker logs grafikarsa-web -f --tail=100
```

### Rollback

```bash
# Gunakan commit hash tertentu
cd /opt/grafikarsa
export IMAGE_TAG=abc1234
docker compose -f docker-compose.deploy.yml pull
docker compose -f docker-compose.deploy.yml up -d
```

---

## ✅ Checklist Deployment

- [ ] VPS Ubuntu 24.04 siap
- [ ] User `deploy` dibuat, SSH key terpasang
- [ ] Firewall (UFW) dikonfigurasi
- [ ] Docker terinstall
- [ ] Nginx terinstall
- [ ] Project files ada di `/opt/grafikarsa/`
- [ ] `.env` dikonfigurasi untuk production
- [ ] Docker services berjalan (`docker ps`)
- [ ] Database schema imported
- [ ] MinIO bucket dibuat
- [ ] Nginx configs aktif (3 sites)
- [ ] Domain di Cloudflare (4 DNS records)
- [ ] SSL mode = Flexible
- [ ] GitHub Secrets configured
- [ ] CI/CD test deploy berhasil
- [ ] Frontend, API, dan Storage accessible via HTTPS
