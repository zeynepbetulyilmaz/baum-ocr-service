import { FormEvent, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import client from '../api/client'

export default function ResetPassword() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const token = searchParams.get('token') || ''

  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState('')
  const [message, setMessage] = useState('')
  const [submitting, setSubmitting] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    setMessage('')

    if (!token) {
      setError('Bağlantı geçersiz — token eksik')
      return
    }
    if (newPassword.length < 6) {
      setError('Şifre en az 6 karakter olmalı')
      return
    }
    if (newPassword !== confirmPassword) {
      setError('Şifreler eşleşmiyor')
      return
    }

    setSubmitting(true)
    try {
      const res = await client.post('/auth/reset-password', { token, new_password: newPassword })
      setMessage(res.data.status)
      setTimeout(() => navigate('/login'), 2000)
    } catch (err: any) {
      setError(err.response?.data?.error || 'Şifre sıfırlanamadı')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="auth-shell">
      <div className="auth-card">
        <span className="auth-eyebrow">Mersin Üniversitesi</span>
        <h3 className="mb-3">Yeni Şifre Belirle</h3>

        {error && (
          <div className="alert alert-danger" role="alert" aria-live="assertive">
            {error}
          </div>
        )}
        {message && (
          <div className="alert alert-success" role="status" aria-live="polite">
            {message}
          </div>
        )}

        {!token && (
          <div className="alert alert-warning">
            Bağlantıda token bulunamadı. E-postandaki (veya backend logundaki) bağlantıyı
            tekrar kontrol et.
          </div>
        )}

        <form onSubmit={handleSubmit}>
          <div className="mb-3">
            <label htmlFor="reset-new-password" className="form-label">
              Yeni Şifre
            </label>
            <input
              id="reset-new-password"
              className="form-control"
              type="password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              minLength={6}
              required
            />
          </div>
          <div className="mb-3">
            <label htmlFor="reset-confirm-password" className="form-label">
              Yeni Şifre (Tekrar)
            </label>
            <input
              id="reset-confirm-password"
              className="form-control"
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              minLength={6}
              required
            />
          </div>
          <button className="btn btn-primary w-100" type="submit" disabled={submitting}>
            {submitting ? 'Kaydediliyor...' : 'Şifreyi Güncelle'}
          </button>
        </form>

        <p className="mt-3 mb-0">
          <Link to="/login">Girişe dön</Link>
        </p>
      </div>
    </div>
  )
}
