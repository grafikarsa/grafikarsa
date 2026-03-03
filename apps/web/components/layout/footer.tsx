import Link from 'next/link';

export function Footer() {
  return (
    <footer className="border-t bg-background">
      <div className="container mx-auto px-4 py-8">
        <div className="grid gap-8 md:grid-cols-3">
          {/* Brand */}
          <div>
            <h3 className="text-lg font-bold text-primary">Grafikarsa</h3>
            <p className="mt-2 text-sm text-muted-foreground">
              Platform Katalog Portofolio & Social Network Warga SMKN 4 Malang
            </p>
          </div>

          {/* Links */}
          <div>
            <h4 className="font-semibold">Navigasi</h4>
            <ul className="mt-2 space-y-2 text-sm text-muted-foreground">
              <li>
                <Link href="/" className="hover:text-primary">
                  Beranda
                </Link>
              </li>
              <li>
                <Link href="/users" className="hover:text-primary">
                  Siswa & Alumni
                </Link>
              </li>
              <li>
                <Link href="/portfolios" className="hover:text-primary">
                  Portofolio
                </Link>
              </li>
            </ul>
          </div>

          {/* Contact */}
          <div>
            <h4 className="font-semibold">Kontak</h4>
            <ul className="mt-2 space-y-2 text-sm text-muted-foreground">
              <li>SMKN 4 Malang</li>
              <li>Jl. Tanimbar No.22, Malang</li>
            </ul>
          </div>
        </div>

        <div className="mt-8 border-t pt-8 text-center text-sm text-muted-foreground">
          <p>&copy; {new Date().getFullYear()} Grafikarsa by Codepacker (RPL). All rights reserved.</p>
        </div>
      </div>
    </footer>
  );
}
