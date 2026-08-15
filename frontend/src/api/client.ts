import axios from 'axios'

const client = axios.create({
  baseURL: import.meta.env.VITE_API_URL || '/api',
})

client.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers = config.headers || {}
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Token süresi dolduğunda (72 saat) ya da geçersiz kılındığında (bkz.
// "tüm cihazlardan çıkış yap") backend her istekte sessizce 401 döner.
// Bu interceptor olmadan kullanıcı ekranda hiçbir şey olmadığını görür —
// istekler konsolda sessizce başarısız olur. Burada 401 yakalanınca
// oturumu temizleyip login sayfasına yönlendiriyoruz.
client.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401 && window.location.pathname !== '/login') {
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  },
)

export default client
