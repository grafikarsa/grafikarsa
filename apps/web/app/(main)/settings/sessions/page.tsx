"use client";

import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { apiGet, apiSend, logout } from "@/lib/api";
import { useAuthStore } from "@/lib/auth-store";

type Session = {
  id: string;
  device_label: string;
  ip: string;
  expires_at: string;
  is_current: boolean;
};

export default function SessionsPage() {
  const user = useAuthStore((s) => s.user);
  const [sessions, setSessions] = useState<Session[]>([]);

  async function load() {
    try {
      const data = await apiGet<Session[]>("v1/auth/sessions");
      setSessions(data);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal.");
    }
  }

  useEffect(() => {
    load();
  }, []);

  return (
    <main className="mx-auto max-w-xl p-4">
      <Card>
        <CardHeader>
          <CardTitle>Sesi — {user?.name ?? "tamu"}</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-2">
          {sessions.map((s) => (
            <div key={s.id} className="flex items-center justify-between border-b py-2">
              <div>
                <p className="text-sm font-semibold">
                  {s.device_label || "Perangkat"} {s.is_current && "(ini)"}
                </p>
                <p className="text-muted-foreground text-xs">{s.ip}</p>
              </div>
              {!s.is_current && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={async () => {
                    await apiSend(`v1/auth/sessions/${s.id}`, "DELETE");
                    load();
                  }}
                >
                  Akhiri
                </Button>
              )}
            </div>
          ))}
          <Button
            variant="destructive"
            onClick={async () => {
              await apiSend("v1/auth/logout-all", "POST");
              logout();
              window.location.href = "/login";
            }}
          >
            Keluar semua perangkat
          </Button>
        </CardContent>
      </Card>
    </main>
  );
}
