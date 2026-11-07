"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

import { isActivePath, mainNav } from "@/lib/nav";
import { cn } from "@/lib/utils";

export function MainNav() {
  const pathname = usePathname();

  return (
    <nav
      aria-label="Navigasi utama"
      className="hidden items-center gap-6 md:flex"
    >
      {mainNav.map((item) => {
        const active = isActivePath(pathname, item.href);

        return (
          <Link
            key={item.href}
            href={item.href}
            aria-current={active ? "page" : undefined}
            className={cn(
              "text-sm font-medium transition-colors hover:text-foreground",
              active ? "text-foreground" : "text-muted-foreground",
            )}
          >
            {item.title}
          </Link>
        );
      })}
    </nav>
  );
}
