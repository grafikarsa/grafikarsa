'use client';

import { useEffect, type ReactNode } from 'react';
import { useAuthStore } from '@/lib/stores/auth-store';
import { authApi, profileApi } from '@/lib/api';

export function AuthProvider({ children }: { children: ReactNode }) {
  const { accessToken, setUser, setLoading, logout } = useAuthStore();

  useEffect(() => {
    const initAuth = async () => {
      // Check if we have a token in the store (from persistence)
      const token = useAuthStore.getState().accessToken;

      if (token) {
        try {
          const response = await profileApi.getMe();
          if (response.success && response.data) {
            setUser(response.data);
          }
        } catch {
          // Token invalid, try refresh ONCE
          try {
            const refreshResponse = await authApi.refresh();
            if (refreshResponse.success && refreshResponse.data) {
              useAuthStore.getState().setAccessToken(refreshResponse.data.access_token);

              // Try fetching profile again with new token
              const meResponse = await profileApi.getMe();
              if (meResponse.success && meResponse.data) {
                setUser(meResponse.data);
              }
            } else {
              // Refresh failed, logout
              logout();
            }
          } catch {
            // Refresh error, logout
            logout();
          }
        }
      }
      setLoading(false);
    };

    initAuth();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []); // Only run on mount

  return <>{children}</>;
}
