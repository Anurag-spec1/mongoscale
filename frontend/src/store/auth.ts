import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { AuthResponse, User } from '../types';

interface AuthState {
  token: string | null;
  user: User | null;
  setSession: (response: AuthResponse) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      token: null,
      user: null,
      setSession: (response) => set({ token: response.token, user: response.user }),
      logout: () => set({ token: null, user: null }),
    }),
    { name: 'mongoscale-auth' },
  ),
);

export function isAdmin(user: User | null): boolean {
  return user?.role === 'admin';
}
