"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Download, Printer } from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import { useForm, type UseFormReturn } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";

import { PasswordInput } from "@/components/auth/password-input";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import {
  postAuthActivate,
  postAuthRecoveryCodesConfirm,
  useGetAuthUsernameAvailability,
} from "@/lib/api/gen/endpoints";
import type { Session, UsernameAvailability } from "@/lib/api/gen/models";
import { ApiError } from "@/lib/api/fetcher";
import { sessionQueryKey, useSession } from "@/lib/auth/session";

const USERNAME_HELP =
  "Format tidak valid (3–30 karakter: huruf kecil, angka, titik, garis bawah)";

function buildActivationSchema(passwordOptional: boolean) {
  return z
    .object({
      username: z
        .string()
        .min(3, USERNAME_HELP)
        .max(30, USERNAME_HELP)
        .regex(/^[a-z0-9._]+$/, USERNAME_HELP),
      newPassword: passwordOptional
        ? z
            .string()
            .max(128, "Maksimal 128 karakter.")
            .refine(
              (value) => value === "" || value.length >= 8,
              "Minimal 8 karakter.",
            )
        : z
            .string()
            .min(8, "Minimal 8 karakter.")
            .max(128, "Maksimal 128 karakter."),
      confirmPassword: z.string(),
    })
    .refine((values) => values.newPassword === values.confirmPassword, {
      message: "Konfirmasi sandi tidak cocok.",
      path: ["confirmPassword"],
    });
}

type ActivationValues = z.infer<ReturnType<typeof buildActivationSchema>>;

function useDebouncedValue<T>(value: T, delay: number): T {
  const [debounced, setDebounced] = useState(value);

  useEffect(() => {
    const timeout = window.setTimeout(() => setDebounced(value), delay);

    return () => window.clearTimeout(timeout);
  }, [value, delay]);

  return debounced;
}

function applyFieldErrors(
  error: unknown,
  form: UseFormReturn<ActivationValues>,
) {
  if (!(error instanceof ApiError) || error.problem?.code !== "VALIDATION_FAILED") {
    return;
  }

  for (const fieldError of error.problem.errors ?? []) {
    if (fieldError.field === "username" || fieldError.field === "newPassword") {
      form.setError(fieldError.field, {
        type: "server",
        message: fieldError.message,
      });
    }
  }
}

function activationErrorMessage(error: unknown): string | null {
  if (!error) {
    return null;
  }

  if (error instanceof ApiError) {
    if (
      error.problem?.code === "VALIDATION_FAILED" &&
      (error.problem.errors?.length ?? 0) > 0
    ) {
      return null;
    }

    return error.problem?.title ?? "Terjadi kesalahan. Coba lagi.";
  }

  return "Terjadi kesalahan. Coba lagi.";
}

function availabilityText(
  data: UsernameAvailability | undefined,
  checking: boolean,
): { message: string; className: string } | null {
  if (checking) {
    return {
      message: "Memeriksa ketersediaan…",
      className: "text-muted-foreground",
    };
  }

  if (!data) {
    return null;
  }

  if (data.available) {
    return { message: "Tersedia", className: "text-emerald-600" };
  }

  switch (data.reason) {
    case "taken":
      return { message: "Sudah dipakai", className: "text-destructive" };
    case "invalid_format":
      return { message: USERNAME_HELP, className: "text-destructive" };
    case "reserved":
      return { message: "Tidak boleh dipakai", className: "text-destructive" };
    default:
      return { message: "Tidak boleh dipakai", className: "text-destructive" };
  }
}

