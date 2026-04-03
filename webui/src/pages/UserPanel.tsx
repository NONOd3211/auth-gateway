import { useState } from 'react'
import axios from 'axios'

interface TokenInfo {
  name: string
  max_requests: number
  used_requests: number
  total_requests: number
  success_count: number
  enabled: boolean
  hourly_limit: boolean
  hourly_requests: number
  hourly_used: number
  weekly_limit: boolean
  weekly_requests: number
  weekly_used: number
  created_at: string
}

export function UserPanel() {
  const [tokenInput, setTokenInput] = useState('')
  const [tokenInfo, setTokenInfo] = useState<TokenInfo | null>(null)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleLookup = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!tokenInput.trim()) return

    setLoading(true)
    setError('')
    setTokenInfo(null)

    try {
      const res = await axios.get('/api/lookup', {
        params: { token: tokenInput.trim() }
      })
      setTokenInfo(res.data)
    } catch (err: any) {
      if (err.response?.status === 404) {
        setError('Token 不存在')
      } else {
        setError('查询失败，请检查 Token 是否正确')
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={styles.container}>
      <div style={styles.card}>
        <h1 style={styles.title}>🔑 Token 登录</h1>
        <p style={styles.subtitle}>输入你的 Token 查看额度信息</p>

        <form onSubmit={handleLookup} style={styles.form}>
          <input
            type="text"
            value={tokenInput}
            onChange={(e) => setTokenInput(e.target.value)}
            placeholder="请输入 sk- 开头的 Token"
            style={styles.input}
          />
          <button type="submit" style={styles.button} disabled={loading}>
            {loading ? '登录中...' : '登录'}
          </button>
        </form>

        {error && <p style={styles.error}>{error}</p>}

        {tokenInfo && (
          <div style={styles.tokenInfo}>
            <h3 style={styles.infoTitle}>Token 信息</h3>
            <div style={styles.infoRow}>
              <span>名称:</span>
              <span>{tokenInfo.name || '未命名'}</span>
            </div>
            <div style={styles.infoRow}>
              <span>状态:</span>
              <span style={{ color: tokenInfo.enabled ? '#4CAF50' : '#f44336' }}>
                {tokenInfo.enabled ? '✅ 启用' : '❌ 禁用'}
              </span>
            </div>
            <div style={styles.infoRow}>
              <span>总使用量:</span>
              <span style={styles.usageValue}>{tokenInfo.used_requests} / {tokenInfo.max_requests || '无限制'}</span>
            </div>
            <div style={styles.infoRow}>
              <span>5小时使用量:</span>
              <span style={styles.usageValue}>{tokenInfo.hourly_used} / {tokenInfo.hourly_requests || '无限制'}</span>
            </div>
            <div style={styles.infoRow}>
              <span>周使用量:</span>
              <span style={styles.usageValue}>{tokenInfo.weekly_used} / {tokenInfo.weekly_requests || '无限制'}</span>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

const styles: Record<string, React.CSSProperties> = {
  container: {
    minHeight: '100vh',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    padding: '1rem',
  },
  card: {
    background: '#fff',
    padding: '2rem',
    borderRadius: '8px',
    width: '100%',
    maxWidth: '500px',
    boxShadow: '0 2px 8px rgba(0,0,0,0.1)',
    border: '1px solid #ddd',
  },
  title: {
    textAlign: 'center',
    marginBottom: '0.5rem',
    color: '#333',
  },
  subtitle: {
    textAlign: 'center',
    color: '#666',
    marginBottom: '1.5rem',
  },
  form: {
    display: 'flex',
    flexDirection: 'column',
    gap: '1rem',
  },
  input: {
    padding: '0.75rem',
    borderRadius: '4px',
    border: '1px solid #ccc',
    background: '#fff',
    color: '#333',
    fontSize: '1rem',
  },
  button: {
    padding: '0.75rem',
    borderRadius: '4px',
    border: 'none',
    background: '#4CAF50',
    color: '#fff',
    fontSize: '1rem',
    cursor: 'pointer',
  },
  error: {
    color: '#f44336',
    textAlign: 'center',
    fontSize: '0.875rem',
    marginTop: '1rem',
  },
  tokenInfo: {
    marginTop: '1.5rem',
    padding: '1rem',
    background: '#f5f5f5',
    borderRadius: '4px',
    border: '1px solid #ddd',
  },
  infoTitle: {
    marginBottom: '0.5rem',
    color: '#333',
  },
  infoRow: {
    display: 'flex',
    justifyContent: 'space-between',
    padding: '0.5rem 0',
    borderBottom: '1px solid #eee',
    color: '#333',
  },
  usageValue: {
    fontWeight: 600,
    color: '#1976D2',
  },
}