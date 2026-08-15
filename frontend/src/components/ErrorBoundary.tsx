import { Component, ErrorInfo, ReactNode } from 'react'

interface Props {
  children: ReactNode
}

interface State {
  hasError: boolean
}

// Herhangi bir alt component render sırasında beklenmedik bir hata
// fırlatırsa (ör. null bir alanı okumaya çalışmak), React'in varsayılan
// davranışı tüm uygulamayı unmount edip bembeyaz bir ekran bırakmaktır.
// Bu component'i App'in etrafına sarmak, kullanıcıya en azından "bir şeyler
// ters gitti, sayfayı yenile" gibi anlaşılır bir mesaj göstermemizi sağlar.
export default class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false }

  static getDerivedStateFromError(): State {
    return { hasError: true }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    // eslint-disable-next-line no-console
    console.error('Beklenmeyen bir hata yakalandı:', error, info)
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="container py-5 text-center">
          <h3>Bir şeyler ters gitti</h3>
          <p className="text-muted">
            Beklenmeyen bir hata oluştu. Sayfayı yenilemeyi deneyebilirsin.
          </p>
          <button className="btn btn-primary" onClick={() => window.location.reload()}>
            Sayfayı Yenile
          </button>
        </div>
      )
    }
    return this.props.children
  }
}
