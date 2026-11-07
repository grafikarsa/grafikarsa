export type NavItem = {
  title: string;
  href: string;
};

export const mainNav: NavItem[] = [
  { title: "Beranda", href: "/" },
  { title: "Karya", href: "/karya" },
  { title: "Blog", href: "/blog" },
  { title: "Jurusan", href: "/jurusan" },
];

export function isActivePath(pathname: string, href: string) {
  return href === "/"
    ? pathname === "/"
    : pathname === href || pathname.startsWith(`${href}/`);
}
