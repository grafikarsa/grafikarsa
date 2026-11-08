"use client";

import { useMemo } from "react";

import { groupPermissions } from "@/components/admin/roles/permission-groups";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import type { AdminPermission } from "@/lib/api/gen/models";
import { canGlobally } from "@/lib/auth/permissions";
import { useSession } from "@/lib/auth/session";
import { cn } from "@/lib/utils";

export function PermissionChecklist({
  permissions,
  selected,
  onChange,
  disabled = false,
}: {
  permissions: AdminPermission[];
  selected: string[];
  onChange: (selected: string[]) => void;
  disabled?: boolean;
}) {
  const { session } = useSession();
  const groups = useMemo(() => groupPermissions(permissions), [permissions]);
  const selectedSet = useMemo(() => new Set(selected), [selected]);

  const toggle = (key: string) => {
    if (selectedSet.has(key)) {
      onChange(selected.filter((item) => item !== key));
    } else {
      onChange([...selected, key]);
    }
  };

  if (groups.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        Katalog izin tidak tersedia.
      </p>
    );
  }

  return (
    <div className="space-y-4">
      {groups.map((group) => (
        <fieldset key={group.prefix} className="space-y-1">
          <legend className="pb-1 text-xs font-medium tracking-wide text-muted-foreground uppercase">
            {group.title}
          </legend>

          {group.items.map((permission) => {
            const held = canGlobally(session, permission.key);
            const row = (
              <label
                className={cn(
                  "flex items-start gap-3 rounded-md px-2 py-1.5",
                  held && !disabled
                    ? "cursor-pointer hover:bg-muted/50"
                    : "cursor-not-allowed opacity-60",
                )}
              >
                <Checkbox
                  className="mt-0.5"
                  checked={selectedSet.has(permission.key)}
                  disabled={disabled || !held}
                  onCheckedChange={() => toggle(permission.key)}
                />
                <span className="min-w-0 space-y-0.5">
                  <span className="block font-mono text-xs">
                    {permission.key}
                  </span>
                  <span className="block text-xs text-muted-foreground">
                    {permission.description}
                  </span>
                </span>
              </label>
            );

            if (!held) {
              return (
                <Tooltip key={permission.key}>
                  <TooltipTrigger asChild>{row}</TooltipTrigger>
                  <TooltipContent>Kamu tidak memiliki izin ini</TooltipContent>
                </Tooltip>
              );
            }

            return <div key={permission.key}>{row}</div>;
          })}
        </fieldset>
      ))}
    </div>
  );
}
