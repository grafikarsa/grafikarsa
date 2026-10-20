import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";
import { Providers } from "./providers";
import { SiteNav } from "@/components/layout/site-nav";

const inter = Inter({ subsets: ["latin"], display: "swap" });

export const metadata: Metadata = {
  title: "Grafikarsa",
  description: "Katalog portfolio warga SMKN 4 Malang",
  // metadataBase dipakai Next.js untuk mengubah URL relatif (gambar OG,
  // canonical, sitemap) jadi absolut. Tanpa ini, gambar OG di halaman karya
  // akan menunjuk ke localhost — crawler tidak bisa mengambilnya, jadi
  // pratinjau kartu sosial tidak pernah muncul.
  metadataBase: new URL(process.env.NEXT_PUBLIC_APP_URL ?? "http://localhost:3000"),
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="id" suppressHydrationWarning>
      <body className={`${inter.className} antialiased`}>
        <Providers>
          <SiteNav />
          {children}
        </Providers>
      </body>
    </html>
  );
}
