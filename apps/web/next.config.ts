import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Output standalone for Docker deployment
  output: "standalone",
  typescript: {
    // Skip TypeScript errors during build for faster deployment
    ignoreBuildErrors: true,
  },
  images: {
    remotePatterns: [
      {
        protocol: "https",
        hostname: "picsum.photos",
      },
      {
        protocol: "https",
        hostname: "i.pravatar.cc",
      },
      {
        protocol: "https",
        hostname: "images.unsplash.com",
      },
      // Production MinIO/CDN - Generic (ganti sesuai domain)
      {
        protocol: "https",
        hostname: "cdn.grafikarsa.com",
      },
      {
        protocol: "https",
        hostname: "storage.grafikarsa.com",
      },
      // Jelastic deployment
      {
        protocol: "https",
        hostname: "storage.rafapradana.com",
      },
      {
        protocol: "https",
        hostname: "cdn.rafapradana.com",
      },
      {
        protocol: "http",
        hostname: "grafikarsa.jh-beon.cloud",
        port: "9000",
      },
      {
        protocol: "https",
        hostname: "grafikarsa.jh-beon.cloud",
        port: "9000",
      },
      // Development MinIO
      {
        protocol: "http",
        hostname: "localhost",
        port: "9000",
      },
    ],
    // Skip image optimization for local development
    unoptimized: process.env.NODE_ENV === "development",
  },
  // Production optimizations
  poweredByHeader: false,
  compress: true,
  async headers() {
    return [
      {
        source: '/:path*',
        headers: [
          {
            key: 'X-DNS-Prefetch-Control',
            value: 'on',
          },
          {
            key: 'Strict-Transport-Security',
            value: 'max-age=63072000; includeSubDomains; preload',
          },
        ],
      },
      {
        // Prevent aggressive caching of the HTML document
        source: '/',
        headers: [
          {
            key: 'Cache-Control',
            value: 'no-store, must-revalidate',
          },
        ],
      },
      {
        // Match all other pages
        source: '/((?!api|_next/static|_next/image|favicon.ico).*)',
        headers: [
          {
            key: 'Cache-Control',
            value: 'no-store, must-revalidate',
          },
        ],
      },
    ];
  },
};

export default nextConfig;