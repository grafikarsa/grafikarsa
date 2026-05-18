'use client';

import { useEffect, useState, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useAuthStore } from '@/lib/stores/auth-store';
import { useAuth } from '@/lib/hooks/use-auth';
import { authApi } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { FloatingPaths } from '@/components/floating-paths';
import { ChevronLeft, Loader2, User, ShieldAlert, RefreshCw } from 'lucide-react';

const loginSchema = z.object({
  username: z.string().min(1, 'Username wajib diisi'),
  password: z.string().min(1, 'Password wajib diisi'),
  captcha_answer: z.string().min(1, 'Jawaban CAPTCHA wajib diisi').optional(),
});

type LoginFormValues = z.infer<typeof loginSchema>;

export default function LoginPage() {
  const router = useRouter();
  const { isAuthenticated, isLoading } = useAuthStore();

  const [captchaId, setCaptchaId] = useState<string>('');
  const [captchaQuestion, setCaptchaQuestion] = useState<string>('');
  const [showCaptcha, setShowCaptcha] = useState(false);

  const form = useForm<LoginFormValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      username: '',
      password: '',
      captcha_answer: '',
    },
  });

  const fetchCaptcha = useCallback(async () => {
    try {
      const response = await authApi.getCaptcha();
      if (response.success && response.data) {
        setCaptchaId(response.data.id);
        setCaptchaQuestion(response.data.question);
        setShowCaptcha(true);
      }
    } catch {
      // If captcha fetch fails, still allow login attempt
    }
  }, []);

  useEffect(() => {
    if (showCaptcha) {
      form.setValue('captcha_answer', '');
    }
  }, [showCaptcha, form]);

  const { login, isLoginPending } = useAuth({
    onCaptchaRequired: fetchCaptcha,
  });

  useEffect(() => {
    if (!isLoading && isAuthenticated) {
      router.push('/');
    }
  }, [isAuthenticated, isLoading, router]);

  if (isLoading || isAuthenticated) {
    return null;
  }

  const onSubmit = (data: LoginFormValues) => {
    const payload: { username: string; password: string; captcha_id?: string; captcha_answer?: number } = {
      username: data.username,
      password: data.password,
    };

    if (showCaptcha && captchaId && data.captcha_answer) {
      payload.captcha_id = captchaId;
      payload.captcha_answer = parseInt(data.captcha_answer, 10);
    }

    login(payload);
  };

  return (
    <main className="relative md:h-screen md:overflow-hidden lg:grid lg:grid-cols-2">
      {/* Left Panel - Decorative */}
      <div className="relative hidden h-full flex-col border-r bg-secondary p-10 lg:flex dark:bg-secondary/20">
        <div className="absolute inset-0 bg-gradient-to-b from-transparent via-transparent to-background" />
        <Link href="/" className="mr-auto">
          <span className="text-xl font-bold text-primary">Grafikarsa</span>
        </Link>
        <div className="absolute inset-0">
          <FloatingPaths position={1} />
          <FloatingPaths position={-1} />
        </div>
      </div>

      {/* Right Panel - Login Form */}
      <div className="relative flex min-h-screen flex-col justify-center p-4">
        <div
          aria-hidden
          className="absolute inset-0 -z-10 isolate opacity-60 contain-strict"
        >
          <div className="absolute right-0 top-0 h-320 w-140 -translate-y-87.5 rounded-full bg-[radial-gradient(68.54%_68.72%_at_55.02%_31.46%,--theme(--color-foreground/.06)_0,hsla(0,0%,55%,.02)_50%,--theme(--color-foreground/.01)_80%)]" />
          <div className="absolute right-0 top-0 h-320 w-60 rounded-full bg-[radial-gradient(50%_50%_at_50%_50%,--theme(--color-foreground/.04)_0,--theme(--color-foreground/.01)_80%,transparent_100%)] [translate:5%_-50%]" />
          <div className="absolute right-0 top-0 h-320 w-60 -translate-y-87.5 rounded-full bg-[radial-gradient(50%_50%_at_50%_50%,--theme(--color-foreground/.04)_0,--theme(--color-foreground/.01)_80%,transparent_100%)]" />
        </div>

        <Button asChild className="absolute left-5 top-7" variant="ghost">
          <Link href="/">
            <ChevronLeft className="mr-1 h-4 w-4" />
            Beranda
          </Link>
        </Button>

        <div className="mx-auto w-full max-w-sm space-y-6">
          <Link href="/" className="lg:hidden">
            <span className="text-xl font-bold text-primary">Grafikarsa</span>
          </Link>

          <div className="flex flex-col space-y-1">
            <h1 className="text-2xl font-bold tracking-wide">Login</h1>
            <p className="text-base text-muted-foreground">
              Masuk ke akun Grafikarsa kamu
            </p>
          </div>

          {showCaptcha && (
            <div className="flex items-center gap-2 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-900 dark:bg-amber-950 dark:text-amber-200">
              <ShieldAlert className="h-4 w-4 shrink-0" />
              <span>Verifikasi CAPTCHA diperlukan. Terlalu banyak percobaan login.</span>
            </div>
          )}

          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
              <FormField
                control={form.control}
                name="username"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Username</FormLabel>
                    <FormControl>
                      <div className="relative">
                        <Input
                          placeholder="Masukkan username"
                          className="pr-10"
                          {...field}
                        />
                        <User className="absolute right-3 top-2.5 h-4 w-4 text-muted-foreground" />
                      </div>
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
                    <FormLabel>Password</FormLabel>
                    <FormControl>
                      <Input
                        type="password"
                        placeholder="Masukkan password"
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {showCaptcha && captchaQuestion && (
                <FormField
                  control={form.control}
                  name="captcha_answer"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Verifikasi</FormLabel>
                      <div className="flex items-center gap-2">
                        <span className="rounded-md bg-secondary px-3 py-2 text-sm font-medium tabular-nums">
                          {captchaQuestion}
                        </span>
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon"
                          className="h-8 w-8 shrink-0"
                          onClick={fetchCaptcha}
                        >
                          <RefreshCw className="h-4 w-4" />
                        </Button>
                      </div>
                      <FormControl>
                        <Input
                          type="text"
                          inputMode="numeric"
                          pattern="[0-9]*"
                          placeholder="Jawaban"
                          maxLength={2}
                          className="mt-2 w-24"
                          {...field}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              )}

              <Button type="submit" className="w-full" disabled={isLoginPending}>
                {isLoginPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                Masuk
              </Button>
            </form>
          </Form>
        </div>
      </div>
    </main>
  );
}
