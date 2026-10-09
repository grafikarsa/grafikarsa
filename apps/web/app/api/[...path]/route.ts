import { NextRequest, NextResponse } from "next/server";

// BFF proxy tipis: meneruskan method/query/body ke API internal.
// Tanpa logika bisnis. Kontrak error dari backend diteruskan apa adanya.
const METHODS = ["GET", "POST", "PUT", "PATCH", "DELETE"] as const;

async function proxy(req: NextRequest, path: string[]) {
  const base = process.env.INTERNAL_API_URL ?? "http://localhost:8080/v1";
  const url = `${base}/${path.join("/")}${req.nextUrl.search}`;
  const headers = new Headers();
  const cookie = req.headers.get("cookie");
  if (cookie) headers.set("cookie", cookie);
  const ct = req.headers.get("content-type");
  if (ct) headers.set("content-type", ct);
  const csrf = req.headers.get("x-csrf-token");
  if (csrf) headers.set("x-csrf-token", csrf);
  const reqId = req.headers.get("x-request-id") ?? crypto.randomUUID();
  headers.set("x-request-id", reqId);

  const hasBody = req.method !== "GET" && req.method !== "HEAD";
  const res = await fetch(url, {
    method: req.method,
    headers,
    body: hasBody ? req.body : undefined,
    // @ts-expect-error duplex wajib untuk streaming body di Node 18+
    duplex: hasBody ? "half" : undefined,
  });
  const out = new Headers();
  const resCt = res.headers.get("content-type");
  if (resCt) out.set("content-type", resCt);
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
