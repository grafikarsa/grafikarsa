"use client";

import { useState } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import {
  KeyRound,
  ShieldCheck,
  ShieldOff,
  UserCheck,
  UserX,
} from "lucide-react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";

import {
  apiErrorMessage,
  applyApiError,
} from "@/components/admin/form-error";
import { OneTimePassword } from "@/components/admin/users/one-time-password";
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@/components/ui/alert";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Textarea } from "@/components/ui/textarea";
import {
  getGetAdminUserQueryKey,
  getGetAdminUsersQueryKey,
  usePostAdminUserContentBlock,
  usePostAdminUserContentUnblock,
  usePostAdminUserDeactivate,
  usePostAdminUserReactivate,
  usePostAdminUserReissueInitialPassword,
  usePostAdminUserResetPassword,
} from "@/lib/api/gen/endpoints";
import type { AdminUserDetail } from "@/lib/api/gen/models";
import { can, canGlobally } from "@/lib/auth/permissions";
import { useSession } from "@/lib/auth/session";

type IssuedPasswordState = {
  password: string;
  expiresAt: string;
  kind: "initial" | "temporary";
};

const blockSchema = z.object({
  reason: z
    .string()
    .trim()
    .min(3, "Alasan minimal 3 karakter.")
    .max(300, "Alasan maksimal 300 karakter."),
});

type BlockValues = z.infer<typeof blockSchema>;

