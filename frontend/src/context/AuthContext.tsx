import { createContext, useContext, useState, ReactNode } from 'react'
import client from '../api/client'

interface User {
    id: string
    username: string
    email: string
    role: string
}

interface AuthContextType {
    user: User | null
    token: string | null
    login: (email: string, password: string) => Promise<void>
    register: (username: string, email: string, password: string) => Promise<void>
    logout: () => void
    logoutAllDevices: () => Promise<void>
    updateUser: (updates: Partial<User>) => void
    deleteMyAccount: () => Promise<void>
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

    async function logoutAllDevices() {
        try {
            await client.post('/auth/logout-all')
        } finally {
            logout()
        }
    }

    async function deleteMyAccount() {
        try {
            await client.delete('/me')
        } finally {
            logout()
        }
    }

    function updateUser(updates: Partial<User>) {
        setUser((prev) => {
            if (!prev) return prev
            const next = { ...prev, ...updates }
            localStorage.setItem('user', JSON.stringify(next))
            return next
        })
    }

    return (
        <AuthContext.Provider
            value={{ user, token, login, register, logout, logoutAllDevices, updateUser, deleteMyAccount }}
        >
            {children}
        </AuthContext.Provider>
    )
}

export function useAuth() {
    const ctx = useContext(AuthContext)
    if (!ctx) throw new Error('useAuth must be used within AuthProvider')
    return ctx
}