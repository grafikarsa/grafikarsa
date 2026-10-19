"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { SelectField } from "@/components/ui/select-field";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { apiGet, apiSend } from "@/lib/api";

type Role = { id: string; slug: string; name: string; scope_type: string; is_system: boolean };

const PERMISSIONS = [
  "portfolios.moderate_jurusan",
  "portfolios.moderate",
  "users.view_jurusan",
  "users.create_jurusan",
  "users.edit_jurusan",
  "academic.manage_classes",
  "assessments.score",
  "series.view",
  "feedback.triage",
  "feedback.reply",
  "changelog.view",
];

export default function RolesPage() {
  const qc = useQueryClient();
  const [slug, setSlug] = useState("");
  const [name, setName] = useState("");
  const [perms, setPerms] = useState<string[]>(["portfolios.moderate_jurusan"]);
  const [assignUser, setAssignUser] = useState("");
  const [assignRole, setAssignRole] = useState("");

  const roles = useQuery({ queryKey: ["admin-roles"], queryFn: () => apiGet<Role[]>("v1/admin/roles") });
  const users = useQuery({ queryKey: ["admin-users-list"], queryFn: () => apiGet<{ id: string; username: string }[]>("v1/admin/users") });

  const create = useMutation({
    mutationFn: () => apiSend("v1/admin/roles", "POST", { slug, name, permissions: perms }),
    onSuccess: () => {
      toast.success("Role dibuat.");
      setSlug("");
      setName("");
      qc.invalidateQueries({ queryKey: ["admin-roles"] });
    },
    onError: (e: unknown) => toast.error(e instanceof Error ? e.message : "Gagal."),
  });

  const assign = useMutation({
    mutationFn: () => apiSend("v1/admin/roles/assign", "POST", { user_id: assignUser, role_slug: assignRole }),
    onSuccess: () => {
      toast.success("Role diberikan.");
      qc.invalidateQueries({ queryKey: ["admin-roles"] });
    },
    onError: (e: unknown) => toast.error(e instanceof Error ? e.message : "Gagal."),
  });

  const roleList = Array.isArray(roles.data) ? roles.data : [];
  const userList = Array.isArray(users.data) ? users.data : [];

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardHeader>
          <CardTitle>Role & permission</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Slug</TableHead>
                <TableHead>Nama</TableHead>
                <TableHead>Scope</TableHead>
                <TableHead>Tipe</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {roleList.map((r) => (
                <TableRow key={r.id}>
                  <TableCell className="font-mono text-xs">{r.slug}</TableCell>
                  <TableCell>{r.name}</TableCell>
                  <TableCell>{r.scope_type}</TableCell>
                  <TableCell>{r.is_system ? <Badge variant="secondary">sistem</Badge> : <Badge>custom</Badge>}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Buat role custom</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <p className="text-muted-foreground text-xs">
            Role custom hanya boleh memuat permission yang kamu miliki sendiri (anti-eskalasi).
          </p>
          <div className="grid gap-2 md:grid-cols-2">
            <div className="grid gap-1">
              <Label htmlFor="rslug">Slug</Label>
              <Input id="rslug" value={slug} onChange={(e) => setSlug(e.target.value)} placeholder="moderator_rpl" />
            </div>
            <div className="grid gap-1">
              <Label htmlFor="rname">Nama</Label>
              <Input id="rname" value={name} onChange={(e) => setName(e.target.value)} placeholder="Moderator RPL" />
            </div>
          </div>
          <div className="flex flex-wrap gap-2">
            {PERMISSIONS.map((p) => (
              <label key={p} className="flex items-center gap-1 rounded border px-2 py-1 text-xs">
                <input
                  type="checkbox"
                  checked={perms.includes(p)}
                  onChange={(e) => setPerms(e.target.checked ? [...perms, p] : perms.filter((x) => x !== p))}
                />
                {p}
              </label>
            ))}
          </div>
          <Button onClick={() => create.mutate()} disabled={!slug || !name || perms.length === 0 || create.isPending}>
            Buat role
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Berikan role ke user</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <div className="grid gap-2 md:grid-cols-3">
            <div className="grid gap-1">
              <Label>User</Label>
              <SelectField
                value={assignUser}
                onValueChange={setAssignUser}
                placeholder="Pilih user..."
                options={userList.map((u) => ({ value: u.id, label: `@${u.username}` }))}
              />
            </div>
            <div className="grid gap-1">
              <Label>Role</Label>
              <SelectField
                value={assignRole}
                onValueChange={setAssignRole}
                placeholder="Pilih role..."
                options={roleList.map((r) => ({ value: r.slug, label: r.name }))}
              />
            </div>
            <Button className="self-end" onClick={() => assign.mutate()} disabled={!assignUser || !assignRole || assign.isPending}>
              Berikan
            </Button>
          </div>
          <p className="text-muted-foreground text-xs">
            Scope jurusan otomatis mengikuti jurusanmu bila kamu admin jurusan.
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
