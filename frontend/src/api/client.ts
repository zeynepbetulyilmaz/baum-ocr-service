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

let redirectingToLogin = false

client.interceptors.response.use(
    (response) => response,
    (error) => {
        if (error.response?.status === 401 && !redirectingToLogin && window.location.pathname !== '/login') {
            redirectingToLogin = true
            localStorage.removeItem('token')
            localStorage.removeItem('user')
            window.location.replace('/login')
        }
        return Promise.reject(error)
    },
)

export default client