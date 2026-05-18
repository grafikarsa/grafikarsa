'use client';

import { useAuthStore } from '@/lib/stores/auth-store';
import { authApi, profileApi } from '@/lib/api';
import { useRouter } from 'next/navigation';
import { useMutation } from '@tanstack/react-query';
import { toast } from 'sonner';
import axios from 'axios';

interface LoginInput {
  username: string;
  password: string;
  captcha_id?: string;
  captcha_answer?: number;
}

interface UseAuthOptions {
  onCaptchaRequired?: () => void;
}

function extractApiError(error: unknown): Error & { code?: string } {
  if (axios.isAxiosError(error) && error.response?.data) {
    const body = error.response.data as { error?: { code?: string; message?: string } };
    const err = new Error(body.error?.message || 'Terjadi kesalahan') as Error & { code?: string };
    err.code = body.error?.code;
    return err;
  }
  if (error instanceof Error) {
    return error as Error & { code?: string };
  }
  return new Error('Terjadi kesalahan') as Error & { code?: string };
}

export function useAuth(options?: UseAuthOptions) {
  const router = useRouter();
  const { user, isAuthenticated, isLoading, login: storeLogin, logout: storeLogout, setUser } = useAuthStore();

  const loginMutation = useMutation({
    mutationFn: async ({ username, password, captcha_id, captcha_answer }: LoginInput) => {
      const response = await authApi.login({ username, password, captcha_id, captcha_answer });
      if (!response.success || !response.data) {
        const err = new Error(response.error?.message || 'Login failed') as Error & { code?: string };
        err.code = response.error?.code;
        throw err;
      }
      return response.data;
    },
    onSuccess: async (data) => {
      storeLogin(data.access_token, data.user);
      try {
        const meResponse = await profileApi.getMe();
        if (meResponse.success && meResponse.data) {
          setUser(meResponse.data);
        }
      } catch {
        // Ignore error, basic user data is already set
      }
      toast.success('Login berhasil!');
      router.push('/');
    },
    onError: (rawError: unknown) => {
      const error = extractApiError(rawError);
      if (error.code === 'CAPTCHA_REQUIRED' || error.code === 'CAPTCHA_INVALID') {
        options?.onCaptchaRequired?.();
        return;
      }
      const message = error.message || 'Login gagal. Periksa username dan password.';
      toast.error(message);
    },
  });

  const logoutMutation = useMutation({
    mutationFn: async () => {
      await authApi.logout();
    },
    onSuccess: () => {
      storeLogout();
      toast.success('Logout berhasil');
      router.push('/');
    },
    onError: () => {
      storeLogout();
      router.push('/');
    },
  });

  return {
    user,
    isAuthenticated,
    isLoading,
    login: loginMutation.mutate,
    logout: logoutMutation.mutate,
    isLoginPending: loginMutation.isPending,
    isLogoutPending: logoutMutation.isPending,
  };
}
