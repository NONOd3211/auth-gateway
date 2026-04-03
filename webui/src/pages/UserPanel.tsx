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
  hourly_used: number
  weekly_limit: boolean
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
        <h1 style={styles.title}>🔑 Token 查询</h1>
        <p style={styles.subtitle}>输入你的 Token 查看额度</p>

        <form onSubmit={handleLookup} style={styles.form}>
          <input
            type="text"
            value={tokenInput}
            onChange={(e) => setTokenInput(e.target.value)}
            placeholder="sk-xxx..."
            style={styles.input}
          />
          <button type="submit" style={styles.button} disabled={loading}>
            {loading ? '查询中...' : '查询'}
          </button>
        </form>

        {error && <p style={styles.error}>{error}</p>}

        {tokenInfo && (
          <div style={styles.tokenInfo}>
            <h3>Token 信息</h3>
            <div style={styles.infoRow}>
              <span>名称:</span>
              <span>{tokenInfo.name || '未命名'}</span>
            </div>
            <div style={styles.infoRow}>
              <span>总请求:</span>
              <span>{tokenInfo.total_requests}</span>
            </div>
            <div style={styles.infoRow}>
              <span>成功请求:</span>
              <span style={{ color: '#4CAF50' }}>{tokenInfo.success_count}</span>
            </div>
            <div style={styles.infoRow}>
              <span>状态:</span>
              <span style={{ color: tokenInfo.enabled ? '#4CAF50' : '#f44336' }}>
                {tokenInfo.enabled ? '✅ 启用' : '❌ 禁用'}
              </span>
            </div>
            {tokenInfo.hourly_limit && (
              <div style={styles.infoRow}>
                <span>小时限额:</span>
                <span>{tokenInfo.hourly_used} / 5小时</span>
              </div>
            )}
            {tokenInfo.weekly_limit && (
              <div style={styles.infoRow}>
                <span>周限额:</span>
                <span>{tokenInfo.weekly_used} / 周</span>
              </div>
            )}
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
    background: '#333',
    padding: '2rem',
    borderRadius: '8px',
    width: '100%',
    maxWidth: '500px',
  },
  title: {
    textAlign: 'center',
    marginBottom: '0.5rem',
  },
  subtitle: {
    textAlign: 'center',
    color: '#888',
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
    border: '1px solid #555',
    background: '#222',
    color: '#fff',
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
    background: '#222',
    borderRadius: '4px',
  },
  infoRow: {
    display: 'flex',
    justifyContent: 'space-between',
    padding: '0.5rem 0',
    borderBottom: '1px solid #444',
  },
}