import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { tokenApi, Token } from '../api/client'

export function TokenList() {
  const [tokens, setTokens] = useState<Token[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    tokenApi.list()
      .then((res) => setTokens(res.data.tokens))
      .finally(() => setLoading(false))
  }, [])

  const handleDelete = async (id: string) => {
    if (!confirm('确定删除此 Token?')) return
    await tokenApi.delete(id)
    setTokens(tokens.filter((t) => t.id !== id))
  }

  const handleToggle = async (token: Token) => {
    await tokenApi.update(token.id, { enabled: !token.enabled })
    setTokens(tokens.map((t) => (t.id === token.id ? { ...t, enabled: !t.enabled } : t)))
  }

  const handleReset = async (id: string) => {
    if (!confirm('确定重置使用次数?')) return
    await tokenApi.resetUsage(id)
    setTokens(tokens.map((t) => (t.id === id ? { ...t, used_requests: 0 } : t)))
  }

  if (loading) return <div>加载中...</div>

  return (
    <div>
      <div style={styles.header}>
        <h1>📝 Token 管理</h1>
        <Link to="/tokens/create" style={styles.createBtn}>+ 新建 Token</Link>
      </div>

      <table style={styles.table}>
        <thead>
          <tr>
            <th>名称</th>
            <th>Token</th>
            <th>创建时间</th>
            <th>过期时间</th>
            <th>使用量</th>
            <th>状态</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          {tokens.map((token) => (
            <tr key={token.id}>
              <td>{token.name}</td>
              <td><code style={styles.code}>{token.token.slice(0, 12)}...</code></td>
              <td>{new Date(token.created_at).toLocaleDateString()}</td>
              <td>{token.expires_at ? new Date(token.expires_at).toLocaleDateString() : '-'}</td>
              <td>
                <span style={styles.usage}>
                  {token.used_requests}
                  {token.max_requests > 0 && ` / ${token.max_requests}`}
                </span>
              </td>
              <td>
                <span style={{
                  ...styles.status,
                  background: token.enabled ? '#4CAF50' : '#666',
                }}>
                  {token.enabled ? '启用' : '禁用'}
                </span>
              </td>
              <td>
                <div style={styles.actions}>
                  <button onClick={() => handleToggle(token)} style={styles.actionBtn}>
                    {token.enabled ? '禁用' : '启用'}
                  </button>
                  <button onClick={() => handleReset(token.id)} style={styles.actionBtn}>
                    重置
                  </button>
                  <button onClick={() => handleDelete(token.id)} style={{ ...styles.actionBtn, background: '#f44336' }}>
                    删除
                  </button>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {tokens.length === 0 && (
        <p style={styles.empty}>暂无 Token，点击"新建 Token"创建</p>
      )}
    </div>
  )
}

const styles: Record<string, React.CSSProperties> = {
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '1.5rem',
  },
  createBtn: {
    background: '#4CAF50',
    color: '#fff',
    padding: '0.5rem 1rem',
    borderRadius: '4px',
    textDecoration: 'none',
  },
  table: {
    width: '100%',
    borderCollapse: 'collapse',
    background: '#333',
    borderRadius: '8px',
    overflow: 'hidden',
  },
  code: {
    background: '#222',
    padding: '0.25rem 0.5rem',
    borderRadius: '4px',
    fontSize: '0.875rem',
  },
  usage: {
    fontSize: '0.875rem',
  },
  status: {
    color: '#fff',
    padding: '0.25rem 0.5rem',
    borderRadius: '4px',
    fontSize: '0.75rem',
  },
  actions: {
    display: 'flex',
    gap: '0.5rem',
  },
  actionBtn: {
    background: '#555',
    color: '#fff',
    border: 'none',
    padding: '0.25rem 0.5rem',
    borderRadius: '4px',
    cursor: 'pointer',
    fontSize: '0.75rem',
  },
  empty: {
    textAlign: 'center',
    padding: '2rem',
    color: '#888',
  },
}
