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
      <Link className="navbar-brand" to="/">
        <div>
          <span className="d-block" style={{ fontSize: '0.7rem', opacity: 0.8 }}>
            MERSİN ÜNİVERSİTESİ
          </span>
          <strong>BAUM PDF OCR Servisi</strong>
        </div>
      </Link>
      {user && (
        <div className="d-flex align-items-center gap-3">
          <span className="text-white">{user.username}</span>
          <button className="btn btn-outline-light btn-sm" onClick={handleLogoutAll} type="button">
            Tüm Cihazlardan Çık
          </button>
          <button className="btn btn-light btn-sm" onClick={logout} type="button">
            Çıkış Yap
          </button>
        </div>
      )}
    </nav>
  )
}
