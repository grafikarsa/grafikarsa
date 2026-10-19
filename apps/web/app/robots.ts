import type { MetadataRoute } from "next";

// robots.txt: halaman privat & API tidak diindeks, sitemap didaftarkan.
export default function robots(): MetadataRoute.Robots {
  const appUrl = process.env.NEXT_PUBLIC_APP_URL ?? "http://localhost:3020";
  return {
    rules: [
      {
        userAgent: "*",
        allow: "/",
        disallow: ["/api/", "/admin", "/settings", "/notifications", "/me/", "/login"],
      },
    ],
    sitemap: `${appUrl}/sitemap.xml`,
  };
}
