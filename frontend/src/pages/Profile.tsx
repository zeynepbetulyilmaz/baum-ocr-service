import { FormEvent, useState } from 'react'
import client from '../api/client'
import { useAuth } from '../context/AuthContext'
import { Link } from 'react-router-dom';

export default function Profile() {
    const { user, updateUser, logout, deleteMyAccount } = useAuth()

    const [deleteConfirmText, setDeleteConfirmText] = useState('')
    const [deleteError, setDeleteError] = useState('')
    const [deleting, setDeleting] = useState(false)

    async function handleDeleteAccount() {
        if (deleteConfirmText !== 'HESABIMI SİL') return
        setDeleteError('')
        setDeleting(true)
        try {
            await deleteMyAccount()
        } catch (err: any) {
            setDeleteError(err.response?.data?.error || 'Hesap silinemedi')
            setDeleting(false)
        }
    }

    const [username, setUsername] = useState(user?.username || '')
    const [email, setEmail] = useState(user?.email || '')
    const [profileError, setProfileError] = useState('')
    const [profileSuccess, setProfileSuccess] = useState('')
    const [savingProfile, setSavingProfile] = useState(false)

    const [currentPassword, setCurrentPassword] = useState('')
    const [newPassword, setNewPassword] = useState('')
    const [passwordError, setPasswordError] = useState('')
    const [passwordSuccess, setPasswordSuccess] = useState('')
    const [savingPassword, setSavingPassword] = useState(false)

    async function handleProfileSubmit(e: FormEvent) {
        e.preventDefault()
        setProfileError('')
        setProfileSuccess('')
        setSavingProfile(true)
        try {
            const res = await client.patch('/me', { username, email })
            updateUser(res.data.user)
            setProfileSuccess('Profil güncellendi.')
        } catch (err: any) {
            setProfileError(err.response?.data?.error || 'Güncelleme başarısız')
        } finally {
            setSavingProfile(false)
        }
    }

    async function handlePasswordSubmit(e: FormEvent) {
        e.preventDefault()
        setPasswordError('')
        setPasswordSuccess('')
        setSavingPassword(true)
        try {
            await client.post('/me/password', {
                current_password: currentPassword,
                new_password: newPassword,
            })
            setPasswordSuccess('Şifre güncellendi, tekrar giriş yapman gerekiyor...')
            setTimeout(() => logout(), 2000)
        } catch (err: any) {
            setPasswordError(err.response?.data?.error || 'Şifre güncellenemedi')
        } finally {
            setSavingPassword(false)
        }
    }

    return (
        <div className="container py-4" style={{ maxWidth: 480 }}>
            <Link to="/" className="btn btn-outline-secondary btn-sm mb-3">
                ← Panele Dön
            </Link>

            <h3 className="mb-4">Profilim</h3>

            <div className="card mb-4">
                <div className="card-body">
                    <h5 className="card-title">Hesap Bilgileri</h5>

                    {profileError && (
                        <div className="alert alert-danger" role="alert" aria-live="assertive">{profileError}</div>
                    )}
                    {profileSuccess && (
                        <div className="alert alert-success" role="status" aria-live="polite">{profileSuccess}</div>
                    )}

                    <form onSubmit={handleProfileSubmit}>
                        <div className="mb-3">
                            <label htmlFor="profile-username" className="form-label">
                                Kullanıcı Adı
                            </label>
                            <input
                                id="profile-username"
                                className="form-control"
                                value={username}
                                onChange={(e) => setUsername(e.target.value)}
                            />
                        </div>

                        <div className="mb-3">
                            <label htmlFor="profile-email" className="form-label">
                                E-posta
                            </label>
                            <input
                                id="profile-email"
                                type="email"
                                className="form-control"
                                value={email}
                                onChange={(e) => setEmail(e.target.value)}
                            />
                        </div>

                        <button
                            type="submit"
                            className="btn btn-primary"
                            disabled={savingProfile}
                        >
                            {savingProfile ? 'Kaydediliyor...' : 'Kaydet'}
                        </button>
                    </form>
                </div>
            </div>

            <div className="card">
                <div className="card-body">
                    <h5 className="card-title">Şifre Değiştir</h5>

                    {passwordError && (
                        <div className="alert alert-danger" role="alert" aria-live="assertive">{passwordError}</div>
                    )}
                    {passwordSuccess && (
                        <div className="alert alert-success" role="status" aria-live="polite">{passwordSuccess}</div>
                    )}

                    <form onSubmit={handlePasswordSubmit}>
                        <div className="mb-3">
                            <label htmlFor="current-password" className="form-label">
                                Mevcut Şifre
                            </label>
                            <input
                                id="current-password"
                                type="password"
                                className="form-control"
                                value={currentPassword}
                                onChange={(e) => setCurrentPassword(e.target.value)}
                            />
                        </div>

                        <div className="mb-3">
                            <label htmlFor="new-password" className="form-label">
                                Yeni Şifre
                            </label>
                            <input
                                id="new-password"
                                type="password"
                                className="form-control"
                                value={newPassword}
                                onChange={(e) => setNewPassword(e.target.value)}
                            />
                        </div>

                        <button
                            type="submit"
                            className="btn btn-outline-primary"
                            disabled={savingPassword}
                        >
                            {savingPassword ? 'Değiştiriliyor...' : 'Şifreyi Değiştir'}
                        </button>
                    </form>
                </div>
            </div>

            <div className="card mt-4 border-danger">
                <div className="card-body">
                    <h5 className="card-title text-danger">Tehlikeli Bölge</h5>
                    <p className="text-muted" style={{ fontSize: '0.9rem' }}>
                        Hesabını sildiğinde, kullanıcı bilgilerin ve yüklediğin tüm belgeler
                        (dosyalar dahil) kalıcı olarak silinir. Bu işlem geri alınamaz.
                    </p>

                    {deleteError && (
                        <div className="alert alert-danger" role="alert" aria-live="assertive">{deleteError}</div>
                    )}

                    <div className="mb-3">
                        <label htmlFor="delete-confirm" className="form-label">
                            Onaylamak için <strong>HESABIMI SİL</strong> yazın
                        </label>
                        <input
                            id="delete-confirm"
                            className="form-control"
                            value={deleteConfirmText}
                            onChange={(e) => setDeleteConfirmText(e.target.value)}
                            aria-describedby="delete-confirm-help"
                        />
                        <div id="delete-confirm-help" className="form-text">
                            Büyük harflerle, boşluklarla birlikte tam olarak yazmalısın.
                        </div>
                    </div>

                    <button
                        type="button"
                        className="btn btn-outline-danger"
                        disabled={deleteConfirmText !== 'HESABIMI SİL' || deleting}
                        onClick={handleDeleteAccount}
                    >
                        {deleting ? 'Siliniyor...' : 'Hesabımı ve Verilerimi Kalıcı Olarak Sil'}
                    </button>
                </div>
            </div>
        </div>
    );
}