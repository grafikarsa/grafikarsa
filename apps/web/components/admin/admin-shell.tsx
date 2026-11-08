"use client";

import { useEffect } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import type { LucideIcon } from "lucide-react";
import {
  CalendarRange,
  ExternalLink,
  GraduationCap,
  LayoutDashboard,
  School,
  ScrollText,
  ShieldCheck,
  Users,
} from "lucide-react";

import NotFound from "@/app/not-found";
import { ThemeToggle } from "@/components/theme-toggle";
import { Separator } from "@/components/ui/separator";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  SidebarRail,
  SidebarSeparator,
  SidebarTrigger,
} from "@/components/ui/sidebar";
import { Skeleton } from "@/components/ui/skeleton";
import { TooltipProvider } from "@/components/ui/tooltip";
import { adminNav } from "@/lib/auth/permissions";
import { useSession } from "@/lib/auth/session";
import { isActivePath } from "@/lib/nav";

const NAV_ICONS: Record<string, LucideIcon> = {
  "/admin": LayoutDashboard,
  "/admin/pengguna": Users,
  "/admin/role": ShieldCheck,
  "/admin/akademik/jurusan": GraduationCap,
  "/admin/akademik/tahun-ajaran": CalendarRange,
  "/admin/akademik/kelas": School,
  "/admin/audit": ScrollText,
};

function AdminShellSkeleton() {
  return (
    <div className="mx-auto flex min-h-[calc(100svh-3.5rem)] w-full max-w-6xl flex-col gap-4 p-4 md:p-6">
      <Skeleton className="h-8 w-48" />
      <Skeleton className="h-10 w-full max-w-xs" />
      <Skeleton className="h-64 w-full" />
    </div>
  );
}

function currentSection(pathname: string, href: string): boolean {
  return isActivePath(pathname, href);
}

export function AdminShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const { session, status } = useSession();

  useEffect(() => {
    if (status === "anonymous") {
      router.replace(`/masuk?kembali=${encodeURIComponent(pathname)}`);
    }
  }, [pathname, router, status]);

  if (status !== "authenticated") {
    return <AdminShellSkeleton />;
  }

  const nav = adminNav(session);

  if (nav.length === 0) {
    return <NotFound />;
  }

  const activeItem = [...nav]
    .filter((item) => currentSection(pathname, item.href))
    .sort((a, b) => b.href.length - a.href.length)[0];
  const title = activeItem?.label ?? "Admin";

  return (
    <TooltipProvider>
      <SidebarProvider className="min-h-[calc(100svh-3.5rem)]">
        <Sidebar
          collapsible="icon"
          className="md:top-14! md:h-[calc(100svh-3.5rem)]!"
        >
          <SidebarHeader>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton
                  asChild
                  size="lg"
                  tooltip="Admin Grafikarsa"
                  className="data-[slot=sidebar-menu-button]:hover:bg-transparent"
                >
                  <Link href="/admin">
                    <span className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground">
                      <GraduationCap aria-hidden="true" />
                    </span>
                    <span className="grid text-left leading-tight">
                      <span className="truncate font-semibold">Grafikarsa</span>
                      <span className="truncate text-xs text-muted-foreground">
                        Area admin
                      </span>
                    </span>
                  </Link>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarHeader>

          <SidebarSeparator />

          <SidebarContent>
            <SidebarGroup>
              <SidebarGroupLabel>Menu</SidebarGroupLabel>
              <SidebarGroupContent>
                <SidebarMenu>
                  {nav.map((item) => {
                    const Icon = NAV_ICONS[item.href] ?? LayoutDashboard;

                    return (
                      <SidebarMenuItem key={item.href}>
                        <SidebarMenuButton
                          asChild
                          isActive={currentSection(pathname, item.href)}
                          tooltip={item.label}
                        >
                          <Link href={item.href}>
                            <Icon aria-hidden="true" />
                            <span>{item.label}</span>
                          </Link>
                        </SidebarMenuButton>
                      </SidebarMenuItem>
                    );
                  })}
                </SidebarMenu>
              </SidebarGroupContent>
            </SidebarGroup>
          </SidebarContent>

          <SidebarFooter>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton asChild tooltip="Kembali ke situs">
                  <Link href="/">
                    <ExternalLink aria-hidden="true" />
                    <span>Kembali ke situs</span>
                  </Link>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
            <div className="flex items-center justify-between gap-2 px-1">
              <span className="truncate text-xs text-muted-foreground">
                {session.user.displayName}
              </span>
              <ThemeToggle />
            </div>
          </SidebarFooter>

          <SidebarRail />
        </Sidebar>

        <div className="relative flex min-w-0 flex-1 flex-col bg-background">
          <header className="flex h-14 shrink-0 items-center gap-2 border-b px-4">
            <SidebarTrigger className="-ml-1" />
            <Separator orientation="vertical" className="mr-1 h-4" />
            <nav aria-label="Breadcrumb" className="min-w-0">
              <ol className="flex items-center gap-1.5 text-sm">
                <li className="text-muted-foreground">
                  <Link href="/admin" className="hover:text-foreground">
                    Admin
                  </Link>
                </li>
                <li aria-hidden="true" className="text-muted-foreground">
                  /
                </li>
                <li className="min-w-0">
                  <h1 className="truncate font-medium">{title}</h1>
                </li>
              </ol>
            </nav>
          </header>
          <div className="flex-1 p-4 md:p-6">{children}</div>
        </div>
      </SidebarProvider>
    </TooltipProvider>
  );
}
