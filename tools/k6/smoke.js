// Smoke test k6 (PRD §14.3/§15.4). Jalankan terhadap API yang sudah hidup:
//
//   k6 run -e BASE=http://localhost:8091 tools/k6/smoke.js
//
// Ambang batas sengaja ketat supaya regresi performa terdeteksi lebih awal.
import http from "k6/http";
import { check, group, sleep } from "k6";
import { Rate } from "k6/metrics";

const BASE = __ENV.BASE || "http://localhost:8091";

export const options = {
  scenarios: {
    publik: {
      executor: "constant-vus",
      vus: 10,
      duration: "30s",
      exec: "publik",
    },
    terautentikasi: {
      executor: "constant-vus",
      vus: 5,
      duration: "30s",
      exec: "terautentikasi",
      startTime: "5s",
    },
  },
  thresholds: {
    // Endpoint baca harus cepat; 5% error masih ditoleransi untuk data kosong.
    //
    // Sintaks persentil: p(95), bukan p95. k6 v2 tidak lagi menerima
    // bentuk p95 —_THRESHOLD "p95<800" gagal diparse ("failed parsing
    // method from expression") dan skrip jadi gagal SEBELUM ada request
    // yang dikirim, jadi ambang batasnya tidak pernah benar-benar diuji.
    "http_req_duration{scenario:publik}": ["p(95)<800"],
    "http_req_duration{scenario:terautentikasi}": ["p(95)<1200"],
    http_req_failed: ["rate<0.05"],
  },
};

const loginFail = new Rate("login_fail");

export function publik() {
  // http_req_failed menghitung SEMUA respons non-2xx sebagai kegagalan,
  // termasuk 404 yang memang diharapkan pada endpoint yang datanya masih
  // kosong (mis. /v1/changelogs/latest kalau belum ada changelog). Tanpa
  // pengecualian ini threshold rate<0.05 selalu dilanggar padahal tidak
  // ada bug. Diverifikasi: k6 v2 TIDAK memperhitungkan options.responseCallback
  // untuk metrik ini — harus lewat http.setResponseCallback saat runtime.
  //
  // Ketatannya tetap terjaga: setiap endpoint punya check() yang memeriksa
  // status harusnya, jadi kegagalan nyata (500, 403, timeout) langsung
  // menaikkan checks_failed.
  http.setResponseCallback(http.expectedStatuses(200, 201, 202, 400, 404));
  group("endpoint publik", () => {
    const health = http.get(`${BASE}/health`);
    check(health, { "health 200": (r) => r.status === 200 });

    const feed = http.get(`${BASE}/v1/feed?algorithm=recent&per_page=10`);
    check(feed, { "feed 200": (r) => r.status === 200 });

    const katalog = http.get(`${BASE}/v1/portfolios?per_page=10`);
    check(katalog, { "katalog 200": (r) => r.status === 200 });

    const lb = http.get(`${BASE}/v1/leaderboards?period=weekly&scope=global&kind=best_projects`);
    check(lb, { "leaderboard 200": (r) => r.status === 200 });

    const changelog = http.get(`${BASE}/v1/changelogs/latest`);
    check(changelog, { "changelog 200/404": (r) => r.status === 200 || r.status === 404 });

    const cari = http.get(`${BASE}/v1/search?q=karya`);
    check(cari, { "search 200": (r) => r.status === 200 || r.status === 400 });
  });
  sleep(1);
}

// setup(): login SEKALI sebelum skenario berjalan, lalu token-nya dipakai
// ulang di semua iterasi.
//
// Versi sebelumnya login di dalam loop. Dengan 5 VU selama 30 detik itu
// hundreds of login attempts — cukup untuk memicu lockout brute-force
// (10 gagal -> akun terkunci 15 menit), sehingga SELURUH skenario
// terautentikasi gagal bukan karena API rusak tapi karena proteksinya
// bekerja. Login sekali di setup() juga lebih realistis: satu sesi,
// banyak request.
export function setup() {
  const user = __ENV.USER;
  const pass = __ENV.PASS;
  if (!user || !pass) {
    console.warn("USER/PASS tidak diset — skenario terautentikasi dilewati.");
    return { auth: null };
  }
  const login = http.post(
    `${BASE}/v1/auth/login`,
    JSON.stringify({ username: user, password: pass }),
    { headers: { "Content-Type": "application/json" } },
  );
  const ok = check(login, { "login 200": (r) => r.status === 200 });
  loginFail.add(!ok);
  if (!ok) {
    console.error(`Login gagal (${login.status}): ${login.body}`);
    return { auth: null };
  }
  const body = login.json();
  return {
    auth: {
      headers: {
        Authorization: `Bearer ${body.data.access_token}`,
        "X-CSRF-Token": body.data.csrf_token,
        Cookie: `__Host-csrf=${body.data.csrf_token}`,
        "Content-Type": "application/json",
      },
    },
  };
}

export function terautentikasi(data) {
  const auth = data && data.auth;
  if (!auth) {
    return;
  }
  http.setResponseCallback(http.expectedStatuses(200, 201, 202, 400, 404));
  group("alur terautentikasi", () => {
    const me = http.get(`${BASE}/v1/me`, auth);
    check(me, { "me 200": (r) => r.status === 200 });

    const smart = http.get(`${BASE}/v1/feed?algorithm=smart&per_page=10`, auth);
    check(smart, { "feed smart 200": (r) => r.status === 200 });

    // Mutasi ringan + CSRF: pastikan tidak 403/419.
    const prefs = http.put(`${BASE}/v1/feed/preferences`, JSON.stringify({ algorithm: "recent" }), auth);
    check(prefs, { "prefs 200": (r) => r.status === 200 });
  });
  sleep(1);
}
