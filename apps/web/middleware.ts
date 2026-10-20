import { NextRequest, NextResponse } from "next/server";

// Middleware: CSP + auth gate berbasis cookie __Host-rt.
// - Auth gate hanya cek keberadaan cookie (validasi token tetap di API).
//
// PUBLIC_PATHS hanya memuat rute yang BENAR-BENAR ada.
//
// Catatan desain: Grafikarsa TIDAK punya pendaftaran mandiri dan TIDAK punya
// reset password oleh user — semua akun dibuat dan di-reset oleh admin.
// Jadi /register, /forgot-password, dan /reset-password bukan "halaman yang
// belum dibuat"; halaman-halaman itu memang tidak boleh ada. Kalau suatu
// saat ditambahkan, keduanya harus tetap berupa halaman privat (butuh login
// admin), bukan publik.
//
// Rute yang tidak ada juga sengaja tidak dicantumkan di sini: mencantumkan
// membuat RESERVED ikut memuat kata-kata itu sehingga username sah seperti
// "register" jadi tertolak tanpa alasan.
const PUBLIC_PATHS = [
  "/",
  "/login",
  "/portfolios",
  "/users",
  "/search",
  "/changelog",
  "/feedback",
  "/leaderboards",
  "/feed",
];

// Kata yang TIDAK boleh diperlakukan sebagai username (jalur privat).
// Hanya path privat yang benar-benar ada di app router.
const RESERVED = new Set(["admin", "me", "settings", "notifications", "api", "_next"]);

function isPublic(pathname: string): boolean {
  if (PUBLIC_PATHS.includes(pathname)) return true;
  for (const p of PUBLIC_PATHS) {
    if (p !== "/" && pathname.startsWith(p + "/")) return true;
  }
  // Profil ({username}) dan karya ({username}/{slug}) — bukan jalur privat.
  //
  // Segmen PERTAMA yang bukan reserved berarti ini rute profil publik, dan
  // seluruh turunannya juga publik. Kedalaman tidak dibatasi: route
  // opengraph-image milik Next.js ada di /{username}/{slug}/opengraph-image
  // dan HARUS bisa diakses tanpa sesi — crawler mesin pencari dan platform
  // sosial mengambilnya tanpa cookie, jadi kalau terkunci, metadata OG-nya
  // tidak pernah terbaca.
  //
  // Yang tetap privat: /admin, /me, /settings, /notifications — semuanya
  // ada di RESERVED.
  const seg = pathname.split("/").filter(Boolean);
  if (seg.length >= 1 && !RESERVED.has(seg[0])) return true;
  return false;
}

// Maintenance mode (PRD §16.1).
//
// Dipakai saat jendela migrasi v1 → v2: aplikasi ditulis freeze supaya
// tidak ada perubahan data baru yang masuk ke v1 selama proses migrasi
// berjalan. Pengunjung melihat halaman "Pindah ke v2", bukan error.
//
// Dua cara menyalakannya:
//
// 1. NEXT_PUBLIC_MAINTENANCE_MODE=true — flag build-time. Ini nama yang
//    dipakai PRD. Mengaktifkannya berarti build ulang + deploy ulang.
//
// 2. Header "x-maintenance-mode: on" — flag runtime, dibaca dari request.
//    Reverse proxy (Caddy/nginx) bisa menambah/melepas header itu lalu
//    reload, sehingga jendela maintenance bisa dibuka & ditutup dalam
//    hitungan detik tanpa build ulang. Untuk go-live, cara kedua jauh lebih
//    aman: kalau ada masalah, flag bisa langsung dimatikan.
//
// Halaman /api/* TIDAK disentuh sama sekali (sudah di-exclude matcher),
// sehingga health check, BFF, dan API tetap hidup selama maintenance.
const MAINTENANCE_HEADER = "x-maintenance-mode";

function maintenanceEnabled(req: NextRequest): boolean {
  const flag = req.headers.get(MAINTENANCE_HEADER);
  if (flag) {
    const v = flag.trim().toLowerCase();
    if (v === "on" || v === "true" || v === "1" || v === "yes") return true;
  }
  return process.env.NEXT_PUBLIC_MAINTENANCE_MODE === "true";
}

// Halaman maintenance bersifat mandiri: HTML inline tanpa JS dan tanpa
// aset dari build. Alasannya: kalau build-nya yang bermasalah, halaman ini
// tetap harus bisa tampil.
const MAINTENANCE_HTML = `<!DOCTYPE html>
<html lang="id"><head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width,initial-scale=1"/>
<meta name="robots" content="noindex"/>
<title>Grafikarsa — Sedang Pindah ke v2</title>
<style>
  :root { color-scheme: light dark; }
  body { margin:0; min-height:100vh; display:grid; place-items:center;
         font-family: ui-sans-serif, system-ui, -apple-system, "Segoe UI", sans-serif;
         background:#fafafa; color:#18181b; }
  @media (prefers-color-scheme: dark) { body { background:#09090b; color:#fafafa; } }
  main { max-width:34rem; padding:2rem; text-align:center; }
  h1 { font-size:1.5rem; margin:0 0 .75rem; letter-spacing:-.01em; }
  p { margin:0 0 1rem; line-height:1.6; opacity:.8; font-size:.95rem; }
  code { background:rgba(127,127,127,.15); padding:.15rem .4rem; border-radius:.3rem; font-size:.85em; }
</style>
</head><body>
<main>
  <h1>Grafikarsa sedang pindah ke v2</h1>
  <p>Kami sedang memindahkan data dan menyempurnakan aplikasinya. DATA ANDA
     TIDAK HILANG — semua karya, akun, dan interaksi tetap aman.</p>
  <p>Kami akan mengumumkan lewat <code>changelog</code> begitu v2 siap dibuka.</p>
  <p>Sampai jumpa.</p>
</main>
</body></html>`;

