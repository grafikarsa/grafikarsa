import { ImageResponse } from "next/og";
import { getPortfolioForOg } from "./data";
import type { PortfolioDetail } from "@/lib/portfolio-types";

// OG image per portfolio (PRD §14.3).
//
// Gambar dibuat on-the-fly oleh Next.js (ImageResponse), jadi tiap karya
// punya OG image sendiri dengan judul dan nama pembuatnya — tanpa harus
// menyimpan file OG per karya di storage.
//
// Ukuran 1200x630 adalah ukuran standar Open Graph.

export const size = { width: 1200, height: 630 };
export const contentType = "image/png";
// Data portfolio berubah; cache 1 jam supaya tidak mem-fetch API tiap kali
// platform sosialasking gambar.
export const alt = "Karya di Grafikarsa";

export default async function OpenGraphImage({
  params,
}: {
  params: Promise<{ username: string; slug: string }>;
}) {
  const { username, slug } = await params;
  const p = await getPortfolioForOg(username, slug);

  const title = p?.title ?? "Karya di Grafikarsa";
  const author = p?.author_name ?? username;

  return new ImageResponse(
    <div
      style={{
        width: "100%",
        height: "100%",
        display: "flex",
        flexDirection: "column",
        justifyContent: "space-between",
        padding: 64,
        background: "linear-gradient(135deg, #09090b 0%, #18181b 55%, #27272a 100%)",
        color: "#fafafa",
        fontFamily: "sans-serif",
      }}
    >
      <div style={{ display: "flex", alignItems: "center", gap: 16 }}>
        <div
          style={{
            width: 16,
            height: 16,
            borderRadius: 9999,
            background: "#fafafa",
            display: "flex",
          }}
        />
        <div style={{ fontSize: 34, letterSpacing: "-0.02em", opacity: 0.85 }}>Grafikarsa</div>
      </div>

      <div style={{ display: "flex", flexDirection: "column", gap: 20 }}>
        <div
          style={{
            fontSize: title.length > 60 ? 62 : 76,
            lineHeight: 1.1,
            letterSpacing: "-0.03em",
            fontWeight: 700,
            display: "flex",
          }}
        >
          {title}
        </div>
        <div style={{ fontSize: 34, opacity: 0.7, display: "flex" }}>oleh {author}</div>
      </div>
    </div>,
    { ...size },
  );
}

// getPortfolio mengambil data karya untuk OG image.
//
// Sengaja tidak memakai helper yang sama dengan halaman: halaman memakai
// `next: { revalidate }`, sedangkan OG image di-route terpisah dan tidak
// boleh ikut mewarisi cache halaman.
