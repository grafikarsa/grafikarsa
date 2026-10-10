import { create } from "zustand";

export type AuthUser = { id: string; username: string; name: string; role_primary: string };

type AuthState = {
  accessToken: string | null;
  csrfToken: string | null;
  user: AuthUser | null;
  isLoading: boolean;
  setSession: (access: string, csrf: string) => void;
  setUser: (user: AuthUser | null) => void;
  setLoading: (v: boolean) => void;
  logout: () => void;
};

// Access token hanya di memori (bukan localStorage). Refresh token ada di
// cookie __Host-rt HttpOnly yang diurus server/BFF.
export const useAuthStore = create<AuthState>((set) => ({
  accessToken: null,
  csrfToken: null,
  user: null,
  isLoading: true,
  setSession: (accessToken, csrfToken) => set({ accessToken, csrfToken }),
  setUser: (user) => set({ user, isLoading: false }),
  setLoading: (isLoading) => set({ isLoading }),
  logout: () => set({ accessToken: null, csrfToken: null, user: null, isLoading: false }),
}));
