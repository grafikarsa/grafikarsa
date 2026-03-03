# 🚀 Deployment — LXC Ubuntu 22 (Cloudflare + Nginx)

Panduan **lengkap step-by-step** untuk deploy Grafikarsa ke LXC container Ubuntu 22.04 dengan Nginx reverse proxy dan Cloudflare DNS/SSL.

> ⚠️ **PENTING**: LXC container harus sudah memiliki **nesting enabled** agar Docker bisa berjalan di dalam LXC. Lihat Step 1.

---

## 📋 Yang Dibutuhkan

| Item | Keterangan |
|------|-----------|
| LXC Container Ubuntu 22.04 | Minimal 2GB RAM, 20GB disk, **nesting enabled** |
| Host Proxmox/LXD | Untuk manage LXC |
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

## 📋 Arsitektur Deployment

```
┌─────────────────────────────────────────────────────────────────┐
│                        CLOUDFLARE                                │
│   grafikarsa.com ──► api.grafikarsa.com ──► storage.grafikarsa.com│
└───────────────────────────┬─────────────────────────────────────┘
                            │ (HTTPS - Proxied)
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                    PROXMOX HOST (NAT)                            │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │              LXC CONTAINER (Ubuntu 22.04)                  │  │
│  │  ┌─────────────────────────────────────────────────────┐  │  │
│  │  │              NGINX (Reverse Proxy Port 80)           │  │  │
│  │  │   /           → grafikarsa-web:3000                 │  │  │
│  │  │   api.        → grafikarsa-backend:8080             │  │  │
│  │  │   storage.    → grafikarsa-minio:9000               │  │  │
│  │  └─────────────────────────────────────────────────────┘  │  │
│  │                          │                                 │  │
│  │  ┌───────────────────────┴───────────────────────────┐    │  │
│  │  │               DOCKER CONTAINERS                    │    │  │
│  │  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────┐  │    │  │
│  │  │  │  Web     │ │ Backend  │ │ Postgres │ │MinIO │  │    │  │
│  │  │  │  :3000   │ │  :8080   │ │  :5432   │ │:9000 │  │    │  │
│  │  │  └──────────┘ └──────────┘ └──────────┘ └──────┘  │    │  │
│  │  └───────────────────────────────────────────────────┘    │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Step 0: Pembuatan LXC Container (CLI)

Jika kamu ingin membuat container via command line di host Proxmox:

```bash
# Login ke Proxmox host via SSH

# Buat container Ubuntu 22.04
# Ganti ID 100, storage local-lvm, dan password sesuai keinginan
pct create 100 local:vztmpl/ubuntu-22.04-standard_22.04-1_amd64.tar.zst \
  --hostname grafikarsa \
  --memory 4096 \
  --cores 2 \
  --rootfs local-lvm:20 \
  --net0 name=eth0,bridge=vmbr0,ip=dhcp \
  --password \
  --unprivileged 1 \
  --features nesting=1,keyctl=1

# Jalankan container
pct start 100

# Masuk ke shell container
pct enter 100
```

---

## Step 1: Konfigurasi LXC Container

### ⚠️ Enable Nesting (WAJIB)

Docker membutuhkan fitur nesting agar bisa berjalan di dalam LXC container. Tanpa ini, Docker **TIDAK AKAN BISA** berjalan.

#### Jika menggunakan Proxmox VE:

**Via Web UI:**
1. Pilih LXC container
2. **Options** → **Features**
3. Centang ✅ **Nesting**
4. Centang ✅ **keyctl** (diperlukan untuk Docker)
5. Klik **OK**
6. Restart container

**Via CLI (di host Proxmox):**
```bash
# Ganti 100 dengan ID container kamu
pct set 100 --features nesting=1,keyctl=1

# Restart container
pct restart 100
```

#### Jika menggunakan LXD:

```bash
# Ganti nama-container dengan nama container kamu
lxc config set nama-container security.nesting true
lxc restart nama-container
```

### 1.1 Verifikasi nesting

```bash
# Masuk ke LXC container
# Proxmox: pct enter 100
# LXD: lxc exec nama-container -- bash

