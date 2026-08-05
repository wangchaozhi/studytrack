import { create } from 'zustand'
import type { User } from '../types'

interface AuthState {
  user: User | null
  hydrated: boolean
  setUser: (user: User | null) => void
  setToken: (token: string) => void
  logout: () => void
  setHydrated: (value: boolean) => void
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  hydrated: false,
  setUser: (user) => set({ user }),
  setToken: (token) => localStorage.setItem('studytrack_token', token),
  logout: () => {
    localStorage.removeItem('studytrack_token')
    set({ user: null })
  },
  setHydrated: (hydrated) => set({ hydrated }),
}))
