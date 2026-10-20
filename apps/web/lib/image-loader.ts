import type { ImageLoaderProps } from "next/image";

/**
 * Loader gambar untuk Next.js Image.
 *
 * PRD §7.3: "Next Image loader memakai `/api/media/:id?w=800&q=75` → BFF
 * teruskan `w` ke API yang serve varian `_800` jika ada (tanpa on-the-fly
 * resize di request panas; resize hanya di worker)."
 *
 * Yang dilakukan loader ini:
 *
 * 1. Meminta lebar yang benar-benar tersedia. Media hanya punya varian
 *    320 dan 800 (dibuat worker saat upload). Kalau Next meminta lebar yang
 *    tidak ada — mis. 1080 — API akan menyajikan file asli, yang jauh lebih
 *    besar dari yang perlu. Jadi lebarnya "dibulatkan" ke varian terdekat.
 *
 * 2. Srcset jadi pendek: hanya 320 dan 800. Tanpa ini, browser mengunduh
 *    beberapa URL yang isinya sama persis — boros bandwidth tanpa untung.
 *
 * Query `q` diteruskan ke API, sesuai PRD.
 */

/** Lebar varian yang benar-benar ada di storage. */
const VARIANTS = [320, 800];

/** Lebar terkecil yang masih masuk akal (dipakai untuk gambar sangat kecil). */
const MIN_WIDTH = 32;

/**
 * snappedWidth membulatkan lebar yang diminta ke varian yang tersedia.
 *
 * Aturannya: pilih varian terkecil yang >= lebar diminta. Kalau tidak ada
 * (mis. diminta 1200 tapi varian maks 800), pakai varian terbesar.
 */
export function snappedWidth(width: number): number {
  if (!Number.isFinite(width) || width <= 0) return VARIANTS[0];
  for (const v of VARIANTS) {
    if (v >= width) return v;
  }
  return VARIANTS[VARIANTS.length - 1];
}

/**
 * mediaLoader adalah loader Next.js untuk media yang disajikan lewat BFF.
 */
export default function mediaLoader({ src, width, quality }: ImageLoaderProps): string {
  // Hanya media internal yang punya varian. URL lain (mis. gambar statis,
  // atau media v1 yang di-redirect) diteruskan apa adanya.
  if (!src.startsWith("/api/v1/media/") && !src.startsWith("/api/media/")) {
    return src;
  }

  const sep = src.includes("?") ? "&" : "?";
  const q = quality ?? 75;
  return `${src}${sep}w=${snappedWidth(width)}&q=${q}`;
}
