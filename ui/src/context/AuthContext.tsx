import { createContext, useContext, useState, type ReactNode } from 'react'
import type { Role } from '../types'

interface AuthContextType {
  role: Role
  setRole: (role: Role) => void
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export function AuthProvider({ children, initialRole = 'Admin' }: { children: ReactNode; initialRole?: Role }) {
  const [role, setRole] = useState<Role>(initialRole)
  return <AuthContext.Provider value={{ role, setRole }}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContextType {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
