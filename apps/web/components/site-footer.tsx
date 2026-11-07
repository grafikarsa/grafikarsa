import Link from "next/link";

const footerNav = [
  { title: "Pedoman", href: "/pedoman" },
  { title: "Tentang", href: "/tentang" },
  { title: "Privasi", href: "/privasi" },
  { title: "Syarat", href: "/syarat" },
];

export function SiteFooter() {
  return (
    <footer className="border-t">
      <div className="mx-auto flex w-full max-w-6xl flex-col items-center justify-between gap-3 px-4 py-6 text-sm text-muted-foreground sm:flex-row">
        <p>© {new Date().getFullYear()} Grafikarsa · SMKN 4 Malang</p>

        <nav aria-label="Navigasi footer">
          <ul className="flex flex-wrap items-center justify-center gap-4">
            {footerNav.map((item) => (
              <li key={item.href}>
                <Link
                  href={item.href}
                  className="transition-colors hover:text-foreground"
                >
                  {item.title}
                </Link>
              </li>
            ))}
          </ul>
        </nav>
      </div>
    </footer>
  );
}
