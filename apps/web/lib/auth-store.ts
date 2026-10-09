import { create } from "zustand";

type AuthState = {
  accessToken: string | null;
  csrfToken: string | null;
  user: { username: string; name: string; role_primary: string } | null;
  setAuth: (access: string, csrf: string, user: AuthState["user"]) => void;
  setAccess: (access: string) => void;
  logout: () => void;
};

export const useAuthStore = create<AuthState>((set) => ({
  accessToken: null,
  csrfToken: null,
  user: null,
  setAuth: (accessToken, csrfToken, user) => set({ accessToken, csrfToken, user }),
  setAccess: (accessToken) => set({ accessToken }),
  logout: () => set({ accessToken: null, csrfToken: null, user: null }),
}));
