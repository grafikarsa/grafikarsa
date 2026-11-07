import Link from "next/link";
import { Search } from "lucide-react";

import { MainNav } from "@/components/main-nav";
import { ThemeToggle } from "@/components/theme-toggle";
import { Button } from "@/components/ui/button";

export function SiteHeader() {
  return (
    <header className="sticky top-0 z-40 w-full border-b bg-background/80 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="mx-auto flex h-14 w-full max-w-6xl items-center justify-between gap-4 px-4">
        <Link href="/" className="text-lg font-semibold tracking-tight">
          Grafikarsa
        </Link>

        <MainNav />

        <div className="flex items-center gap-1 sm:gap-2">
          <Button asChild variant="ghost" size="icon">
            <Link href="/cari" aria-label="Cari">
              <Search aria-hidden="true" />
            </Link>
          </Button>
          <ThemeToggle />
          <Button asChild size="sm" className="ml-1">
            <Link href="/masuk">Masuk</Link>
          </Button>
        </div>
      </div>
    </header>
  );
}
