import { Link } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'

export default function NavBar() {
    const { user, logout, logoutAllDevices } = useAuth()

    async function handleLogoutAll() {
        if (!window.confirm('Tüm cihazlardaki oturumları kapatmak istediğine emin misin?')) return
        await logoutAllDevices()
    }

    return (
        <nav className="navbar navbar-baum px-3 py-3">
            <Link className="navbar-brand d-flex align-items-center gap-2" to="/">
                <span
                    className="d-flex align-items-center justify-content-center"
                    style={{
                        width: 40,
                        height: 40,
                        borderRadius: '50%',
                        background: '#ff8c1a',
                        border: '2px solid #ffffff',
                        flexShrink: 0,
                    }}
                >
                    <span
                        className="d-flex align-items-center justify-content-center"
                        style={{
                            width: 30,
                            height: 30,
                            borderRadius: '50%',
                            background: '#0a1f44',
                            color: '#ffffff',
                            fontSize: '0.6rem',
                            fontWeight: 700,
                        }}
                    >
                        MEÜ
                    </span>
                </span>
                <div>
                    <span
                        className="d-block"
                        style={{ fontSize: '0.68rem', opacity: 0.85, letterSpacing: '0.08em' }}
                    >
                        MERSİN ÜNİVERSİTESİ
                    </span>
                    <strong style={{ letterSpacing: '0.02em' }}>BAUM PDF OCR Servisi</strong>
                </div>
            </Link>

            {user && (
                <div className="d-flex align-items-center gap-3">
                    <span className="text-white fw-semibold" style={{ letterSpacing: '0.03em' }}>
                        {user.username.toUpperCase()}
                    </span>

                    <Link
                        className="btn btn-outline-light btn-sm text-uppercase"
                        style={{ letterSpacing: '0.05em', fontSize: '0.78rem' }}
                        to="/profile"
                    >
                        Profilim
                    </Link>

                    {user.role === 'admin' && (
                        <Link
                            className="btn btn-outline-light btn-sm text-uppercase"
                            style={{ letterSpacing: '0.05em', fontSize: '0.78rem' }}
                            to="/admin"
                        >
                            Yönetici Paneli
                        </Link>
                    )}

                    <button
                        className="btn btn-light btn-sm text-uppercase"
                        style={{ letterSpacing: '0.05em', fontSize: '0.78rem' }}
                        onClick={logout}
                        type="button"
                    >
                        Çıkış Yap
                    </button>

                    <button
                        className="btn btn-outline-light btn-sm text-uppercase"
                        style={{ letterSpacing: '0.05em', fontSize: '0.78rem' }}
                        onClick={handleLogoutAll}
                        type="button"
                    >
                        Tüm Cihazlardan Çık
                    </button>
                </div>
            )}
        </nav>
    )
}