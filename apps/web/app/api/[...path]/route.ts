import { NextRequest, NextResponse } from "next/server";

// BFF proxy: meneruskan request/response header penting ke browser.
// - request: cookie, authorization, content-type, csrf, range, if-none-match
// - response: set-cookie (WAJIB untuk __Host-rt/__Host-csrf), etag,
//   cache-control, content-range/accept-ranges (media Range), disposition
// Tanpa logika bisnis; error code dari backend tidak diubah.
const METHODS = ["GET", "POST", "PUT", "PATCH", "DELETE"] as const;

const FORWARD_REQ = [
  "cookie",
  "authorization",
  "content-type",
  "x-csrf-token",
  "range",
  "if-none-match",
  "if-modified-since",
  "accept-language",
];

const FORWARD_RES = [
  "content-type",
  "content-length",
  "content-range",
  "accept-ranges",
  "content-disposition",
  "cache-control",
  "etag",
  "last-modified",
  "vary",
];

function allowedOrigin(req: NextRequest): boolean {
  const origin = req.headers.get("origin");
  if (!origin) return true; // same-origin fetch tanpa Origin (browser lama/SSR)
  const appUrl = process.env.NEXT_PUBLIC_APP_URL ?? "http://localhost:3020";
  try {
    return new URL(origin).host === new URL(appUrl).host;
  } catch {
    return false;
  }
}

async function proxy(req: NextRequest, path: string[]) {
  const base = (process.env.INTERNAL_API_URL ?? "http://localhost:8080/v1").replace(/\/$/, "");
  // Normalisasi: base boleh berisi /v1 dan path FE juga berawalan v1/ —
  // jangan sampai jadi /v1/v1. Terima kedua bentuk env.
  const joined = path.join("/");
  const upstreamPath = base.endsWith("/v1") && joined.startsWith("v1/") ? joined.slice(3) : joined;
  const url = `${base}/${upstreamPath}${req.nextUrl.search}`;
  const headers = new Headers();
  for (const h of FORWARD_REQ) {
    const v = req.headers.get(h);
    if (v) headers.set(h, v);
  }
  const reqId = req.headers.get("x-request-id") ?? crypto.randomUUID();
  headers.set("x-request-id", reqId);

  if (["POST", "PUT", "PATCH", "DELETE"].includes(req.method) && !allowedOrigin(req)) {
    return NextResponse.json(
      { success: false, error: { code: "CSRF_INVALID", message: "Origin tidak diizinkan." } },
      { status: 403 },
    );
  }

  const hasBody = req.method !== "GET" && req.method !== "HEAD";
  const res = await fetch(url, {
    method: req.method,
    headers,
    body: hasBody ? req.body : undefined,
    // @ts-expect-error duplex wajib untuk streaming body di Node 18+
    duplex: hasBody ? "half" : undefined,
    redirect: "manual",
  });

  const out = new Headers();
  for (const h of FORWARD_RES) {
    const v = res.headers.get(h);
    if (v) out.set(h, v);
  }
  // Set-Cookie bisa lebih dari satu (rt + csrf) — pakai getSetCookie bila ada.
  const setCookies = res.headers.getSetCookie?.() ?? [];
  for (const c of setCookies) out.append("set-cookie", c);
  out.set("x-request-id", reqId);
  return new NextResponse(res.body, { status: res.status, headers: out });
}

function handler(req: NextRequest, ctx: { params: Promise<{ path: string[] }> }) {
  if (!METHODS.includes(req.method as (typeof METHODS)[number])) {
    return NextResponse.json(
      { success: false, error: { code: "METHOD_NOT_ALLOWED", message: "Metode tidak didukung." } },
      { status: 405 },
    );
  }
  return ctx.params.then((p) => proxy(req, p.path));
}

export const GET = handler;
export const POST = handler;
export const PUT = handler;
export const PATCH = handler;
export const DELETE = handler;
