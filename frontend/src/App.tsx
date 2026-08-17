import { JSX } from 'react'
import { Navigate, Route, Routes } from 'react-router-dom'
import { useAuth } from './context/AuthContext'
import Login from './pages/Login'
import Register from './pages/Register'
import ForgotPassword from './pages/ForgotPassword'
import ResetPassword from './pages/ResetPassword'
import Dashboard from './pages/Dashboard'
import JobDetail from './pages/JobDetail'
import Profile from './pages/Profile'
import AdminPanel from './pages/AdminPanel'
import NavBar from './components/NavBar'
import ErrorBoundary from './components/ErrorBoundary'

function PrivateRoute({ children }: { children: JSX.Element }) {
    const { token } = useAuth()
    return token ? children : <Navigate to="/login" replace />
}

function AdminRoute({ children }: { children: JSX.Element }) {
    const { token, user } = useAuth()
    if (!token) return <Navigate to="/login" replace />
    if (user?.role !== 'admin') return <Navigate to="/" replace />
    return children
}

export default function App() {
    return (
        <ErrorBoundary>
            <NavBar />
            <div className="container py-4">
                <Routes>
                    <Route path="/login" element={<Login />} />
                    <Route path="/register" element={<Register />} />
                    <Route path="/forgot-password" element={<ForgotPassword />} />
                    <Route path="/reset-password" element={<ResetPassword />} />
                    <Route
                        path="/"
                        element={
                            <PrivateRoute>
                                <Dashboard />
                            </PrivateRoute>
                        }
                    />
                    <Route
                        path="/documents/:id"
                        element={
                            <PrivateRoute>
                                <JobDetail />
                            </PrivateRoute>
                        }
                    />
                    <Route
                        path="/profile"
                        element={
                            <PrivateRoute>
                                <Profile />
                            </PrivateRoute>
                        }
                    />
                    <Route
                        path="/admin"
                        element={
                            <AdminRoute>
                                <AdminPanel />
                            </AdminRoute>
                        }
                    />
                </Routes>
            </div>
            <footer className="page-footer">
                Mersin Üniversitesi Bilgi İşlem Daire Başkanlığı — BAUM Yaz Staj 2026
            </footer>
        </ErrorBoundary>
    )
}