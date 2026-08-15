import { createContext, useContext, useState, ReactNode } from 'react'
import client from '../api/client'

interface User {
  id: string
  username: string
  email: string
}

interface AuthContextType {
  user: User | null
  token: string | null
  login: (email: string, password: string) => Promise<void>
  register: (username: string, email: string, password: string) => Promise<void>
  logout: () => void
  logoutAllDevices: () => Promise<void>
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setToken] = useState<string | null>(localStorage.getItem('token'))
  const [user, setUser] = useState<User | null>(
    JSON.parse(localStorage.getItem('user') || 'null'),
  )

  async function login(email: string, password: string) {
    const res = await client.post('/auth/login', { email, password })
    persist(res.data.token, res.data.user)
  }

  async function register(username: string, email: string, password: string) {
    const res = await client.post('/auth/register', { username, email, password })
    persist(res.data.token, res.data.user)
  }

  function persist(newToken: string, newUser: User) {
    localStorage.setItem('token', newToken)
    localStorage.setItem('user', JSON.stringify(newUser))
    setToken(newToken)
    setUser(newUser)
  }

  function logout() {
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    setToken(null)
    setUser(null)
  }

  // Backend'deki token_version'ı artırır — bu hesaba ait, tarayıcıda şu an
  // açık olan dahil, daha önce üretilmiş TÜM token'lar anında geçersiz
  // olur. "Başka bir cihazdan da giriş yapmıştım" ya da "hesabım ele
  // geçirilmiş olabilir" durumları için.
  async function logoutAllDevices() {
    try {
      await client.post('/auth/logout-all')
    } finally {
      logout()
    }
  }

  return (
    <AuthContext.Provider value={{ user, token, login, register, logout, logoutAllDevices }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
