# 🚀 Deployment — LXC Ubuntu 22 (Single-Domain + Caddy)

Deploy Grafikarsa ke LXC container dengan **1 domain saja, tanpa Nginx host**.

> ⚠️ LXC harus **nesting enabled** agar Docker jalan. Lihat Step 0.

Inti setup SAMA dengan VPS (baca `deployment-ubuntu-vps.md` untuk detail env/Caddy/CI-CD).
Dokumen ini hanya menambah hal yang spesifik LXC: nesting + NAT port-forwarding.

---

## Arsitektur

```
                 1 DNS record (A @ -> IP HOST Proxmox)
                              │
                              ▼ port-forward 80/443 ke LXC
┌──────────────────────────────────────────────────────┐
│  PROXMOX HOST (NAT)                                   │
│  ┌────────────────────────────────────────────────┐  │
│  │  LXC (Ubuntu 22.04, nesting=1)                  │  │
│  │  DOCKER: proxy Caddy (:80/:443)                 │  │
│  │    /          -> web:3100                       │  │
│  │    /api/*     -> backend:8080 (internal only)   │  │
│  │    /storage/* -> minio:9000                     │  │
│  │  + db + redis (internal)                        │  │
│  └────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────┘
```

---

## Step 0: Nesting (WAJIB, di host Proxmox)

```bash
pct set 100 --features nesting=1,keyctl=1
pct restart 100
```

Verifikasi di dalam LXC: `docker run hello-world` harus sukses.

---

## Step 1: Setup LXC

```bash
pct enter 100   # atau ssh root@IP_LXC
apt update && apt upgrade -y
adduser deploy && usermod -aG sudo deploy
apt install -y ufw
ufw allow OpenSSH && ufw allow 80/tcp && ufw allow 443/tcp && ufw enable
```

Install Docker (lihat panduan VPS Step 2 — sama persis).

---

## Step 2: Project (sama seperti VPS)

```bash
scp docker-compose.deploy.yml Caddyfile .env.example deploy@IP_LXC:~/
scp -r db/ deploy@IP_LXC:~/
scp scripts/server-init.sh deploy@IP_LXC:~/
```

Di LXC: `DOMAIN=grafikarsa.com ./server-init.sh` (atau IP publik host jika tanpa domain).
Isi `.env` identik dengan panduan VPS — tidak ada subdomain.

---

## Step 3: Port Forwarding + NAT (jika LXC di belakang Proxmox)

Jalankan di **host Proxmox** (bukan di LXC). Ganti `IP_LXC`:

```bash
LXC_IP="192.168.1.100"

# HTTP/HTTPS ke LXC
iptables -t nat -A PREROUTING -p tcp --dport 80 -j DNAT --to $LXC_IP:80
iptables -t nat -A PREROUTING -p tcp --dport 443 -j DNAT --to $LXC_IP:443

# SSH custom (opsional, agar tidak bentrok host)
iptables -t nat -A PREROUTING -p tcp --dport 2222 -j DNAT --to $LXC_IP:22

iptables -t nat -A POSTROUTING -s $LXC_IP -j MASQUERADE

# Persist
apt install -y iptables-persistent && netfilter-persistent save
```

DNS tetap 1 record: `A @ -> IP PUBLIK HOST` (bukan IP private LXC).
GitHub Secret `SSH_PORT=2222` jika pakai forward SSH di atas.

---

## Step 4: Verifikasi & CI/CD

Sama seperti VPS (Step 4–6 panduan VPS): health check via proxy,
secrets `DOMAIN` + Docker Hub + SSH. Tidak ada Nginx sites yang perlu dibuat.

---

## 🐛 Troubleshooting LXC-specific

```bash
pct config 100 | grep features     # harus nesting=1,keyctl=1
sudo systemctl restart docker      # setelah enable nesting
sudo journalctl -u docker -f       # log docker
df -h                              # disk penuh? docker system prune -a
```

Selebihnya (TLS, redis, upload): lihat tabel troubleshooting panduan VPS.
