import type { AdminPermission } from "@/lib/api/gen/models";

export type PermissionGroup = {
  prefix: string;
  title: string;
  items: AdminPermission[];
};

const GROUP_TITLES: { prefix: string; title: string }[] = [
  { prefix: "content.", title: "Konten" },
  { prefix: "comment.", title: "Komentar" },
  { prefix: "profile.", title: "Profil" },
  { prefix: "report.", title: "Laporan" },
  { prefix: "moderation.", title: "Moderasi" },
  { prefix: "appeal.", title: "Banding" },
  { prefix: "group.", title: "Grup" },
  { prefix: "user.", title: "Pengguna" },
  { prefix: "role.", title: "Role" },
  { prefix: "academic.", title: "Akademik" },
  { prefix: "promotion.", title: "Kenaikan Kelas" },
  { prefix: "badge.", title: "Badge" },
  { prefix: "taxonomy.", title: "Taksonomi" },
  { prefix: "settings.", title: "Pengaturan" },
  { prefix: "audit.", title: "Audit" },
  { prefix: "dashboard.", title: "Dashboard" },
  { prefix: "curation.", title: "Kurasi" },
  { prefix: "assignment.", title: "Penugasan" },
  { prefix: "assessment.", title: "Asesmen" },
];

const GROUP_ORDER = new Map(
  GROUP_TITLES.map((group, index) => [group.prefix, index]),
);

const GROUP_TITLE = new Map(
  GROUP_TITLES.map((group) => [group.prefix, group.title]),
);

/** Returns the domain prefix of a permission key, e.g. "content." */
export function permissionPrefix(key: string): string {
  const separator = key.indexOf(".");

  return separator === -1 ? "" : key.slice(0, separator + 1);
}

function groupTitle(prefix: string): string {
  return GROUP_TITLE.get(prefix) ?? "Lainnya";
}

function sortGroups(groups: PermissionGroup[]): PermissionGroup[] {
  return groups.sort((a, b) => {
    const aOrder = GROUP_ORDER.get(a.prefix) ?? Number.MAX_SAFE_INTEGER;
    const bOrder = GROUP_ORDER.get(b.prefix) ?? Number.MAX_SAFE_INTEGER;

    if (aOrder !== bOrder) {
      return aOrder - bOrder;
    }

    return a.title.localeCompare(b.title);
  });
}

/** Groups the permission catalog by domain prefix in display order. */
export function groupPermissions(
  permissions: AdminPermission[],
): PermissionGroup[] {
  const groups = new Map<string, PermissionGroup>();

  for (const permission of permissions) {
    const prefix = permissionPrefix(permission.key);
    const group = groups.get(prefix) ?? {
      prefix,
      title: groupTitle(prefix),
      items: [],
    };

    group.items.push(permission);
    groups.set(prefix, group);
  }

  for (const group of groups.values()) {
    group.items.sort((a, b) => a.key.localeCompare(b.key));
  }

  return sortGroups([...groups.values()]);
}

/**
 * Groups role permission keys by domain prefix, attaching descriptions from
 * the permission catalog.
 */
export function groupPermissionKeys(
  keys: string[],
  catalog: AdminPermission[],
): PermissionGroup[] {
  const byKey = new Map(
    catalog.map((permission) => [permission.key, permission]),
  );

  return groupPermissions(
    keys.map(
      (key) =>
        byKey.get(key) ?? { key, description: "", scopable: false },
    ),
  );
}
