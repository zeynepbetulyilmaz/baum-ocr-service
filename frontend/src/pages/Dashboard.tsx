import { FormEvent, useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import client from '../api/client'

interface DocSummary {
  id: string
  original_filename: string
  status: string
  page_count: number
  error_message?: string
  created_at: string
}

const statusLabel: Record<string, string> = {
  queued: 'Sırada',
  processing: 'İşleniyor',
  done: 'Tamamlandı',
  failed: 'Başarısız',
}

const statusClass: Record<string, string> = {
  queued: 'bg-secondary',
  processing: 'bg-warning text-dark',
  done: 'bg-success',
  failed: 'bg-danger',
}

const PAGE_SIZE = 10
const ACTIVE_STATUSES = new Set(['queued', 'processing'])

export default function Dashboard() {
  const [docs, setDocs] = useState<DocSummary[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [uploading, setUploading] = useState(false)
  const [error, setError] = useState('')
  const fileRef = useRef<HTMLInputElement>(null)
  const langRef = useRef<HTMLSelectElement>(null)
  // Interval callback'i her zaman en güncel belge listesini görebilsin diye
  // (stale closure sorunu yaşamadan) bir ref'te de tutuyoruz.
  const docsRef = useRef<DocSummary[]>([])

  async function fetchDocs(targetPage = page) {
    const res = await client.get('/documents', {
      params: { page: targetPage, page_size: PAGE_SIZE },
    })
    const items: DocSummary[] = res.data.items || []
    setDocs(items)
    docsRef.current = items
    setTotal(res.data.total || 0)
    setPage(res.data.page || targetPage)
  }

    useEffect(() => {
        fetchDocs(1).catch(() => {
            // İlk yükleme başarısız olursa (ör. sayfa açılışında token zaten
            // geçersizse) sessizce yut — axios interceptor yönlendirmeyi
            // zaten yapacak.
        })

        const interval = setInterval(() => {
            const hasActive = docsRef.current.some((d) => ACTIVE_STATUSES.has(d.status))
            if (hasActive) {
                // fetchDocs başarısız olursa (ör. token geçersiz kılınmış, 401)
                // polling'i sonsuza kadar her 4 saniyede bir tekrar denemek yerine
                // tamamen durduruyoruz.
                fetchDocs().catch(() => clearInterval(interval))
            }
        }, 4000)

        return () => clearInterval(interval)
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])
  async function handleUpload(e: FormEvent) {
    e.preventDefault()
    setError('')
    const file = fileRef.current?.files?.[0]
    if (!file) return

    const formData = new FormData()
    formData.append('file', file)
    formData.append('lang', langRef.current?.value || 'tur+eng')
    setUploading(true)
    try {
      await client.post('/documents', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      })
      if (fileRef.current) fileRef.current.value = ''
      await fetchDocs(1)
    } catch (err: any) {
      setError(err.response?.data?.error || 'Yükleme başarısız')
    } finally {
      setUploading(false)
    }
  }

  async function handleDelete(id: string) {
    if (!window.confirm('Bu belgeyi silmek istediğine emin misin?')) return
    try {
      await client.delete(`/documents/${id}`)
      await fetchDocs(page)
    } catch (err: any) {
      setError(err.response?.data?.error || 'Silme başarısız')
    }
  }

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  return (
    <div>
      <h3 className="mb-3">Belge Yükle (PDF / Görüntü)</h3>
      {error && <div className="alert alert-danger">{error}</div>}
      <form className="d-flex gap-2 mb-4" onSubmit={handleUpload}>
        <input
          ref={fileRef}
          type="file"
          className="form-control"
          accept=".pdf,.png,.jpg,.jpeg,.tif,.tiff"
          required
        />
        <select
          ref={langRef}
          className="form-select"
          style={{ maxWidth: 220 }}
          defaultValue="tur+eng"
          aria-label="OCR dili"
        >
          <option value="tur+eng">Türkçe + İngilizce</option>
          <option value="tur">Türkçe</option>
          <option value="eng">İngilizce</option>
        </select>
        <button className="btn btn-primary text-nowrap" type="submit" disabled={uploading}>
          {uploading ? 'Yükleniyor...' : 'Yükle'}
        </button>
      </form>

      <h4 className="mb-3">Belgelerim</h4>
      <div className="table-responsive">
        <table className="table table-hover align-middle">
          <thead>
            <tr>
              <th>Dosya</th>
              <th>Durum</th>
              <th>Sayfa</th>
              <th>Tarih</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {docs.map((d) => (
              <tr key={d.id}>
                <td style={{ wordBreak: 'break-word', maxWidth: 260 }}>{d.original_filename}</td>
                <td>
                  <span className={`badge ${statusClass[d.status] || 'bg-secondary'}`}>
                    {statusLabel[d.status] || d.status}
                  </span>
                </td>
                <td>{d.page_count || '-'}</td>
                <td style={{ whiteSpace: 'nowrap' }}>{new Date(d.created_at).toLocaleString('tr-TR')}</td>
                <td className="d-flex gap-2" style={{ whiteSpace: 'nowrap' }}>
                  <Link className="btn btn-sm btn-outline-primary" to={`/documents/${d.id}`}>
                    Detay
                  </Link>
                  <button
                    className="btn btn-sm btn-outline-danger"
                    onClick={() => handleDelete(d.id)}
                    type="button"
                  >
                    Sil
                  </button>
                </td>
              </tr>
            ))}
            {docs.length === 0 && (
              <tr>
                <td colSpan={5} className="text-center text-muted">
                  Henüz belge yok
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {total > 0 && (
        <div className="d-flex justify-content-between align-items-center">
          <span className="text-muted small">
            Toplam {total} belge — sayfa {page}/{totalPages}
          </span>
          <div className="d-flex gap-2">
            <button
              className="btn btn-sm btn-outline-primary"
              disabled={page <= 1}
              onClick={() => fetchDocs(page - 1)}
              type="button"
            >
              Önceki
            </button>
            <button
              className="btn btn-sm btn-outline-primary"
              disabled={page >= totalPages}
              onClick={() => fetchDocs(page + 1)}
              type="button"
            >
              Sonraki
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
