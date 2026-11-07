"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { ChevronLeft, Inbox, Pencil } from "lucide-react";

import NotFound from "@/app/not-found";
import { apiErrorMessage } from "@/components/admin/form-error";
import { QueryError } from "@/components/admin/query-error";
import { EditUserDialog } from "@/components/admin/users/edit-user-dialog";
import {
  ACCOUNT_STATUS_LABELS,
  ACCOUNT_STATUS_VARIANTS,
  ACCOUNT_TYPE_LABELS,
  PASSWORD_KIND_LABELS,
  formatDateTime,
  initialsOf,
} from "@/components/admin/users/labels";
import { MajorBadges } from "@/components/admin/users/major-combobox";
import { UserActions } from "@/components/admin/users/user-actions";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  useGetAdminMajors,
  useGetAdminUser,
} from "@/lib/api/gen/endpoints";
import type { AdminUserDetail, Major } from "@/lib/api/gen/models";
import { ApiError } from "@/lib/api/fetcher";
import { ADMIN_PERMISSIONS, can, canGlobally } from "@/lib/auth/permissions";
import { useSession } from "@/lib/auth/session";

function Field({
  label,
  value,
  mono = false,
}: {
  label: string;
  value: React.ReactNode;
  mono?: boolean;
}) {
  return (
    <div className="space-y-1">
      <dt className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
        {label}
      </dt>
      <dd className={mono ? "font-mono text-sm" : "text-sm"}>{value}</dd>
    </div>
  );
}

function EmptyState({ message }: { message: string }) {
  return (
    <div className="py-10 text-center">
      <Inbox
        aria-hidden="true"
        className="mx-auto mb-2 size-6 text-muted-foreground"
      />
      <p className="text-sm text-muted-foreground">{message}</p>
    </div>
  );
}

function outcomeLabel(outcome: string | null): string {
  if (!outcome) {
    return "—";
  }

  switch (outcome) {
    case "lulus":
      return "Lulus";
    case "pindah":
      return "Pindah";
    default:
      return outcome;
  }
}

function ProfileTab({
  user,
  majors,
  canEdit,
  onEdit,
}: {
  user: AdminUserDetail;
  majors: Major[];
  canEdit: boolean;
  onEdit: () => void;
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-start justify-between gap-3">
        <div className="space-y-1">
          <CardTitle>Profil</CardTitle>
          <CardDescription>
            Data profil dan akademik pengguna.
          </CardDescription>
        </div>
        {canEdit ? (
          <Button type="button" variant="outline" size="sm" onClick={onEdit}>
            <Pencil aria-hidden="true" />
            Ubah
          </Button>
        ) : null}
      </CardHeader>
      <CardContent>
        <dl className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <Field label="Nama lengkap" value={user.displayName} />
          <Field
            label="Tipe akun"
            value={ACCOUNT_TYPE_LABELS[user.accountType]}
          />
          <Field
            label="NISN"
            value={user.nisn ?? "—"}
            mono={Boolean(user.nisn)}
          />
          <Field
            label="NIP/ID staf"
            value={user.staffNumber ?? "—"}
            mono={Boolean(user.staffNumber)}
          />
          <Field
            label="Jurusan diampu"
            value={
              user.accountType === "teacher" ? (
                <MajorBadges
                  majors={majors}
                  majorIds={user.teacherMajorIds}
                />
              ) : (
                "—"
              )
            }
          />
          <Field
            label="Angkatan"
            value={user.entryYear?.toString() ?? "—"}
          />
          <Field
            label="Tahun lulus"
            value={user.graduationYear?.toString() ?? "—"}
          />
          <Field
            label="Terakhir masuk"
            value={formatDateTime(user.lastLoginAt)}
          />
          <Field label="Dibuat" value={formatDateTime(user.createdAt)} />
          {user.deactivatedAt ? (
            <Field
              label="Dinonaktifkan"
              value={formatDateTime(user.deactivatedAt)}
            />
          ) : null}
        </dl>
      </CardContent>
    </Card>
  );
}

