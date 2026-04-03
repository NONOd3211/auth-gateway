import axios from 'axios'

const api = axios.create({
  baseURL: '/api/admin',
})

api.interceptors.request.use((config) => {
  const password = localStorage.getItem('admin_password')
  if (password) {
    config.headers.Authorization = `Bearer ${password}`
  }
  return config
})

export interface Token {
  id: string
  token: string
  name: string
  created_at: string
  expires_at: string | null
  max_requests: number
  used_requests: number
  enabled: boolean
  user_id: string
  description: string
  hourly_limit: boolean
  hourly_requests: number
  hourly_used: number
  weekly_limit: boolean
  weekly_requests: number
  weekly_used: number
}

export interface UsageStats {
  total_requests: number
  success_count: number
  failure_count: number
  total_tokens: number
  input_tokens: number
  output_tokens: number
}

export const tokenApi = {
  list: () => api.get<{ tokens: Token[] }>('/tokens'),
  get: (id: string) => api.get<{ token: Token; usage_count: number }>(`/tokens/${id}`),
  create: (data: Partial<Token>) => api.post('/tokens', data),
  update: (id: string, data: Partial<Token>) => api.put(`/tokens/${id}`, data),
  delete: (id: string) => api.delete(`/tokens/${id}`),
  resetUsage: (id: string) => api.post(`/tokens/${id}/reset`),
}

export const usageApi = {
  stats: (params?: { token_id?: string; start_date?: string; end_date?: string }) =>
    api.get<UsageStats>('/usage', { params }),
  daily: (params?: { token_id?: string }) =>
    api.get<{ daily: Array<{ date: string; requests: number; total_tokens: number }> }>('/usage/daily', { params }),
  byToken: (id: string) => api.get<{ records: any[] }>(`/usage/token/${id}`),
}

export default api
