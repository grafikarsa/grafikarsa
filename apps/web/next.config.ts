import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  // Loader gambar khusus (lib/image-loader.ts).
  //
  // Media disajikan lewat BFF di origin yang sama (/api/v1/media/:id), jadi
  // tidak ada remotePatterns: seluruh media sudah same-origin. Memakai URL
  // absolut untuk media justru membocorkan struktur object key storage.
  //
  // deviceSizes & imageSizes sengaja dibuat hanya 320 dan 800 — dua lebar
  // yang benar-benar ada sebagai varian di storage. Tanpa ini Next membuat
  // srcset dengan lebar (1080, 1200, ...) yang tidak punya varian, sehingga
  // API menyajikan file asli yang jauh lebih besar untuk setiap request.
  images: {
    loader: "custom",
    loaderFile: "./lib/image-loader.ts",
    deviceSizes: [320, 800],
    imageSizes: [320, 800],
  },
};

export default nextConfig;
