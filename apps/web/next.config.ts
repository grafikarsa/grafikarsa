import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  // Tidak ada remotePatterns: seluruh media di aplikasi disajikan lewat BFF
  // pada origin yang sama (/api/v1/media/...), jadi next/image cukup memakai
  // path relatif. Memakai URL absolut untuk media justru membocorkan struktur
  // object key storage.
  images: {
    remotePatterns: [],
  },
};

export default nextConfig;
