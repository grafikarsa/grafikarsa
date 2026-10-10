"use client";

import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { apiGet, apiSend } from "@/lib/api";

type Notif = { id: string; type: string; title: string; body: string; created_at: string; is_read: boolean };

export default function NotificationsPage() {
  const [items, setItems] = useState<Notif[]>([]);

  async function load() {
    try {
      const data = await apiGet<Notif[]>("v1/notifications");
      setItems(data);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal memuat.");
    }
  }

  useEffect(() => {
    load();
    apiSend("v1/notifications/read-all", "POST").catch(() => null);
  }, []);

  return (
    <main className="mx-auto max-w-2xl p-4">
      <Card>
        <CardHeader>
          <CardTitle>Notifikasi</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-2">
          {items.length === 0 && <p className="text-muted-foreground text-sm">Tidak ada notifikasi.</p>}
          {items.map((n) => (
            <div key={n.id} className="border-b py-2">
              <p className="font-semibold">{n.title}</p>
              <p className="text-sm">{n.body}</p>
              <p className="text-muted-foreground text-xs">{n.type} · {n.created_at}</p>
              <Button
                variant="ghost"
                size="sm"
                onClick={async () => {
                  await apiSend(`v1/notifications/${n.id}`, "DELETE");
                  load();
                }}
              >
                Hapus
              </Button>
            </div>
          ))}
        </CardContent>
      </Card>
    </main>
  );
}
