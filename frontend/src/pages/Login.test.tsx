import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import { AuthProvider } from '../context/AuthContext'
import Login from './Login'

function renderLogin() {
  render(
    <BrowserRouter>
      <AuthProvider>
        <Login />
      </AuthProvider>
    </BrowserRouter>,
  )
}

describe('Login sayfası', () => {
  it('e-posta ve şifre alanlarını gösterir', () => {
    renderLogin()
    expect(screen.getByLabelText('E-posta')).toBeInTheDocument()
    expect(screen.getByLabelText('Şifre')).toBeInTheDocument()
  })

  it('kayıt sayfasına link içerir', () => {
    renderLogin()
    expect(screen.getByText('Kayıt ol')).toBeInTheDocument()
  })
})
