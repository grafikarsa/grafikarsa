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
  const nonce = crypto.randomUUID().replace(/-/g, "");
  const isDev = process.env.NODE_ENV !== "production";
  const csp = [
    "default-src 'self'",
    `script-src 'self' 'nonce-${nonce}' 'strict-dynamic' https://challenges.cloudflare.com${isDev ? " 'unsafe-eval'" : ""}`,
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

  const reqHeaders = new Headers(req.headers);
  reqHeaders.set("x-nonce", nonce);
  reqHeaders.set("Content-Security-Policy", csp);

  const hasSession = Boolean(req.cookies.get("__Host-rt")?.value);
  if (!hasSession && !isPublic(req.nextUrl.pathname)) {
    const url = req.nextUrl.clone();
    url.pathname = "/login";
    url.search = `?next=${encodeURIComponent(req.nextUrl.pathname)}`;
    return NextResponse.redirect(url);
  }

  const res = NextResponse.next({ request: { headers: reqHeaders } });
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
