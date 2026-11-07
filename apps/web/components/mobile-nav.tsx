"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  Home,
  LayoutGrid,
  Newspaper,
  PlusSquare,
  User,
  type LucideIcon,
} from "lucide-react";

import { useSession } from "@/lib/auth/session";
import { isActivePath } from "@/lib/nav";
import { cn } from "@/lib/utils";

type MobileNavItem = {
  title: string;
  href: string;
  icon: LucideIcon;
  emphasized?: boolean;
};

const mobileNav: MobileNavItem[] = [
  { title: "Beranda", href: "/", icon: Home },
  { title: "Karya", href: "/karya", icon: LayoutGrid },
  { title: "Buat", href: "/studio/baru", icon: PlusSquare, emphasized: true },
  { title: "Blog", href: "/blog", icon: Newspaper },
  { title: "Profil", href: "/masuk", icon: User },
];

export function MobileNav() {
  const pathname = usePathname();
  const { session, status } = useSession();
  const profileHref =
    status === "authenticated" && session.user.username
      ? `/@${session.user.username}`
      : "/masuk";

  return (
    <nav
      aria-label="Navigasi bawah"
      className="fixed inset-x-0 bottom-0 z-40 border-t bg-background/95 pb-[env(safe-area-inset-bottom)] backdrop-blur md:hidden"
    >
      <ul className="mx-auto grid w-full max-w-6xl grid-cols-5">
        {mobileNav.map((item) => {
          const href = item.href === "/masuk" ? profileHref : item.href;
          const active = isActivePath(pathname, href);
          const Icon = item.icon;

          return (
            <li key={item.href}>
              <Link
                href={href}
                aria-current={active ? "page" : undefined}
                className={cn(
                  "flex flex-col items-center justify-center gap-1 py-2 text-[0.65rem] font-medium transition-colors",
                  active
                    ? "text-foreground"
                    : "text-muted-foreground hover:text-foreground",
                )}
              >
                <span
                  className={cn(
                    "flex size-8 items-center justify-center rounded-full",
                    item.emphasized &&
                      "size-9 bg-primary text-primary-foreground shadow-sm",
                  )}
                >
                  <Icon aria-hidden="true" className="size-5" />
                </span>
                <span>{item.title}</span>
              </Link>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}
