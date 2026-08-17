import { FormEvent, useState } from 'react'
import { Link } from 'react-router-dom'
import client from '../api/client'

export default function ForgotPassword() {
  const [email, setEmail] = useState('')
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    setMessage('')
    setSubmitting(true)
    try {
      const res = await client.post('/auth/forgot-password', { email })
      setMessage(res.data.status)
    } catch (err: any) {
      setError(err.response?.data?.error || 'Bir hata oluştu')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="auth-shell">
      <div className="auth-card">
        <span className="auth-eyebrow">Mersin Üniversitesi</span>
        <h3 className="mb-3">Şifremi Unuttum</h3>

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

        <p className="text-muted" style={{ fontSize: '0.85rem' }}>
          Kayıtlı e-posta adresini gir, sıfırlama bağlantısı gönderelim.
        </p>

        <form onSubmit={handleSubmit}>
          <div className="mb-3">
            <label htmlFor="forgot-email" className="form-label">
              E-posta
            </label>
            <input
              id="forgot-email"
              className="form-control"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
            />
          </div>
          <button className="btn btn-primary w-100" type="submit" disabled={submitting}>
            {submitting ? 'Gönderiliyor...' : 'Sıfırlama Bağlantısı Gönder'}
          </button>
        </form>

        <p className="mt-3 mb-0">
          <Link to="/login">Girişe dön</Link>
        </p>
      </div>
    </div>
  )
}
