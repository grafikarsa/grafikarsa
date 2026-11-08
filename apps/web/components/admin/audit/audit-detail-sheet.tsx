"use client";

import { actionLabel, shortId, targetTypeLabel } from "@/components/admin/audit/audit-labels";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import type { AuditLogEntry } from "@/lib/api/gen/models";

const DATE_TIME = new Intl.DateTimeFormat("id-ID", {
  dateStyle: "full",
  timeStyle: "long",
});

function formatOccurredAt(value: string): string {
  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return DATE_TIME.format(date);
}

function DetailRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex gap-3 text-sm">
      <dt className="w-32 shrink-0 text-muted-foreground">{label}</dt>
      <dd className="min-w-0 break-words">{value}</dd>
    </div>
  );
}

export function AuditDetailSheet({
  entry,
  onClose,
}: {
  entry: AuditLogEntry | null;
  onClose: () => void;
}) {
  const hasData = entry ? Object.keys(entry.data).length > 0 : false;

  return (
    <Sheet
      open={entry !== null}
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
    >
      {entry ? (
        <SheetContent className="sm:max-w-lg">
          <SheetHeader>
            <SheetTitle>
              {actionLabel(entry.action) ?? (
                <code className="font-mono text-sm">{entry.action}</code>
              )}
            </SheetTitle>
            <SheetDescription>
              Entri #{entry.id} · {formatOccurredAt(entry.occurredAt)}
            </SheetDescription>
          </SheetHeader>

          <div className="flex-1 space-y-4 overflow-y-auto px-4 pb-6">
            <dl className="space-y-2">
              <DetailRow
                label="Aktor"
                value={
                  entry.actorId ? (
                    <span>
                      {entry.actorName ?? "Pengguna"}
                      {entry.actorUsername ? (
                        <span className="text-muted-foreground">
                          {" "}
                          @{entry.actorUsername}
                        </span>
                      ) : null}
                    </span>
                  ) : (
                    "Sistem"
                  )
                }
              />
              <DetailRow label="Aksi" value={<code className="font-mono text-xs">{entry.action}</code>} />
              <DetailRow
                label="Target"
                value={
                  entry.targetId ? (
                    <span>
                      {targetTypeLabel(entry.targetType) ?? "Target"}{" "}
                      <code
                        className="font-mono text-xs text-muted-foreground"
                        title={entry.targetId}
                      >
                        {shortId(entry.targetId)}
                      </code>
                    </span>
                  ) : (
                    "—"
                  )
                }
              />
              <DetailRow
                label="Scope jurusan"
                value={
                  entry.scopeMajorId ? (
                    <code className="font-mono text-xs" title={entry.scopeMajorId}>
                      {shortId(entry.scopeMajorId)}
                    </code>
                  ) : (
                    "—"
                  )
                }
              />
              {entry.actorIp ? (
                <DetailRow
                  label="IP"
                  value={<code className="font-mono text-xs">{entry.actorIp}</code>}
                />
              ) : null}
            </dl>

            <div className="space-y-2">
              <p className="text-sm font-medium">Data</p>
              {hasData ? (
                <pre className="max-w-full overflow-x-auto rounded-lg bg-muted p-3 font-mono text-xs">
                  {JSON.stringify(entry.data, null, 2)}
                </pre>
              ) : (
                <p className="text-sm text-muted-foreground">
                  Tidak ada data tambahan.
                </p>
              )}
            </div>
          </div>
        </SheetContent>
      ) : null}
    </Sheet>
  );
}
