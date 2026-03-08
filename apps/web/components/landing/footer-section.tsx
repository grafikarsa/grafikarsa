'use client';

import Link from 'next/link';
import { Instagram, Youtube } from 'lucide-react';

export function FooterSection() {
  const currentYear = new Date().getFullYear();

  return (
    <footer className="border-t bg-card">
      <div className="mx-auto max-w-6xl px-6 py-12">
        <div className="grid grid-cols-1 gap-8 md:grid-cols-4">
          {/* Brand */}
          <div className="md:col-span-2">
            <h3 className="text-lg font-bold mb-3">GRAFIKARSA</h3>
            <p className="text-sm text-muted-foreground max-w-sm">
              Platform Portfolio Digital dan Social Network untuk warga SMKN 4 Malang. 
              Tunjukkan karya kreatifmu dan terhubung dengan komunitas.
            </p>
          </div>

          {/* Quick Links */}
          <div>
            <h4 className="text-sm font-semibold mb-3">Navigasi</h4>
            <ul className="space-y-2 text-sm">
              <li>
                <Link href="/portfolios" className="text-muted-foreground hover:text-foreground transition-colors">
                  Jelajahi Portfolio
                </Link>
              </li>
              <li>
                <Link href="/users" className="text-muted-foreground hover:text-foreground transition-colors">
                  Temukan Siswa
                </Link>
              </li>
              <li>
                <Link href="/login" className="text-muted-foreground hover:text-foreground transition-colors">
                  Login
                </Link>
              </li>
            </ul>
          </div>

          {/* Social Media */}
          <div>
            <h4 className="text-sm font-semibold mb-3">Social Media</h4>
            <ul className="space-y-2 text-sm">
              <li>
                <a 
                  href="https://www.instagram.com/official.smkn4malang/" 
                  target="_blank" 
                  rel="noopener noreferrer"
                  className="text-muted-foreground hover:text-foreground transition-colors inline-flex items-center gap-2"
                >
                  <Instagram className="h-4 w-4" />
                  Instagram
                </a>
              </li>
              <li>
                <a 
                  href="https://www.youtube.com/@SMKN4MalangOfficial" 
                  target="_blank" 
                  rel="noopener noreferrer"
                  className="text-muted-foreground hover:text-foreground transition-colors inline-flex items-center gap-2"
                >
                  <Youtube className="h-4 w-4" />
                  YouTube
                </a>
              </li>
            </ul>
          </div>
        </div>

        {/* Bottom Bar */}
        <div className="mt-8 pt-8 border-t">
          <div className="flex flex-col md:flex-row justify-between items-center gap-4 text-sm text-muted-foreground">
            <p>© {currentYear} SMKN 4 Malang. All rights reserved.</p>
            <div className="flex gap-6">
              <Link href="/changelog" className="hover:text-foreground transition-colors">
                Changelog
              </Link>
              <Link href="/feedback" className="hover:text-foreground transition-colors">
                Feedback
              </Link>
            </div>
          </div>
        </div>
      </div>
    </footer>
  );
}
