import type { Session } from "@/lib/api/gen/models";

export const ADMIN_PERMISSIONS = {
  userView: "user.view",
  roleManage: "role.manage",
  roleAssign: "role.assign",
  manageMajors: "academic.manage_majors",
  manageYears: "academic.manage_years",
  manageClasses: "academic.manage_classes",
  auditView: "audit.view",
} as const;

export type AdminNavItem = {
  label: string;
  href: string;
  permissions: string[];
};

const ADMIN_SECTIONS: AdminNavItem[] = [
  {
    label: "Pengguna",
    href: "/admin/pengguna",
    permissions: [ADMIN_PERMISSIONS.userView],
  },
  {
    label: "Role",
    href: "/admin/role",
    permissions: [ADMIN_PERMISSIONS.roleManage, ADMIN_PERMISSIONS.roleAssign],
  },
  {
    label: "Jurusan",
    href: "/admin/akademik/jurusan",
    permissions: [ADMIN_PERMISSIONS.manageMajors],
  },
  {
    label: "Tahun Ajaran",
    href: "/admin/akademik/tahun-ajaran",
    permissions: [ADMIN_PERMISSIONS.manageYears],
  },
  {
    label: "Kelas",
    href: "/admin/akademik/kelas",
    permissions: [ADMIN_PERMISSIONS.manageClasses],
  },
  {
    label: "Audit",
    href: "/admin/audit",
    permissions: [ADMIN_PERMISSIONS.auditView],
  },
];

function hasGrant(session: Session | null, permission: string): boolean {
  return (
    session?.grants.some((grant) => grant.permission === permission) ?? false
  );
}

/** Reports whether the user holds the permission globally or scoped. */
export function can(session: Session | null, permission: string): boolean {
  return hasGrant(session, permission);
}

/** Reports whether the user holds the permission without a major scope. */
export function canGlobally(
  session: Session | null,
  permission: string,
): boolean {
  return (
    session?.grants.some(
      (grant) => grant.permission === permission && grant.global,
    ) ?? false
  );
}

/** Reports whether the user holds the permission globally or for one major. */
export function canForMajor(
  session: Session | null,
  permission: string,
  majorId: string,
): boolean {
  return (
    session?.grants.some(
      (grant) =>
        grant.permission === permission &&
        (grant.global || grant.majorIds.includes(majorId)),
    ) ?? false
  );
}

/**
 * Returns the admin navigation items visible to the session. The dashboard is
 * only shown when at least one section is visible, so an account without admin
 * permissions sees no admin navigation at all.
 */
export function adminNav(session: Session | null): AdminNavItem[] {
  const sections = ADMIN_SECTIONS.filter((item) =>
    item.permissions.some((permission) => hasGrant(session, permission)),
  );

  if (sections.length === 0) {
    return [];
  }

  return [{ label: "Dashboard", href: "/admin", permissions: [] }, ...sections];
}
