"use client";

import { Copy } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { formatDateTime } from "@/components/admin/users/labels";

/**
 * Displays a password that the API returns only once. The password is rendered
 * in a large monospace box so it can be copied or written down easily.
 */
export function OneTimePassword({
  password,
  expiresAt,
  label = "Sandi",
}: {
  password: string;
  expiresAt?: string | null;
  label?: string;
}) {
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(password);
      toast.success("Sandi disalin.");
    } catch {
      toast.error("Tidak bisa menyalin. Salin sandi secara manual.");
    }
  };

  return (
    <div className="space-y-3">
      <div className="space-y-1.5">
        <p className="text-sm font-medium">{label}</p>
        <div className="flex flex-col gap-2 rounded-xl border bg-muted/40 p-4 sm:flex-row sm:items-center">
          <code className="min-w-0 flex-1 text-center font-mono text-xl leading-relaxed font-semibold tracking-wider break-all sm:text-left sm:text-2xl">
            {password}
          </code>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={copy}
            className="self-center sm:self-auto"
          >
            <Copy aria-hidden="true" />
            Salin
          </Button>
        </div>
      </div>

      {expiresAt ? (
        <p className="text-sm text-muted-foreground">
          Berlaku sampai {formatDateTime(expiresAt)}.
        </p>
      ) : null}

      <p className="text-sm font-medium text-destructive">
        Sandi ini hanya ditampilkan sekali. Catat atau cetak kartu akun.
      </p>
    </div>
  );
}
