import { NextRequest, NextResponse } from "next/server";

// Middleware: CSP nonce + auth gate berbasis cookie __Host-rt.
// - Nonce dipakai Next untuk script sendiri (tanpa unsafe-inline).
// - Auth gate hanya cek keberadaan cookie (validasi token tetap di API).
//
// PUBLIC_PATHS hanya memuat rute yang BENAR-BENAR ada. Jalur yang tidak ada
// (mis. /register, /forgot-password) sengaja tidak dicantumkan: mencantumkan
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
  const seg = pathname.split("/").filter(Boolean);
  if (seg.length === 1 && !RESERVED.has(seg[0])) return true;
  if (seg.length === 2 && !RESERVED.has(seg[0])) return true;
  return false;
}

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