export function middleware(req: NextRequest) {
  const isDev = process.env.NODE_ENV !== "production";
  // CSP tanpa nonce.
  //
  // Kenapa bukan nonce + 'strict-dynamic' (yang sebelumnya dipakai):
  // Next.js hanya bisa menyuntik nonce ke halaman yang di-render SECARA
  // DINAMIS. Halaman yang di-prerender saat build (landing, /login, dan
  // semua halaman dengan revalidate) tidak punya request header saat
  // build, jadi nonce-nya tidak pernah ada di HTML.
  //
  // Akibatnya: 26 dari 26 tag <script> di halaman app-router TIDAK punya
  // nonce, sementara 'strict-dynamic' membuat 'self' diabaikan ->
  // SEMUA script diblokir -> aplikasi tidak ter-hydrate sama sekali
  // (form login tak pernah muncul, tidak ada interaksi apa pun).
  //
  // Nonce mengharuskan semua halaman jadi dynamic: static generation dan
  // ISR dimatikan, tidak bisa dicache CDN, dan tiap request butuh SSR —
  // biaya yang jauh lebih besar untuk aplikasi yang sengaja memakai
  // revalidate (landing 600s, detail karya 60s, changelog 300s).
  //
  // Pendekatan ini mengikuti panduan resmi Next.js untuk aplikasi yang tidak
  // butuh nonce. 'self' + 'unsafe-inline' mengizinkan script dari origin
  // sendiri dan script inline milik Next.js.
  const csp = [
    "default-src 'self'",
    `script-src 'self' 'unsafe-inline'${isDev ? " 'unsafe-eval'" : ""}`,
    "style-src 'self' 'unsafe-inline'",
    "img-src 'self' data: blob: https:",
    "media-src 'self'",
    "font-src 'self' data:",
    "connect-src 'self' https://challenges.cloudflare.com",
    "frame-src 'self' https://www.youtube-nocookie.com https://www.youtube.com https://www.figma.com https://www.canva.com https://docs.google.com https://drive.google.com https://slides.google.com https://codepen.io https://codesandbox.io",
    "object-src 'none'",
    "base-uri 'self'",
    "form-action 'self'",
    "frame-ancestors 'self'",
  ].join("; ");

  // Maintenance mode dicek SEBELUM auth gate: kalau maintenance aktif,
  // pengunjung tidak perlu login cuma untuk melihat halaman "&quot;Pindah ke v2&quot;".
  if (maintenanceEnabled(req)) {
    const res = new NextResponse(MAINTENANCE_HTML, {
      status: 503,
      headers: { "Content-Type": "text/html; charset=utf-8" },
    });
    res.headers.set("Content-Security-Policy", csp);
    res.headers.set("Retry-After", "600");
    res.headers.set("Cross-Origin-Opener-Policy", "same-origin");
    res.headers.set("Cache-Control", "no-store");
    return res;
  }

  // Auth gate: tanpa cookie sesi, halaman privat dialihkan ke /login.
  // Yang dicek hanya KEHADIRAN cookie __Host-rt — validasi tokennya tetap
  // dilakukan API (middleware tidak bisa memverifikasi JWT tanpa secret).
  //
  // Dua nama dicek: "__Host-rt" (produksi, cookie Secure) dan "rt" (dev).
  // Prefix "__Host-" hanya sah kalau cookie punya flag Secure, sedangkan dev
  // berjalan di http://localhost — jadi di dev API memakai nama polos.
  const hasSession = Boolean(req.cookies.get("__Host-rt")?.value || req.cookies.get("rt")?.value);
  if (!hasSession && !isPublic(req.nextUrl.pathname)) {
    const url = req.nextUrl.clone();
    url.pathname = "/login";
    url.search = `?next=${encodeURIComponent(req.nextUrl.pathname)}`;
    const redirect = NextResponse.redirect(url);
    redirect.headers.set("Content-Security-Policy", csp);
    redirect.headers.set("Cross-Origin-Opener-Policy", "same-origin");
    return redirect;
  }

  const res = NextResponse.next();
  res.headers.set("Content-Security-Policy", csp);
  res.headers.set("Cross-Origin-Opener-Policy", "same-origin");
  res.headers.set("x-request-id", req.headers.get("x-request-id") ?? crypto.randomUUID());
  return res;
}

export const config = {
  matcher: [
    "/((?!api|_next/static|_next/image|favicon.ico|.*\\.(?:png|jpg|jpeg|svg|webp|ico|txt|xml)).*)",
  ],
};
