"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { useForm } from "react-hook-form";
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
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Skeleton } from "@/components/ui/skeleton";
import { postAuthPasswordChange } from "@/lib/api/gen/endpoints";
import { ApiError } from "@/lib/api/fetcher";
import { sessionQueryKey, useSession } from "@/lib/auth/session";

const changeSchema = z
  .object({
    newPassword: z
      .string()
      .min(8, "Minimal 8 karakter.")
      .max(128, "Maksimal 128 karakter."),
    confirmPassword: z.string(),
  })
  .refine((values) => values.newPassword === values.confirmPassword, {
    message: "Konfirmasi sandi tidak cocok.",
    path: ["confirmPassword"],
  });

type ChangeValues = z.infer<typeof changeSchema>;

function changeErrorMessage(error: unknown): string | null {
  if (!error) {
    return null;
  }

  if (error instanceof ApiError) {
    return error.problem?.title ?? "Terjadi kesalahan. Coba lagi.";
  }

  return "Terjadi kesalahan. Coba lagi.";
}

export function ChangePasswordForm() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const { session, status } = useSession();

  const form = useForm<ChangeValues>({
    resolver: zodResolver(changeSchema),
    defaultValues: {
      newPassword: "",
      confirmPassword: "",
    },
  });

  const changeMutation = useMutation({
    mutationFn: (values: ChangeValues) =>
      postAuthPasswordChange({ newPassword: values.newPassword }),
    onSuccess: (data) => {
      queryClient.setQueryData(sessionQueryKey, data);
      router.replace("/");
    },
    onError: (error) => {
      if (
        error instanceof ApiError &&
        error.problem?.code === "VALIDATION_FAILED"
      ) {
        form.setError("newPassword", {
          type: "server",
          message:
            error.problem.errors?.find((item) => item.field === "newPassword")
              ?.message ?? "Sandi baru tidak valid.",
        });
      }
    },
  });

  useEffect(() => {
    if (status === "anonymous") {
      router.replace("/masuk");
      return;
    }

    if (
      status === "authenticated" &&
      session.nextStep !== "change_password"
    ) {
      router.replace("/");
    }
  }, [router, session, status]);

  if (status !== "authenticated" || session.nextStep !== "change_password") {
    return (
      <div className="space-y-4 rounded-xl p-4 ring-1 ring-foreground/10">
        <Skeleton className="h-6 w-40" />
        <Skeleton className="h-4 w-64" />
        <Skeleton className="h-8 w-full" />
        <Skeleton className="h-9 w-full" />
      </div>
    );
  }

  const errorMessage = changeErrorMessage(changeMutation.error);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-xl">Ganti sandi</CardTitle>
        <CardDescription>
          Kamu masuk dengan sandi sementara. Buat sandi baru untuk melanjutkan.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {errorMessage ? (
          <Alert variant="destructive">
            <AlertTitle>Tidak bisa mengganti sandi</AlertTitle>
            <AlertDescription>{errorMessage}</AlertDescription>
          </Alert>
        ) : null}

        <Form {...form}>
          <form
            onSubmit={form.handleSubmit((values) =>
              changeMutation.mutate(values),
            )}
            className="space-y-4"
          >
            <FormField
              control={form.control}
              name="newPassword"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Sandi baru</FormLabel>
                  <FormControl>
                    <PasswordInput
                      autoComplete="new-password"
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>Minimal 8 karakter.</FormDescription>
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
              disabled={changeMutation.isPending}
            >
              {changeMutation.isPending ? "Menyimpan…" : "Simpan sandi baru"}
            </Button>
          </form>
        </Form>

        <p className="text-sm text-muted-foreground">
          Setelah sandi diganti, semua sesi di perangkat lain akan keluar.
        </p>
      </CardContent>
    </Card>
  );
}