export function ActivationFlow() {
  const router = useRouter();
  const { session, status } = useSession();
  const [codes, setCodes] = useState<string[] | null>(null);

  useEffect(() => {
    if (status === "anonymous") {
      router.replace("/masuk");
      return;
    }

    if (status === "authenticated" && session.nextStep !== "activate") {
      router.replace("/");
    }
  }, [router, session, status]);

  if (status !== "authenticated" || session.nextStep !== "activate") {
    return (
      <div className="space-y-4 rounded-xl p-4 ring-1 ring-foreground/10">
        <Skeleton className="h-6 w-40" />
        <Skeleton className="h-4 w-64" />
        <Skeleton className="h-8 w-full" />
        <Skeleton className="h-8 w-full" />
        <Skeleton className="h-9 w-full" />
      </div>
    );
  }

  if (codes) {
    return (
      <RecoveryCodesStep
        codes={codes}
        onConfirmed={() => {
          setCodes(null);
          router.replace("/");
        }}
      />
    );
  }

  return (
    <ActivationForm
      session={session}
      onActivated={(recoveryCodes) => setCodes(recoveryCodes)}
    />
  );
}

function ActivationForm({
  session,
  onActivated,
}: {
  session: Session;
  onActivated: (codes: string[]) => void;
}) {
  const isMigratedAccount = session.isMigratedAccount ?? false;
  const schema = useMemo(
    () => buildActivationSchema(isMigratedAccount),
    [isMigratedAccount],
  );

  const form = useForm<ActivationValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      username: session.user.username ?? "",
      newPassword: "",
      confirmPassword: "",
    },
  });

  const username = form.watch("username");
  const normalizedUsername = username.trim().toLowerCase();
  const debouncedUsername = useDebouncedValue(normalizedUsername, 400);
  const availabilityQuery = useGetAuthUsernameAvailability(
    { u: debouncedUsername },
    {
      query: {
        enabled: debouncedUsername.length >= 3,
        retry: false,
      },
    },
  );

  const checkingAvailability =
    debouncedUsername !== normalizedUsername ||
    availabilityQuery.isFetching;

  const availability = availabilityText(
    availabilityQuery.data,
    checkingAvailability,
  );

  const activationMutation = useMutation({
    mutationFn: (values: ActivationValues) =>
      postAuthActivate({
        username: values.username,
        ...(values.newPassword ? { newPassword: values.newPassword } : {}),
      }),
    onSuccess: (data) => onActivated(data.codes),
    onError: (error) => applyFieldErrors(error, form),
  });

  const errorMessage = activationErrorMessage(activationMutation.error);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-xl">Aktivasi akun</CardTitle>
        <CardDescription>
          Pilih username dan atur sandi baru sebelum mulai memakai Grafikarsa.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {errorMessage ? (
          <Alert variant="destructive">
            <AlertTitle>Aktivasi gagal</AlertTitle>
            <AlertDescription>{errorMessage}</AlertDescription>
          </Alert>
        ) : null}

        <Form {...form}>
          <form
            onSubmit={form.handleSubmit((values) =>
              activationMutation.mutate(values),
            )}
            className="space-y-4"
          >
            <FormField
              control={form.control}
              name="username"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Username</FormLabel>
                  <FormControl>
                    <Input
                      autoComplete="username"
                      autoCapitalize="none"
                      spellCheck={false}
                      maxLength={30}
                      placeholder="mis. budi.santoso"
                      {...field}
                      onChange={(event) =>
                        field.onChange(event.target.value.toLowerCase())
                      }
                    />
                  </FormControl>
                  {normalizedUsername.length >= 3 ? (
                    availability ? (
                      <p
                        className={`text-sm ${availability.className}`}
                        aria-live="polite"
                      >
                        {availability.message}
                      </p>
                    ) : null
                  ) : (
                    <FormDescription>{USERNAME_HELP}</FormDescription>
                  )}
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="newPassword"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {isMigratedAccount ? "Sandi baru (opsional)" : "Sandi baru"}
                  </FormLabel>
                  <FormControl>
                    <PasswordInput
                      autoComplete="new-password"
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    Minimal 8 karakter. Gunakan kombinasi huruf, angka, dan
                    simbol, serta hindari sandi yang mudah ditebak.
                    {isMigratedAccount
                      ? " Kosongkan jika tetap memakai sandi lama."
                      : ""}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="confirmPassword"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Ulangi sandi baru</FormLabel>
                  <FormControl>
                    <PasswordInput
                      autoComplete="new-password"
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <Button
              type="submit"
              className="w-full"
              size="lg"
              disabled={activationMutation.isPending}
            >
              {activationMutation.isPending
                ? "Memproses…"
                : "Aktifkan akun"}
            </Button>
          </form>
        </Form>
      </CardContent>
    </Card>
  );
}