# Verifikasi
cat /proc/1/status | grep NSpid
# Harus menampilkan nested PID info
```

---

## Step 2: Setup LXC Container

### 2.1 Login ke LXC

```bash
# Proxmox
pct enter 100

# Atau SSH jika sudah di-setup
ssh root@LXC_IP_ADDRESS
```

### 2.2 Update system

```bash
apt update && apt upgrade -y
```

### 2.3 Buat user deploy

```bash
# Buat user
adduser deploy
usermod -aG sudo deploy

# Setup SSH (opsional, untuk CI/CD)
mkdir -p /home/deploy/.ssh
# Tambahkan public key kamu
nano /home/deploy/.ssh/authorized_keys
chown -R deploy:deploy /home/deploy/.ssh
chmod 700 /home/deploy/.ssh
chmod 600 /home/deploy/.ssh/authorized_keys
```

### 2.4 Setup Firewall

```bash
apt install -y ufw
ufw allow OpenSSH
ufw allow 80/tcp
ufw allow 443/tcp
ufw enable
ufw status
```

---

## Step 3: Install Docker di LXC

> ⚠️ Docker di LXC memerlukan langkah tambahan dibanding di VPS biasa.

```bash
# Login sebagai deploy
su - deploy

# Install dependencies
sudo apt install -y ca-certificates curl gnupg lsb-release

# Tambah Docker GPG key
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg

# Tambah Docker repository
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# Install Docker
sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# Tambah user ke docker group
sudo usermod -aG docker $USER

# Logout dan login lagi
exit
su - deploy

# Verify
docker --version
docker compose version
docker run hello-world
```

**Jika `docker run hello-world` gagal:**

```bash
# Cek apakah nesting enabled
# Jika tidak, kembali ke Step 1 dan enable nesting

# Alternatif: cek storage driver
sudo docker info | grep "Storage Driver"
# Harus: overlay2

# Jika masalah dengan AppArmor:
sudo systemctl stop apparmor
sudo systemctl disable apparmor
sudo systemctl restart docker
```

---

## Step 4: Install Nginx

```bash
sudo apt install -y nginx
sudo systemctl enable nginx
sudo systemctl start nginx
sudo systemctl status nginx
```

---

## Step 5: Setup Project di LXC

### 5.1 Buat directory

```bash
sudo mkdir -p /opt/grafikarsa
sudo chown -R deploy:deploy /opt/grafikarsa
cd /opt/grafikarsa
```

### 5.2 Clone repo atau copy files

**Opsi 1: Clone repo**
```bash
git clone https://github.com/grafikarsa/grafikarsa.git .
```

**Opsi 2: Copy file minimal**
```bash
# Dari mesin lokal (ganti LXC_IP):
scp docker-compose.deploy.yml deploy@LXC_IP:/opt/grafikarsa/
scp .env.example deploy@LXC_IP:/opt/grafikarsa/
scp -r db/ deploy@LXC_IP:/opt/grafikarsa/
```

### 5.3 Konfigurasi environment production

```bash
cd /opt/grafikarsa
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

## Step 5.5: Deploy Pertama Kali (Manual)

Jika CI/CD belum siap, kamu bisa melakukan push image pertama kali dari komputer lokal kamu.

### 5.5.1 Login ke Docker Hub (Lokal)
```bash
docker login
```

### 5.5.2 Build & Push (Lokal)
Gunakan script build yang sudah disediakan di folder `scripts/`:

**Windows (PowerShell):**
```powershell
.\scripts\build.ps1 -Version "1.0.0"
.\scripts\push.ps1 -Version "1.0.0"
```

**Linux/macOS (Bash):**
```bash
./scripts/build.sh 1.0.0
./scripts/push.sh 1.0.0
```

### 5.5.3 Copy Database Schema ke LXC
```bash
# Ganti HOST_IP dan port SSH yang sesuai
scp -P 22 db/db.sql deploy@HOST_IP:/opt/grafikarsa/db/
```

---

## Step 6: Start Services

```bash
cd /opt/grafikarsa

# Pull images dari Docker Hub
docker compose -f docker-compose.deploy.yml pull

# Start semua services
docker compose -f docker-compose.deploy.yml up -d

# Cek status
docker ps

# Cek logs
docker compose -f docker-compose.deploy.yml logs -f
```

