import { create } from "zustand";
import { persist } from "zustand/middleware";
import { useEffect, useState } from "react";
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

// Hook to handle hydration - returns true once client-side state is ready
export const useHydration = () => {
  const [hydrated, setHydrated] = useState(false);

  useEffect(() => {
    const unsubscribe = useAuthStore.persist.onFinishHydration(() => {
      setHydrated(true);
    });

    // Check if already hydrated
    if (useAuthStore.persist.hasHydrated()) {
      setHydrated(true);
    }

    return () => {
      unsubscribe();
    };
  }, []);

  return hydrated;
};
