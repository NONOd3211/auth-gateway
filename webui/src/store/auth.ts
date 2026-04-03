import { create } from 'zustand'

interface AuthState {
  password: string | null
  isAuthenticated: boolean
  login: (password: string) => void
  logout: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
  password: localStorage.getItem('admin_password'),
  isAuthenticated: !!localStorage.getItem('admin_password'),
  login: (password: string) => {
    localStorage.setItem('admin_password', password)
    set({ password, isAuthenticated: true })
  },
  logout: () => {
    localStorage.removeItem('admin_password')
    set({ password: null, isAuthenticated: false })
  },
}))
