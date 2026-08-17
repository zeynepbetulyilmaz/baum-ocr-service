import { FormEvent, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'

function EyeIcon({ off }: { off: boolean }) {
  return off ? (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="M17.94 17.94A10.94 10.94 0 0 1 12 20c-7 0-11-8-11-8a18.9 18.9 0 0 1 5.06-5.94M9.9 4.24A10.94 10.94 0 0 1 12 4c7 0 11 8 11 8a18.9 18.9 0 0 1-2.16 3.19M14.12 14.12a3 3 0 1 1-4.24-4.24" />
      <line x1="1" y1="1" x2="23" y2="23" />
    </svg>
  ) : (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8Z" />
      <circle cx="12" cy="12" r="3" />
    </svg>
  )
}

const USERNAME_PATTERN = /^[a-zA-Z0-9_çğıöşüÇĞİÖŞÜ]+$/

function passwordStrength(pw: string): { ok: boolean; message: string } {
  if (pw.length < 6) return { ok: false, message: 'Şifre en az 6 karakter olmalı' }
  const hasLetter = /[a-zA-ZçğıöşüÇĞİÖŞÜ]/.test(pw)
  const hasDigit = /[0-9]/.test(pw)
  if (!hasLetter || !hasDigit) {
    return { ok: false, message: 'Şifre en az bir harf ve bir rakam içermeli' }
  }
  return { ok: true, message: '' }
}

export default function Register() {
  const { register } = useAuth()
  const navigate = useNavigate()
  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [consent, setConsent] = useState(false)
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')

    const trimmedUsername = username.trim()
    if (!USERNAME_PATTERN.test(trimmedUsername)) {
      setError('Kullanıcı adı sadece harf, rakam ve alt çizgi (_) içerebilir')
      return
    }

    const strength = passwordStrength(password)
    if (!strength.ok) {
      setError(strength.message)
      return
    }

    if (password !== confirmPassword) {
      setError('Şifreler eşleşmiyor')
      return
    }

    if (!consent) {
      setError('Devam etmek için gizlilik ve veri kullanımı onayını işaretlemelisin')
      return
    }

    setSubmitting(true)
    try {
      await register(trimmedUsername, email, password)
      navigate('/')
    } catch (err: any) {
      setError(err.response?.data?.error || 'Kayıt başarısız')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="auth-shell">
      <div className="auth-card">
        <span className="auth-eyebrow">Mersin Üniversitesi</span>
        <h3 className="mb-3">Kayıt Ol</h3>
        {error && (
          <div className="alert alert-danger" role="alert" aria-live="assertive">
            {error}
          </div>
        )}
        <form onSubmit={handleSubmit}>
          <div className="mb-3">
            <label htmlFor="register-username" className="form-label">
              Kullanıcı Adı
            </label>
            <input
              id="register-username"
              className="form-control"
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              minLength={3}
              maxLength={32}
              required
            />
          </div>
          <div className="mb-3">
            <label htmlFor="register-email" className="form-label">
              E-posta
            </label>
            <input
              id="register-email"
              className="form-control"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
            />
          </div>
          <div className="mb-3">
            <label htmlFor="register-password" className="form-label">
              Şifre
            </label>
            <div className="input-group">
              <input
                id="register-password"
                className="form-control"
                type={showPassword ? 'text' : 'password'}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                minLength={6}
                required
              />
              <button
                type="button"
                className="btn btn-outline-secondary"
                onClick={() => setShowPassword((v) => !v)}
                aria-label={showPassword ? 'Şifreyi gizle' : 'Şifreyi göster'}
                tabIndex={-1}
              >
                <EyeIcon off={showPassword} />
              </button>
            </div>
            <div className="form-text">En az 6 karakter, en az bir harf ve bir rakam içermeli.</div>
          </div>

          <div className="mb-3">
            <label htmlFor="register-confirm-password" className="form-label">
              Şifre (Tekrar)
            </label>
            <input
              id="register-confirm-password"
              className="form-control"
              type={showPassword ? 'text' : 'password'}
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              required
            />
          </div>

          <div className="mb-3 form-check">
            <input
              id="register-consent"
              className="form-check-input"
              type="checkbox"
              checked={consent}
              onChange={(e) => setConsent(e.target.checked)}
              required
            />
            <label htmlFor="register-consent" className="form-check-label" style={{ fontSize: '0.85rem' }}>
              Kişisel verilerimin (kullanıcı adı, e-posta, yüklediğim belgeler) bu servisi
              işletmek amacıyla saklanmasını kabul ediyorum. Hesabımı ve tüm verilerimi
              istediğim zaman Profilim sayfasından kalıcı olarak silebileceğimi biliyorum.
            </label>
          </div>

          <button className="btn btn-primary w-100" type="submit" disabled={submitting}>
            {submitting ? 'Kayıt olunuyor...' : 'Kayıt Ol'}
          </button>
        </form>
        <p className="mt-3 mb-0">
          Zaten hesabın var mı? <Link to="/login">Giriş yap</Link>
        </p>
      </div>
    </div>
  )
}
