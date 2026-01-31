import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { User, Organization } from "./api";

interface AuthState {
  token: string | null;
  user: User | null;
  organization: Organization | null;
  setAuth: (token: string, user: User, organization: Organization) => void;
  clearAuth: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      token: null,
      user: null,
      organization: null,
      setAuth: (token, user, organization) =>
        set({ token, user, organization }),
      clearAuth: () => set({ token: null, user: null, organization: null }),
    }),
    {
      name: "sentinel-auth",
    }
  )
);
