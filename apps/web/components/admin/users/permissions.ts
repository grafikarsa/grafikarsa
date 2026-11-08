import type { Session } from "@/lib/api/gen/models";

export type PermissionScope = {
  global: boolean;
  majorIds: string[];
};

/**
 * Resolves the effective scope of a permission from the session grants. Global
 * access wins over scoped grants, matching the API's permission resolution.
 */
export function permissionScope(
  session: Session | null,
  permission: string,
): PermissionScope {
  const grants =
    session?.grants.filter((grant) => grant.permission === permission) ?? [];

  if (grants.some((grant) => grant.global)) {
    return { global: true, majorIds: [] };
  }

  const majorIds = new Set<string>();

  for (const grant of grants) {
    for (const majorId of grant.majorIds) {
      majorIds.add(majorId);
    }
  }

  return { global: false, majorIds: [...majorIds] };
}
