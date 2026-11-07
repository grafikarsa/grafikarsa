"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
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
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { postAuthLogin } from "@/lib/api/gen/endpoints";
import { ApiError } from "@/lib/api/fetcher";
import {
  requiredPathFor,
  sessionQueryKey,
  useSession,
} from "@/lib/auth/session";

const loginSchema = z.object({
  identifier: z
    .string()
    .trim()
    .min(1, "NISN/NIP/username wajib diisi.")
    .max(64, "Maksimal 64 karakter."),
  password: z
    .string()
    .min(1, "Sandi wajib diisi.")
    .max(128, "Maksimal 128 karakter."),
});

type LoginValues = z.infer<typeof loginSchema>;

function safeReturnPath(value: string | null): string {
  if (value && value.startsWith("/") && !value.startsWith("//")) {
    return value;
  }

  return "/";
}

function loginErrorMessage(error: unknown): string | null {
  if (!error) {
    return null;
  }

  if (error instanceof ApiError) {
    switch (error.problem?.code) {
      case "INVALID_CREDENTIALS":
        return "NISN/NIP/username atau sandi salah.";
      case "RATE_LIMITED":
        return "Terlalu banyak percobaan. Coba lagi beberapa menit lagi.";
      case "ACCOUNT_INACTIVE":
      case "PASSWORD_EXPIRED":
        return error.problem?.title ?? "Akun tidak dapat digunakan.";
      default:
        return "Terjadi kesalahan. Coba lagi.";
    }
  }

  return "Terjadi kesalahan. Coba lagi.";
}

export function LoginForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const queryClient = useQueryClient();
  const { session, status } = useSession();
  const returnPath = safeReturnPath(searchParams.get("kembali"));

  const form = useForm<LoginValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      identifier: "",
      password: "",
    },
  });

  const loginMutation = useMutation({
    mutationFn: (values: LoginValues) => postAuthLogin(values),
    onSuccess: (data) => {
      queryClient.setQueryData(sessionQueryKey, data);
      router.replace(requiredPathFor(data.nextStep) ?? returnPath);
    },
  });

  useEffect(() => {
    if (status === "authenticated" && session.nextStep === "none") {
      router.replace(returnPath);
    }
  }, [returnPath, router, session, status]);

  const errorMessage = loginErrorMessage(loginMutation.error);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-xl">Masuk</CardTitle>
        <CardDescription>
          Gunakan NISN, NIP/ID staf, atau username.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {errorMessage ? (
          <Alert variant="destructive">
            <AlertTitle>Tidak bisa masuk</AlertTitle>
            <AlertDescription>{errorMessage}</AlertDescription>
          </Alert>
        ) : null}

        <Form {...form}>
          <form
            onSubmit={form.handleSubmit((values) => loginMutation.mutate(values))}
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
              name="password"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Sandi</FormLabel>
                  <FormControl>
                    <PasswordInput
                      autoComplete="current-password"
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
              disabled={loginMutation.isPending}
            >
              {loginMutation.isPending ? "Memproses…" : "Masuk"}
            </Button>
          </form>
        </Form>

        <p className="text-center text-sm text-muted-foreground">
          <Link
            href="/lupa-sandi"
            className="font-medium text-foreground underline-offset-4 hover:underline"
          >
            Lupa sandi?
          </Link>
        </p>
      </CardContent>
    </Card>
  );
}