function RecoveryCodesStep({
  codes,
  onConfirmed,
}: {
  codes: string[];
  onConfirmed: () => void;
}) {
  const queryClient = useQueryClient();
  const [confirmed, setConfirmed] = useState(false);

  const confirmMutation = useMutation({
    mutationFn: () => postAuthRecoveryCodesConfirm(),
    onSuccess: (data) => {
      queryClient.setQueryData(sessionQueryKey, data);
      onConfirmed();
    },
  });

  const copyAll = async () => {
    try {
      await navigator.clipboard.writeText(codes.join("\n"));
      toast.success("Kode pemulihan disalin.");
    } catch {
      toast.error("Tidak bisa menyalin. Salin kode secara manual.");
    }
  };

  const download = () => {
    const content = [
      "Kode pemulihan Grafikarsa",
      "Simpan di tempat aman. Setiap kode hanya bisa dipakai sekali.",
      "",
      ...codes,
      "",
    ].join("\n");
    const blob = new Blob([content], { type: "text/plain;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");

    link.href = url;
    link.download = "grafikarsa-recovery-codes.txt";
    link.click();
    URL.revokeObjectURL(url);
  };

  const errorMessage =
    confirmMutation.error instanceof ApiError
      ? (confirmMutation.error.problem?.title ??
        "Terjadi kesalahan. Coba lagi.")
      : confirmMutation.error
        ? "Terjadi kesalahan. Coba lagi."
        : null;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-xl">Simpan kode pemulihan</CardTitle>
        <CardDescription>
          Kode pemulihan dipakai untuk mengatur ulang sandi tanpa email. Kode
          ini hanya ditampilkan sekali dan setiap kode hanya bisa dipakai
          sekali.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {errorMessage ? (
          <Alert variant="destructive">
            <AlertTitle>Gagal menyelesaikan aktivasi</AlertTitle>
            <AlertDescription>{errorMessage}</AlertDescription>
          </Alert>
        ) : null}

        <div className="grid grid-cols-2 gap-2 rounded-lg border bg-muted/40 p-3 font-mono text-sm">
          {codes.map((code) => (
            <span key={code} className="text-center tracking-wider">
              {code}
            </span>
          ))}
        </div>

        <div className="flex flex-wrap gap-2">
          <Button type="button" variant="outline" size="sm" onClick={copyAll}>
            Salin semua
          </Button>
          <Button type="button" variant="outline" size="sm" onClick={download}>
            <Download aria-hidden="true" />
            Unduh .txt
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => window.print()}
          >
            <Printer aria-hidden="true" />
            Cetak
          </Button>
        </div>

        <div className="flex items-start gap-2">
          <Checkbox
            id="konfirmasi-kode-pemulihan"
            checked={confirmed}
            onCheckedChange={(value) => setConfirmed(value === true)}
          />
          <Label
            htmlFor="konfirmasi-kode-pemulihan"
            className="leading-snug font-normal"
          >
            Saya sudah menyimpan kode pemulihan ini di tempat aman
          </Label>
        </div>

        <Button
          type="button"
          className="w-full"
          size="lg"
          disabled={!confirmed || confirmMutation.isPending}
          onClick={() => confirmMutation.mutate()}
        >
          {confirmMutation.isPending ? "Menyimpan…" : "Lanjutkan"}
        </Button>
      </CardContent>
    </Card>
  );
}
