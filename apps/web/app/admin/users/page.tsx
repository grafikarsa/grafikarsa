"use client";

import { useQueryClient } from "@tanstack/react-query";
import { Suspense, useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { SelectField } from "@/components/ui/select-field";
import { DataTable, type DataTableColumn } from "@/components/admin/data-table";
import { apiSend } from "@/lib/api";

type User = {
  id: string;
  username: string;
  name: string;
  role_primary: string;
  status: string;
};

const ROLES = [
  { value: "siswa", label: "Siswa" },
  { value: "guru", label: "Guru" },
  { value: "admin_jurusan", label: "Admin jurusan" },
  { value: "admin", label: "Admin" },
];

// Form create user. Password dibuat manual oleh admin (minimal 10 karakter,
// score zxcvbn >= 3 — dicek ulang di API).
type FormState = {
  username: string;
  email: string;
  name: string;
  role: string;
  password: string;
};

const kosong: FormState = { username: "", email: "", name: "", role: "siswa", password: "" };

export default function AdminUsersPage() {
  const [buka, setBuka] = useState(false);
  const [form, setForm] = useState<FormState>(kosong);
  const [sibuk, setSibuk] = useState(false);
  const queryClient = useQueryClient();

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (form.password.length < 10) {
      toast.error("Password minimal 10 karakter.");
      return;
    }
    setSibuk(true);
    try {
      await apiSend("v1/admin/users", "POST", {
        username: form.username.trim().toLowerCase(),
        email: form.email.trim(),
        name: form.name.trim(),
        role_primary: form.role,
        password: form.password,
      });
      toast.success(`Akun ${form.username} dibuat. User wajib ganti password saat login pertama.`);
      setForm(kosong);
      setBuka(false);
      // Segarkan tabel: DataTable meng-cache hasil query, jadi tanpa
      // invalidasi akun baru tidak akan muncul sampai cache kedaluwarsa.
      await queryClient.invalidateQueries({ queryKey: ["v1/admin/users"] });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal membuat akun.");
    } finally {
      setSibuk(false);
    }
  }

  const columns: DataTableColumn<User>[] = [
    {
      key: "username",
      label: "Username",
      sortKey: "username",
      render: (u) => u.username,
    },
    { key: "name", label: "Nama", sortKey: "name", render: (u) => u.name },
    { key: "role", label: "Role", sortKey: "role", render: (u) => u.role_primary },
    { key: "status", label: "Status", sortKey: "status", render: (u) => u.status },
    {
      key: "aksi",
      label: "Aksi",
      render: (u) => (
        <Button
          variant="outline"
          size="sm"
          onClick={async () => {
            try {
              const r = await apiSend<{ password: string }>(
                `v1/admin/users/${u.id}/reset-password`,
                "POST",
              );
              toast.success(`Password baru: ${r.password}`);
            } catch (err) {
              toast.error(err instanceof Error ? err.message : "Gagal.");
            }
          }}
        >
          Reset password
        </Button>
      ),
    },
  ];

  return (
    <main className="mx-auto max-w-4xl p-8">
      <Card>
        <CardHeader className="flex flex-row items-center justify-between gap-4">
          {/* CardTitle dirender sebagai <div>, jadi halaman ini butuh <h1>
              sendiri supaya struktur dokumennya benar. */}
          <h1 className="text-2xl font-bold">Pengguna</h1>
          {/* Grafikarsa tidak punya pendaftaran mandiri: seluruh akun dibuat
              dari sini oleh admin. */}
          <Dialog open={buka} onOpenChange={setBuka}>
            <Button size="sm" onClick={() => setBuka(true)}>
              Tambah pengguna
            </Button>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Tambah pengguna</DialogTitle>
                <DialogDescription>
                  Akun dibuat dengan password sementara. User wajib menggantinya saat login pertama.
                </DialogDescription>
              </DialogHeader>
              <form onSubmit={submit} className="flex flex-col gap-3">
                <div className="grid gap-2">
                  <Label htmlFor="new-username">Username</Label>
                  <Input
                    id="new-username"
                    value={form.username}
                    onChange={(e) => setForm({ ...form, username: e.target.value })}
                    placeholder="huruf kecil, angka, titik, _ atau - (3-30 karakter)"
                    required
                    minLength={3}
                    maxLength={30}
                    pattern="[a-z0-9._\-]{3,30}"
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="new-name">Nama lengkap</Label>
                  <Input
                    id="new-name"
                    value={form.name}
                    onChange={(e) => setForm({ ...form, name: e.target.value })}
                    required
                    minLength={2}
                    maxLength={100}
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="new-email">Email</Label>
                  <Input
                    id="new-email"
                    type="email"
                    value={form.email}
                    onChange={(e) => setForm({ ...form, email: e.target.value })}
                    required
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="new-role">Role</Label>
                  <SelectField
                    value={form.role}
                    onValueChange={(v) => setForm({ ...form, role: v })}
                    options={ROLES}
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="new-password">Password sementara</Label>
                  <Input
                    id="new-password"
                    type="text"
                    value={form.password}
                    onChange={(e) => setForm({ ...form, password: e.target.value })}
                    required
                    minLength={10}
                    placeholder="minimal 10 karakter"
                    autoComplete="off"
                  />
                  <p className="text-muted-foreground text-xs">
                    Password-nya hanya disimpan di hash. User akan diminta menggantinya setelah
                    login pertama.
                  </p>
                </div>
                <DialogFooter>
                  <Button type="submit" disabled={sibuk}>
                    {sibuk ? "Menyimpan..." : "Buat akun"}
                  </Button>
                </DialogFooter>
              </form>
            </DialogContent>
          </Dialog>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          {/* DataTable server-side: pencarian, pengurutan, dan pagination
              모두 lewat query string ke API (PRD §12.3). */}
          <Suspense fallback={<p className="text-muted-foreground text-sm">Memuat pengguna...</p>}>
            <DataTable endpoint="v1/admin/users" columns={columns} />
          </Suspense>
        </CardContent>
      </Card>
    </main>
  );
}
