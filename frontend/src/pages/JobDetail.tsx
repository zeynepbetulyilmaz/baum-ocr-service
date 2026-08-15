import { useEffect, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import client from '../api/client'

interface DocDetail {
  id: string
  original_filename: string
  status: string
  page_count: number
  error_message?: string
  text: string
  has_pdf: boolean
}

export default function JobDetail() {
  const { id } = useParams()
  const [doc, setDoc] = useState<DocDetail | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    let interval: ReturnType<typeof setInterval> | undefined

    async function fetchDoc() {
      try {
        const res = await client.get(`/documents/${id}`)
        setDoc(res.data)
        if (res.data.status === 'done' || res.data.status === 'failed') {
          if (interval) clearInterval(interval)
        }
      } catch (err: any) {
        setError(err.response?.data?.error || 'Belge alınamadı')
        if (interval) clearInterval(interval)
      }
    }

    fetchDoc()
    interval = setInterval(fetchDoc, 3000)
    return () => {
      if (interval) clearInterval(interval)
    }
  }, [id])

  async function download(kind: 'text' | 'pdf') {
    const res = await client.get(`/documents/${id}/${kind}`, { responseType: 'blob' })
    const url = window.URL.createObjectURL(new Blob([res.data]))
    const a = document.createElement('a')
    a.href = url
    a.download = kind === 'text' ? 'sonuc.txt' : 'sonuc.pdf'
    a.click()
    window.URL.revokeObjectURL(url)
  }

  if (error) return <div className="alert alert-danger">{error}</div>
  if (!doc) return <p>Yükleniyor...</p>

  return (
    <div>
      <Link to="/" className="btn btn-sm btn-outline-secondary mb-3">
        ← Geri
      </Link>
      <h4>{doc.original_filename}</h4>
      <p>
        Durum: <strong>{doc.status}</strong> — Sayfa: {doc.page_count || '-'}
      </p>
      {doc.error_message && <div className="alert alert-danger">{doc.error_message}</div>}

      {doc.status === 'done' && (
        <div className="d-flex gap-2 mb-3">
          <button className="btn btn-primary" onClick={() => download('text')} type="button">
            .txt indir
          </button>
          {doc.has_pdf && (
            <button className="btn btn-outline-primary" onClick={() => download('pdf')} type="button">
              Aranabilir .pdf indir
            </button>
          )}
        </div>
      )}

      {doc.text && (
        <pre className="border rounded p-3 bg-white" style={{ whiteSpace: 'pre-wrap' }}>
          {doc.text}
        </pre>
      )}
    </div>
  )
}