function BlockContentDialog({
  userId,
  onClose,
}: {
  userId: string;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const [formError, setFormError] = useState<string | null>(null);

  const form = useForm<BlockValues>({
    resolver: zodResolver(blockSchema),
    defaultValues: { reason: "" },
  });

  const blockMutation = usePostAdminUserContentBlock({
    mutation: {
      onSuccess: () => {
        toast.success("Hak membuat konten dicabut");
        void queryClient.invalidateQueries({
          queryKey: getGetAdminUserQueryKey(userId),
        });
        void queryClient.invalidateQueries({
          queryKey: getGetAdminUsersQueryKey(),
        });
        onClose();
      },
      onError: (error) => setFormError(applyApiError(form, error)),
    },
  });

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
    >
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Cabut hak membuat konten</DialogTitle>
          <DialogDescription>
            Pengguna tidak akan dapat membuat portfolio atau blog sampai haknya
            dipulihkan. Alasan dicatat di audit log.
          </DialogDescription>
        </DialogHeader>

        {formError ? (
          <Alert variant="destructive">
            <AlertTitle>Gagal mencabut hak</AlertTitle>
            <AlertDescription>{formError}</AlertDescription>
          </Alert>
        ) : null}

        <Form {...form}>
          <form
            id="block-content-form"
            className="space-y-4"
            onSubmit={form.handleSubmit((values) => {
              setFormError(null);
              blockMutation.mutate({
                id: userId,
                data: { reason: values.reason },
              });
            })}
          >
            <FormField
              control={form.control}
              name="reason"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Alasan</FormLabel>
                  <FormControl>
                    <Textarea
                      placeholder="Jelaskan alasan pencabutan hak…"
                      maxLength={300}
                      rows={4}
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>3–300 karakter.</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </form>
        </Form>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={onClose}
            disabled={blockMutation.isPending}
          >
            Batal
          </Button>
          <Button
            type="submit"
            form="block-content-form"
            variant="destructive"
            disabled={blockMutation.isPending}
          >
            {blockMutation.isPending ? "Memproses…" : "Cabut hak"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function UserActions({ user }: { user: AdminUserDetail }) {
  const queryClient = useQueryClient();
  const { session } = useSession();
  const [resetOpen, setResetOpen] = useState(false);
  const [deactivateOpen, setDeactivateOpen] = useState(false);
  const [reactivateOpen, setReactivateOpen] = useState(false);
  const [blockOpen, setBlockOpen] = useState(false);
  const [unblockOpen, setUnblockOpen] = useState(false);
  const [issued, setIssued] = useState<IssuedPasswordState | null>(null);

  const canReset = can(session, "user.reset_password");
  const canDeactivate = can(session, "user.deactivate");
  const canBlock = canGlobally(session, "user.block_content");
  const isSelf = session?.user.id === user.id;

  const invalidate = () => {
    void queryClient.invalidateQueries({
      queryKey: getGetAdminUserQueryKey(user.id),
    });
    void queryClient.invalidateQueries({
      queryKey: getGetAdminUsersQueryKey(),
    });
  };

  const resetMutation = usePostAdminUserResetPassword({
    mutation: {
      onSuccess: (data) => {
        setResetOpen(false);
        setIssued({ ...data, kind: "temporary" });
        toast.success("Sandi sementara diterbitkan");
        invalidate();
      },
      onError: (error) => {
        toast.error(
          apiErrorMessage(error, "Sandi sementara gagal diterbitkan."),
        );
      },
    },
  });

  const reissueMutation = usePostAdminUserReissueInitialPassword({
    mutation: {
      onSuccess: (data) => {
        setIssued({ ...data, kind: "initial" });
        toast.success("Sandi awal diterbitkan ulang");
        invalidate();
      },
      onError: (error) => {
        toast.error(apiErrorMessage(error, "Sandi awal gagal diterbitkan."));
      },
    },
  });

  const deactivateMutation = usePostAdminUserDeactivate({
    mutation: {
      onSuccess: () => {
        setDeactivateOpen(false);
        toast.success("Pengguna dinonaktifkan");
        invalidate();
      },
      onError: (error) => {
        toast.error(apiErrorMessage(error, "Pengguna gagal dinonaktifkan."));
      },
    },
  });

  const reactivateMutation = usePostAdminUserReactivate({
    mutation: {
      onSuccess: () => {
        setReactivateOpen(false);
        toast.success("Pengguna diaktifkan kembali");
        invalidate();
      },
      onError: (error) => {
        toast.error(apiErrorMessage(error, "Pengguna gagal diaktifkan."));
      },
    },
  });

  const unblockMutation = usePostAdminUserContentUnblock({
    mutation: {
      onSuccess: () => {
        setUnblockOpen(false);
        toast.success("Hak membuat konten dipulihkan");
        invalidate();
      },
      onError: (error) => {
        toast.error(apiErrorMessage(error, "Hak membuat konten gagal dipulihkan."));
      },
    },
  });

  return (
    <>
      <div className="flex flex-wrap gap-2">
        {canReset && !isSelf ? (
          user.activated ? (
            <Button
              type="button"
              variant="outline"
              onClick={() => setResetOpen(true)}
            >
              <KeyRound aria-hidden="true" />
              Reset sandi
            </Button>
          ) : (
            <Button
              type="button"
              variant="outline"
              onClick={() => reissueMutation.mutate({ id: user.id })}
              disabled={reissueMutation.isPending}
            >
              <KeyRound aria-hidden="true" />
              {reissueMutation.isPending
                ? "Memproses…"
                : "Terbitkan ulang sandi awal"}
            </Button>
          )
        ) : null}

        {canDeactivate ? (
          user.status === "inactive" ? (
            <Button
              type="button"
              variant="outline"
              onClick={() => setReactivateOpen(true)}
            >
              <UserCheck aria-hidden="true" />
              Aktifkan kembali
            </Button>
          ) : !isSelf ? (
            <Button
              type="button"
              variant="outline"
              onClick={() => setDeactivateOpen(true)}
            >
              <UserX aria-hidden="true" />
              Nonaktifkan
            </Button>
          ) : null
        ) : null}

        {canBlock ? (
          user.contentBlocked ? (
            <Button
              type="button"
              variant="outline"
              onClick={() => setUnblockOpen(true)}
            >
              <ShieldCheck aria-hidden="true" />
              Pulihkan hak membuat konten
            </Button>
          ) : (
            <Button
              type="button"
              variant="outline"
              onClick={() => setBlockOpen(true)}
            >
              <ShieldOff aria-hidden="true" />
              Cabut hak membuat konten
            </Button>
          )
        ) : null}
      </div>

      <AlertDialog open={resetOpen} onOpenChange={setResetOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Reset sandi pengguna?</AlertDialogTitle>
            <AlertDialogDescription>
              Sandi sementara baru akan diterbitkan dan seluruh sesi aktif
              pengguna langsung diakhiri. Pengguna wajib mengganti sandi saat
              masuk berikutnya.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={resetMutation.isPending}>
              Batal
            </AlertDialogCancel>
            <AlertDialogAction
              disabled={resetMutation.isPending}
              onClick={(event) => {
                event.preventDefault();
                resetMutation.mutate({ id: user.id });
              }}
            >
              {resetMutation.isPending ? "Memproses…" : "Reset sandi"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={deactivateOpen} onOpenChange={setDeactivateOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Nonaktifkan pengguna?</AlertDialogTitle>
            <AlertDialogDescription>
              Akun {user.displayName} akan dinonaktifkan dan seluruh sesinya
              dicabut. Pengguna tidak dapat masuk sampai akun diaktifkan
              kembali.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deactivateMutation.isPending}>
              Batal
            </AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={deactivateMutation.isPending}
              onClick={(event) => {
                event.preventDefault();
                deactivateMutation.mutate({ id: user.id });
              }}
            >
              {deactivateMutation.isPending ? "Memproses…" : "Nonaktifkan"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={reactivateOpen} onOpenChange={setReactivateOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Aktifkan kembali pengguna?</AlertDialogTitle>
            <AlertDialogDescription>
              Akun {user.displayName} akan dapat masuk kembali. Akun dengan
              tahun lulus akan kembali berstatus alumni.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={reactivateMutation.isPending}>
              Batal
            </AlertDialogCancel>
            <AlertDialogAction
              disabled={reactivateMutation.isPending}
              onClick={(event) => {
                event.preventDefault();
                reactivateMutation.mutate({ id: user.id });
              }}
            >
              {reactivateMutation.isPending
                ? "Memproses…"
                : "Aktifkan kembali"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={unblockOpen} onOpenChange={setUnblockOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Pulihkan hak membuat konten?</AlertDialogTitle>
            <AlertDialogDescription>
              Pengguna dapat kembali membuat portfolio dan blog.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={unblockMutation.isPending}>
              Batal
            </AlertDialogCancel>
            <AlertDialogAction
              disabled={unblockMutation.isPending}
              onClick={(event) => {
                event.preventDefault();
                unblockMutation.mutate({ id: user.id });
              }}
            >
              {unblockMutation.isPending ? "Memproses…" : "Pulihkan"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {blockOpen ? (
        <BlockContentDialog
          userId={user.id}
          onClose={() => setBlockOpen(false)}
        />
      ) : null}

      <Dialog
        open={issued !== null}
        onOpenChange={(open) => {
          if (!open) {
            setIssued(null);
          }
        }}
      >
        <DialogContent className="max-h-[90svh] overflow-y-auto sm:max-w-md">
          <DialogHeader>
            <DialogTitle>
              {issued?.kind === "initial"
                ? "Sandi awal diterbitkan"
                : "Sandi sementara diterbitkan"}
            </DialogTitle>
            <DialogDescription>
              {issued?.kind === "initial"
                ? `Sandi awal baru untuk ${user.displayName}.`
                : `Sandi sementara untuk ${user.displayName}. Semua sesi lama sudah diakhiri.`}
            </DialogDescription>
          </DialogHeader>

          {issued ? (
            <OneTimePassword
              password={issued.password}
              expiresAt={issued.expiresAt}
              label={
                issued.kind === "initial"
                  ? "Sandi awal"
                  : "Sandi sementara"
              }
            />
          ) : null}

          <DialogFooter>
            <Button onClick={() => setIssued(null)}>Selesai</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