### 6.1 Import database schema (pertama kali saja)

```bash
docker exec -i grafikarsa-db psql -U grafikarsa -d grafikarsa < db/db.sql
```

### 6.2 Setup MinIO bucket (pertama kali saja)

```bash
docker exec -it grafikarsa-minio sh
mc alias set local http://localhost:9000 grafikarsa_admin PASSWORD_MINIO_KAMU
mc mb local/grafikarsa
mc anonymous set download local/grafikarsa
exit
```

---

## Step 7: Konfigurasi Nginx

### 7.1 Frontend (grafikarsa.com)

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

### 7.2 Backend API (api.grafikarsa.com)

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

### 7.3 MinIO Storage (storage.grafikarsa.com)

```bash
sudo nano /etc/nginx/sites-available/grafikarsa-storage
```

```nginx
server {
    listen 80;
    server_name storage.grafikarsa.com;

    client_max_body_size 100M;

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

### 7.4 Enable semua sites

```bash
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

## Step 8: Konfigurasi Cloudflare

### 8.1 Tambahkan domain ke Cloudflare

1. Login ke [Cloudflare Dashboard](https://dash.cloudflare.com)
2. **Add a Site** → masukkan `grafikarsa.com`
3. Pilih plan **Free**
4. Update nameservers di domain registrar

### 8.2 Setup DNS Records

> **Catatan LXC**: Jika LXC ada di belakang NAT (misalnya host Proxmox), gunakan IP **host Proxmox** dan setup port forwarding. Jika LXC punya IP publik langsung, gunakan IP LXC.

| Type | Name | Content | Proxy |
|------|------|---------|-------|
| A | `@` | `SERVER_IP` | ☁️ Proxied |
| A | `www` | `SERVER_IP` | ☁️ Proxied |
| A | `api` | `SERVER_IP` | ☁️ Proxied |
| A | `storage` | `SERVER_IP` | ☁️ Proxied |

### 8.3 Port Forwarding & NAT (Jika LXC di belakang Proxmox Host)

Jika LXC container kamu menggunakan IP private (misal `192.168.1.100`), kamu harus melakukan port forwarding di **Host Proxmox** agar trafik domain dari Internet bisa sampai ke LXC.

#### 8.3.1 Forward Port 80 & 443 (HTTP/HTTPS)

Jalankan ini di **Host Proxmox** (bukan di dalam LXC):

```bash
# Ganti 192.168.1.100 dengan IP private LXC kamu
LXC_IP="192.168.1.100"

# Forward HTTP (80)
iptables -t nat -A PREROUTING -i vmbr0 -p tcp --dport 80 -j DNAT --to $LXC_IP:80

# Forward HTTPS (443)
iptables -t nat -A PREROUTING -i vmbr0 -p tcp --dport 443 -j DNAT --to $LXC_IP:443

# Forward SSH (opsional, misal ke port 2222 agar tidak bentrok dengan host)
iptables -t nat -A PREROUTING -i vmbr0 -p tcp --dport 2222 -j DNAT --to $LXC_IP:22

# Masquerade (agar LXC bisa akses Internet keluar)
iptables -t nat -A POSTROUTING -s $LXC_IP -j MASQUERADE
```

#### 8.3.2 Persist Iptables

Agar rules tidak hilang saat Proxmox reboot:

```bash
apt install -y iptables-persistent
netfilter-persistent save
```

### 8.4 SSL Settings (Cloudflare)

1. **SSL/TLS** → **Overview** → Mode: **Flexible**
2. **SSL/TLS** → **Edge Certificates**:
   - Always Use HTTPS: ✅ **ON**
   - Automatic HTTPS Rewrites: ✅ **ON**
   - Minimum TLS Version: **1.2**

### 8.5 Cache Settings (opsional)

1. **Rules** → **Page Rules**:
   - `storage.grafikarsa.com/*` → Cache Level = **Cache Everything**, Edge Cache TTL = **1 month**

---

## Step 9: Setup CI/CD (GitHub Actions)

### 9.1 GitHub Secrets

Di GitHub repo → **Settings** → **Secrets and variables** → **Actions**:

| Secret | Value |
|--------|-------|
| `DOCKERHUB_USERNAME` | Username Docker Hub |
| `DOCKERHUB_TOKEN` | Docker Hub access token |
| `SSH_HOST` | IP LXC (atau host Proxmox jika NAT) |
| `SSH_PORT` | Port SSH (default: `22`, atau custom jika NAT) |
| `SSH_USERNAME` | `deploy` |
| `SSH_PRIVATE_KEY` | Private key SSH |
| `NEXT_PUBLIC_API_URL` | `https://api.grafikarsa.com/api/v1` |
| `NEXT_PUBLIC_APP_URL` | `https://grafikarsa.com` |
| `NEXT_PUBLIC_STORAGE_URL` | `https://storage.grafikarsa.com/grafikarsa` |

### 9.2 Port Forwarding SSH (jika LXC di belakang NAT)

```bash
# Di host Proxmox — forward port SSH ke LXC
# Contoh: port 2222 di host → port 22 di LXC
iptables -t nat -A PREROUTING -i vmbr0 -p tcp --dport 2222 -j DNAT --to LXC_IP:22

# Persist
netfilter-persistent save
```

Maka di GitHub Secret: `SSH_PORT=2222`.

### 9.3 Generate SSH Key

```bash
# Di mesin lokal
ssh-keygen -t ed25519 -C "github-deploy" -f ~/.ssh/github_deploy

# Copy ke LXC
ssh-copy-id -i ~/.ssh/github_deploy.pub -p 2222 deploy@HOST_IP

# Private key ini yang ditaruh di GitHub Secret SSH_PRIVATE_KEY:
cat ~/.ssh/github_deploy
```

### 9.4 Test deploy

```bash
git add .
git commit -m "chore: setup deployment"
git push origin main
```

---

## Step 10: Verifikasi

```bash
# Di LXC, cek semua running
docker ps

# Cek dari browser
# https://grafikarsa.com        → Frontend
# https://api.grafikarsa.com    → Backend API
# https://storage.grafikarsa.com → File storage

# Cek health
curl https://api.grafikarsa.com/health
```

---

## 🔧 Maintenance

### Update aplikasi

Push ke `main` → CI/CD auto deploy.

Manual:
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

### Logs

```bash
cd /opt/grafikarsa
docker compose -f docker-compose.deploy.yml logs -f
docker logs grafikarsa-backend -f --tail=100
```

---

## 🐛 Troubleshooting LXC-specific

### Docker tidak jalan

```bash
# Pastikan nesting enabled
# Di host Proxmox:
pct config 100 | grep features
# Harus ada: nesting=1,keyctl=1

# Restart Docker
sudo systemctl restart docker

# Cek logs
sudo journalctl -u docker -f
```

### Permission denied saat start container

```bash
# Mungkin AppArmor blocking
sudo systemctl stop apparmor
sudo systemctl disable apparmor
sudo systemctl restart docker
```

### Disk penuh

```bash
# Cek penggunaan disk
df -h

# Bersihkan Docker resources
docker system prune -a --volumes

# Minta admin Proxmox untuk resize disk LXC
```

---

## ✅ Checklist Deployment

- [ ] LXC container dibuat dengan **nesting enabled** dan **keyctl enabled**
- [ ] Ubuntu 22.04 installed di LXC
- [ ] User `deploy` dibuat
- [ ] Firewall (UFW) dikonfigurasi
- [ ] Docker terinstall dan `docker run hello-world` berhasil
- [ ] Nginx terinstall
- [ ] Project files ada di `/opt/grafikarsa/`
- [ ] `.env` dikonfigurasi untuk production
- [ ] Docker services berjalan (`docker ps`)
- [ ] Database schema imported
- [ ] MinIO bucket dibuat
- [ ] Nginx configs aktif (3 sites)
- [ ] Port forwarding setup (jika LXC di belakang NAT)
- [ ] Domain di Cloudflare (4 DNS records)
- [ ] SSL mode = Flexible
- [ ] GitHub Secrets configured
- [ ] CI/CD test deploy berhasil
- [ ] Frontend, API, dan Storage accessible via HTTPS