function KelasTab({ user }: { user: AdminUserDetail }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Riwayat kelas</CardTitle>
        <CardDescription>
          Kelas yang pernah diikuti, terbaru lebih dulu.
        </CardDescription>
      </CardHeader>
      <CardContent>
        {user.enrollments.length === 0 ? (
          <EmptyState message="Belum ada riwayat kelas." />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Kelas</TableHead>
                <TableHead className="w-28">Jurusan</TableHead>
                <TableHead className="w-36">Tahun ajaran</TableHead>
                <TableHead className="w-28">Hasil</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {user.enrollments.map((enrollment) => (
                <TableRow key={`${enrollment.classId}-${enrollment.academicYearLabel}`}>
                  <TableCell className="font-medium">
                    {enrollment.className}
                  </TableCell>
                  <TableCell className="font-mono text-xs">
                    {enrollment.majorCode}
                  </TableCell>
                  <TableCell>{enrollment.academicYearLabel}</TableCell>
                  <TableCell>{outcomeLabel(enrollment.outcome)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}

function RoleTab({
  user,
  majors,
}: {
  user: AdminUserDetail;
  majors: Major[];
}) {
  const majorById = useMemo(
    () => new Map(majors.map((major) => [major.id, major])),
    [majors],
  );

  return (
    <Card>
      <CardHeader>
        <CardTitle>Role</CardTitle>
        <CardDescription>
          Penugasan role aktif pengguna. Pengelolaan role menyusul.
        </CardDescription>
      </CardHeader>
      <CardContent>
        {user.roles.length === 0 ? (
          <EmptyState message="Belum ada role aktif." />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Role</TableHead>
                <TableHead className="w-40">Cakupan</TableHead>
                <TableHead className="w-44">Berlaku sampai</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {user.roles.map((role) => (
                <TableRow key={role.id}>
                  <TableCell>
                    <div className="flex flex-col gap-0.5">
                      <span className="font-medium">{role.roleName}</span>
                      <span className="font-mono text-xs text-muted-foreground">
                        {role.roleSlug}
                      </span>
                    </div>
                  </TableCell>
                  <TableCell>
                    {role.majorId
                      ? (majorById.get(role.majorId)?.code ?? role.majorId)
                      : "Global"}
                  </TableCell>
                  <TableCell>
                    {role.expiresAt
                      ? formatDateTime(role.expiresAt)
                      : "Tanpa batas"}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}

function KeamananTab({ user }: { user: AdminUserDetail }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Keamanan</CardTitle>
        <CardDescription>
          Status sandi, sesi aktif, dan izin membuat konten.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <dl className="grid gap-4 sm:grid-cols-2">
          <Field
            label="Jenis sandi"
            value={PASSWORD_KIND_LABELS[user.passwordKind]}
          />
          <Field
            label="Sandi berlaku sampai"
            value={
              user.passwordExpiresAt
                ? formatDateTime(user.passwordExpiresAt)
                : "Tidak berlaku"
            }
          />
          <Field
            label="Sesi aktif"
            value={`${user.activeSessions} sesi`}
          />
          <Field
            label="Status akun"
            value={
              <Badge variant={ACCOUNT_STATUS_VARIANTS[user.status]}>
                {ACCOUNT_STATUS_LABELS[user.status]}
              </Badge>
            }
          />
          <Field
            label="Hak membuat konten"
            value={
              user.contentBlocked ? (
                <Badge variant="destructive">Dicabut</Badge>
              ) : (
                "Diizinkan"
              )
            }
          />
          <Field
            label="Terakhir masuk"
            value={formatDateTime(user.lastLoginAt)}
          />
        </dl>
      </CardContent>
    </Card>
  );
}

function DetailSkeleton() {
  return (
    <div className="space-y-4">
      <Skeleton className="h-8 w-40" />
      <Skeleton className="h-36 w-full" />
      <Skeleton className="h-10 w-full max-w-md" />
      <Skeleton className="h-64 w-full" />
    </div>
  );
}

export function UserDetailView({ id }: { id: string }) {
  const { session, status } = useSession();
  const [editOpen, setEditOpen] = useState(false);

  const authenticated = status === "authenticated";
  const canView = authenticated && can(session, ADMIN_PERMISSIONS.userView);

  const userQuery = useGetAdminUser(id, {
    query: { enabled: canView },
  });
  const majorsQuery = useGetAdminMajors({
    query: { enabled: canView },
  });

  const majors = useMemo(
    () => majorsQuery.data?.items ?? [],
    [majorsQuery.data],
  );

  if (!authenticated) {
    return <DetailSkeleton />;
  }

  if (!canView) {
    return <NotFound />;
  }

  if (
    userQuery.error instanceof ApiError &&
    userQuery.error.status === 404
  ) {
    return <NotFound />;
  }

  if (userQuery.isError) {
    return (
      <div className="space-y-4">
        <Button variant="ghost" size="sm" asChild className="-ml-2">
          <Link href="/admin/pengguna">
            <ChevronLeft aria-hidden="true" />
            Kembali ke daftar
          </Link>
        </Button>
        <QueryError
          title="Gagal memuat pengguna"
          message={apiErrorMessage(
            userQuery.error,
            "Detail pengguna tidak dapat dimuat.",
          )}
          onRetry={() => {
            void userQuery.refetch();
          }}
          isRetrying={userQuery.isFetching}
        />
      </div>
    );
  }

  if (userQuery.isPending) {
    return <DetailSkeleton />;
  }

  const user = userQuery.data;
  const canEdit =
    can(session, "user.update") &&
    (user.accountType === "student" || canGlobally(session, "user.update"));

  return (
    <div className="space-y-4">
      <Button variant="ghost" size="sm" asChild className="-ml-2">
        <Link href="/admin/pengguna">
          <ChevronLeft aria-hidden="true" />
          Kembali ke daftar
        </Link>
      </Button>

      <Card>
        <CardContent className="space-y-4">
          <div className="flex flex-wrap items-start gap-4">
            <Avatar size="lg" className="size-14">
              <AvatarFallback className="text-lg font-medium">
                {initialsOf(user.displayName)}
              </AvatarFallback>
            </Avatar>

            <div className="min-w-0 flex-1 space-y-1.5">
              <div className="flex flex-wrap items-center gap-2">
                <h2 className="text-xl font-semibold tracking-tight">
                  {user.displayName}
                </h2>
                {user.username ? (
                  <span className="text-sm text-muted-foreground">
                    @{user.username}
                  </span>
                ) : (
                  <Badge variant="outline">Belum aktivasi</Badge>
                )}
              </div>

              <div className="flex flex-wrap items-center gap-1.5">
                <Badge variant="secondary">
                  {ACCOUNT_TYPE_LABELS[user.accountType]}
                </Badge>
                <Badge variant={ACCOUNT_STATUS_VARIANTS[user.status]}>
                  {ACCOUNT_STATUS_LABELS[user.status]}
                </Badge>
                {user.contentBlocked ? (
                  <Badge variant="destructive">Konten dibatasi</Badge>
                ) : null}
              </div>

              <div className="flex flex-wrap gap-x-4 gap-y-1 text-sm text-muted-foreground">
                <span>
                  {user.accountType === "student"
                    ? "NISN"
                    : "NIP/ID staf"}
                  :{" "}
                  <span className="font-mono text-foreground">
                    {user.accountType === "student"
                      ? (user.nisn ?? "—")
                      : (user.staffNumber ?? "—")}
                  </span>
                </span>
                <span>
                  Kelas:{" "}
                  <span className="text-foreground">
                    {user.currentClass?.name ?? "—"}
                  </span>
                </span>
              </div>
            </div>
          </div>

          <UserActions user={user} />
        </CardContent>
      </Card>

      <Tabs defaultValue="profil">
        <TabsList variant="line">
          <TabsTrigger value="profil">Profil</TabsTrigger>
          <TabsTrigger value="kelas">Kelas</TabsTrigger>
          <TabsTrigger value="role">Role</TabsTrigger>
          <TabsTrigger value="keamanan">Keamanan</TabsTrigger>
        </TabsList>

        <TabsContent value="profil" className="pt-4">
          <ProfileTab
            user={user}
            majors={majors}
            canEdit={canEdit}
            onEdit={() => setEditOpen(true)}
          />
        </TabsContent>

        <TabsContent value="kelas" className="pt-4">
          <KelasTab user={user} />
        </TabsContent>

        <TabsContent value="role" className="pt-4">
          <RoleTab user={user} majors={majors} />
        </TabsContent>

        <TabsContent value="keamanan" className="pt-4">
          <KeamananTab user={user} />
        </TabsContent>
      </Tabs>

      {editOpen ? (
        <EditUserDialog
          user={user}
          canEditGlobally={canGlobally(session, "user.update")}
          onClose={() => setEditOpen(false)}
        />
      ) : null}
    </div>
  );
}
