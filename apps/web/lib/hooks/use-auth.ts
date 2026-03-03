'use client';

import { useAuthStore } from '@/lib/stores/auth-store';
import { authApi, profileApi } from '@/lib/api';
import { useRouter } from 'next/navigation';
import { useMutation } from '@tanstack/react-query';
import { toast } from 'sonner';

export function useAuth() {
  const router = useRouter();
  const { user, isAuthenticated, isLoading, login: storeLogin, logout: storeLogout, setUser } = useAuthStore();

  const loginMutation = useMutation({
    mutationFn: async ({ username, password }: { username: string; password: string }) => {
      const response = await authApi.login({ username, password });
      if (!response.success || !response.data) {
        throw new Error('Login failed');
      }
      return response.data;
    },
    onSuccess: async (data) => {
      storeLogin(data.access_token, data.user);
      // Fetch full user data including special_roles and capabilities
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
    onError: (error: Error & { response?: { data?: { error?: { message?: string } } } }) => {
      const message = error.response?.data?.error?.message || 'Login gagal. Periksa username dan password.';
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
      // Even if API fails, logout locally
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
