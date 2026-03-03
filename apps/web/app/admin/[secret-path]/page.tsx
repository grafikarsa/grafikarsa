'use client';

import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useMutation } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { FloatingPaths } from '@/components/floating-paths';
import { authApi } from '@/lib/api';
import { useAuthStore } from '@/lib/stores/auth-store';
import { ChevronLeft, Loader2, Shield, User } from 'lucide-react';

const loginSchema = z.object({
  username: z.string().min(1, 'Username atau email wajib diisi'),
  password: z.string().min(1, 'Password wajib diisi'),
});

type LoginFormData = z.infer<typeof loginSchema>;

export default function AdminLoginPage() {
  const router = useRouter();
  const { login } = useAuthStore();

  const form = useForm<LoginFormData>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      username: '',
      password: '',
    },
  });

  const loginMutation = useMutation({
    mutationFn: (data: LoginFormData) => authApi.login(data),
    onSuccess: (response) => {
      if (response.data) {
        const { access_token, user } = response.data;
        if (user.role !== 'admin') {
          toast.error('Akses ditolak. Hanya admin yang dapat login.');
          return;
        }
        login(access_token, user);
        toast.success('Login berhasil');
        router.push('/admin');
      }
    },
    onError: () => {
      toast.error('Email atau password salah');
    },
  });

  return (
    <main className="relative md:h-screen md:overflow-hidden lg:grid lg:grid-cols-2">
      {/* Left Panel - Decorative */}
      <div className="relative hidden h-full flex-col border-r bg-secondary p-10 lg:flex dark:bg-secondary/20">
        <div className="absolute inset-0 bg-gradient-to-b from-transparent via-transparent to-background" />
        <div className="mr-auto flex items-center gap-2">
          <Shield className="h-5 w-5 text-primary" />
          <span className="text-xl font-bold text-primary">Admin Panel</span>
        </div>
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
          <div className="flex items-center gap-2 lg:hidden">
            <Shield className="h-5 w-5 text-primary" />
            <span className="text-xl font-bold text-primary">Admin Panel</span>
          </div>

          <div className="flex flex-col space-y-1">
            <h1 className="text-2xl font-bold tracking-wide">Admin Login</h1>
            <p className="text-base text-muted-foreground">
              Masuk ke panel administrasi Grafikarsa
            </p>
          </div>

          <form
            onSubmit={form.handleSubmit((data) => loginMutation.mutate(data))}
            className="space-y-4"
          >
            <div className="space-y-2">
              <Label htmlFor="username">Username atau Email</Label>
              <div className="relative">
                <Input
                  id="username"
                  type="text"
                  placeholder="admin atau admin@grafikarsa.com"
                  className="pr-10"
                  {...form.register('username')}
                />
                <User className="absolute right-3 top-2.5 h-4 w-4 text-muted-foreground" />
              </div>
              {form.formState.errors.username && (
                <p className="text-sm text-destructive">
                  {form.formState.errors.username.message}
                </p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="password">Password</Label>
              <Input
                id="password"
                type="password"
                placeholder="Masukkan password"
                {...form.register('password')}
              />
              {form.formState.errors.password && (
                <p className="text-sm text-destructive">
                  {form.formState.errors.password.message}
                </p>
              )}
            </div>

            <Button type="submit" className="w-full" disabled={loginMutation.isPending}>
              {loginMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Masuk
            </Button>
          </form>
        </div>
      </div>
    </main>
  );
}
