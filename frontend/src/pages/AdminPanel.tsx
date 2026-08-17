import { useEffect, useState } from 'react'
import client from '../api/client'
import { useAuth } from '../context/AuthContext'

interface AdminUser {
    id: string
    username: string
    email: string
    role: string
    document_count: number
    created_at: string
}

interface AdminDocument {
    id: string
    original_filename: string
    status: string
    page_count: number
    created_at: string
    owner_username: string
    owner_email: string
}

interface AuditLog {
    id: string
    actor_username: string
    action: string
    target_type: string
    target_id: string
    details: string
    created_at: string
}

const ACTION_LABELS: Record<string, string> = {
    delete_user: 'Kullanıcı sildi',
}

export default function AdminPanel() {
    const { user } = useAuth()
    const [users, setUsers] = useState<AdminUser[]>([])
    const [docs, setDocs] = useState<AdminDocument[]>([])
    const [auditLogs, setAuditLogs] = useState<AuditLog[]>([])
    const [error, setError] = useState('')
    const [tab, setTab] = useState<'users' | 'documents' | 'audit'>('users')

    async function load() {
        try {
            const [usersRes, docsRes, auditRes] = await Promise.all([
                client.get('/admin/users'),
                client.get('/admin/documents'),
                client.get('/admin/audit-logs'),
            ])
            setUsers(usersRes.data.items || [])
            setDocs(docsRes.data.items || [])
            setAuditLogs(auditRes.data.items || [])
        } catch (err: any) {
            setError(err.response?.data?.error || 'Veriler alınamadı')
        }
    }

    useEffect(() => {
        load()
    }, [])

    async function handleDeleteUser(id: string, username: string) {
        if (!window.confirm(`"${username}" kullanıcısını ve tüm belgelerini silmek istediğine emin misin?`)) return
        try {
            await client.delete(`/admin/users/${id}`)
            await load()
        } catch (err: any) {
            setError(err.response?.data?.error || 'Silme başarısız')
        }
    }

    if (user?.role !== 'admin') {
        return <div className="alert alert-warning">Bu sayfayı görüntülemek için yönetici yetkisi gerekiyor.</div>
    }

    return (
        <div>
            <h3 className="mb-3">Yönetici Paneli</h3>
            {error && <div className="alert alert-danger">{error}</div>}

            <ul className="nav nav-tabs mb-3">
                <li className="nav-item">
                    <button
                        className={`nav-link ${tab === 'users' ? 'active' : ''}`}
                        onClick={() => setTab('users')}
                        type="button"
                    >
                        Kullanıcılar ({users.length})
                    </button>
                </li>
                <li className="nav-item">
                    <button
                        className={`nav-link ${tab === 'documents' ? 'active' : ''}`}
                        onClick={() => setTab('documents')}
                        type="button"
                    >
                        Tüm Belgeler ({docs.length})
                    </button>
                </li>
                <li className="nav-item">
                    <button
                        className={`nav-link ${tab === 'audit' ? 'active' : ''}`}
                        onClick={() => setTab('audit')}
                        type="button"
                    >
                        İşlem Geçmişi ({auditLogs.length})
                    </button>
                </li>
            </ul>

            {tab === 'users' && (
                <div className="table-responsive">
                    <table className="table table-hover align-middle">
                        <thead>
                            <tr>
                                <th>Kullanıcı Adı</th>
                                <th>E-posta</th>
                                <th>Rol</th>
                                <th>Belge Sayısı</th>
                                <th>Kayıt Tarihi</th>
                                <th></th>
                            </tr>
                        </thead>
                        <tbody>
                            {users.map((u) => (
                                <tr key={u.id}>
                                    <td>{u.username}</td>
                                    <td>{u.email}</td>
                                    <td>
                                        <span className={`badge ${u.role === 'admin' ? 'bg-primary' : 'bg-secondary'}`}>
                                            {u.role}
                                        </span>
                                    </td>
                                    <td>{u.document_count}</td>
                                    <td style={{ whiteSpace: 'nowrap' }}>{new Date(u.created_at).toLocaleString('tr-TR')}</td>
                                    <td>
                                        {u.id !== user.id && (
                                            <button
                                                className="btn btn-sm btn-outline-danger"
                                                onClick={() => handleDeleteUser(u.id, u.username)}
                                                type="button"
                                            >
                                                Sil
                                            </button>
                                        )}
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            )}

            {tab === 'documents' && (
                <div className="table-responsive">
                    <table className="table table-hover align-middle">
                        <thead>
                            <tr>
                                <th>Dosya</th>
                                <th>Sahibi</th>
                                <th>Durum</th>
                                <th>Sayfa</th>
                                <th>Tarih</th>
                            </tr>
                        </thead>
                        <tbody>
                            {docs.map((d) => (
                                <tr key={d.id}>
                                    <td style={{ wordBreak: 'break-word', maxWidth: 240 }}>{d.original_filename}</td>
                                    <td>
                                        {d.owner_username} <span className="text-muted small">({d.owner_email})</span>
                                    </td>
                                    <td>{d.status}</td>
                                    <td>{d.page_count || '-'}</td>
                                    <td style={{ whiteSpace: 'nowrap' }}>{new Date(d.created_at).toLocaleString('tr-TR')}</td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            )}

            {tab === 'audit' && (
                <div className="table-responsive">
                    {auditLogs.length === 0 ? (
                        <p className="text-muted">Henüz kaydedilmiş bir yönetici işlemi yok.</p>
                    ) : (
                        <table className="table table-hover align-middle">
                            <thead>
                                <tr>
                                    <th>Tarih</th>
                                    <th>Yapan</th>
                                    <th>İşlem</th>
                                    <th>Hedef</th>
                                    <th>Detay</th>
                                </tr>
                            </thead>
                            <tbody>
                                {auditLogs.map((log) => (
                                    <tr key={log.id}>
                                        <td style={{ whiteSpace: 'nowrap' }}>
                                            {new Date(log.created_at).toLocaleString('tr-TR')}
                                        </td>
                                        <td>{log.actor_username}</td>
                                        <td>
                                            <span className="badge bg-warning">
                                                {ACTION_LABELS[log.action] || log.action}
                                            </span>
                                        </td>
                                        <td className="text-muted small">
                                            {log.target_type}: {log.target_id}
                                        </td>
                                        <td>{log.details}</td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    )}
                </div>
            )}
        </div>
    )
}