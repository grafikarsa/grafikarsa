"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import Link from "next/link";
import { useState } from "react";
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
import { Input } from "@/components/ui/input";
import { postAuthPasswordRecover } from "@/lib/api/gen/endpoints";
import { ApiError } from "@/lib/api/fetcher";

const recoverSchema = z
  .object({
    identifier: z
      .string()
      .trim()
      .min(1, "NISN/NIP/username wajib diisi.")
      .max(64, "Maksimal 64 karakter."),
    recoveryCode: z
      .string()
      .trim()
      .min(10, "Kode pemulihan tidak valid.")
      .max(20, "Kode pemulihan tidak valid."),
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

type RecoverValues = z.infer<typeof recoverSchema>;

function recoverErrorMessage(error: unknown): string | null {
  if (!error) {
    return null;
  }

  if (error instanceof ApiError) {
    switch (error.problem?.code) {
      case "INVALID_CREDENTIALS":
      case "VALIDATION_FAILED":
        return "Identitas atau kode pemulihan salah.";
      case "RATE_LIMITED":
        return "Terlalu banyak percobaan. Coba lagi beberapa menit lagi.";
      default:
        return "Terjadi kesalahan. Coba lagi.";
    }
  }

  return "Terjadi kesalahan. Coba lagi.";
}

export function RecoverPasswordForm() {
  const [recovered, setRecovered] = useState(false);

  const form = useForm<RecoverValues>({
    resolver: zodResolver(recoverSchema),
    defaultValues: {
      identifier: "",
      recoveryCode: "",
      newPassword: "",
      confirmPassword: "",
    },
  });

  const recoverMutation = useMutation({
    mutationFn: (values: RecoverValues) =>
      postAuthPasswordRecover({
        identifier: values.identifier,
        recoveryCode: values.recoveryCode,
        newPassword: values.newPassword,
      }),
    onSuccess: () => setRecovered(true),
  });

  if (recovered) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-xl">Sandi berhasil diganti</CardTitle>
          <CardDescription>
            Sandi berhasil diganti. Silakan masuk kembali.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Button asChild className="w-full" size="lg">
            <Link href="/masuk">Masuk</Link>
          </Button>
        </CardContent>
      </Card>
    );
  }

  const errorMessage = recoverErrorMessage(recoverMutation.error);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-xl">Lupa sandi</CardTitle>
        <CardDescription>
          Atur ulang sandi dengan salah satu kode pemulihan yang kamu simpan
          saat aktivasi.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {errorMessage ? (
          <Alert variant="destructive">
            <AlertTitle>Tidak bisa mengatur ulang sandi</AlertTitle>
            <AlertDescription>{errorMessage}</AlertDescription>
          </Alert>
        ) : null}

        <Form {...form}>
          <form
            onSubmit={form.handleSubmit((values) =>
              recoverMutation.mutate(values),
            )}
            className="space-y-4"
          >
            <FormField
              control={form.control}
              name="identifier"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>NISN / NIP / username</FormLabel>
                  <FormControl>
                    <Input
                      autoComplete="username"
                      autoCapitalize="none"
                      spellCheck={false}
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="recoveryCode"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Kode pemulihan</FormLabel>
                  <FormControl>
                    <Input
                      autoComplete="off"
                      autoCapitalize="none"
                      spellCheck={false}
                      placeholder="xxxxx-xxxxx"
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>Format: xxxxx-xxxxx</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

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
              disabled={recoverMutation.isPending}
            >
              {recoverMutation.isPending ? "Memproses…" : "Atur ulang sandi"}
            </Button>
          </form>
        </Form>

        <p className="text-sm text-muted-foreground">
          Kode pemulihan hilang? Minta wali kelas atau admin sekolah untuk
          mereset sandimu.
        </p>

        <p className="text-center text-sm">
          <Link
            href="/masuk"
            className="font-medium text-foreground underline-offset-4 hover:underline"
          >
            Kembali ke halaman masuk
          </Link>
        </p>
      </CardContent>
    </Card>
  );
}
