import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { tokenApi, Token } from '../api/client'

export function TokenList() {
  const [tokens, setTokens] = useState<Token[]>([])
  const [loading, setLoading] = useState(true)
  const navigate = useNavigate()

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

  const handleCopy = (token: string) => {
    navigator.clipboard.writeText(token)
    alert('已复制到剪贴板')
  }

  if (loading) return <div style={styles.loading}>加载中...</div>

  return (
    <div>
      <div style={styles.header}>
        <h1>Token 管理</h1>
        <Link to="/tokens/create" style={styles.createBtn}>+ 新建 Token</Link>
      </div>

      <table style={styles.table}>
        <thead>
          <tr>
            <th style={styles.th}>名称</th>
            <th style={styles.th}>Token</th>
            <th style={styles.th}>创建时间</th>
            <th style={styles.th}>总限制量</th>
            <th style={styles.th}>5小时限制</th>
            <th style={styles.th}>周限制</th>
            <th style={styles.th}>状态</th>
            <th style={styles.th}>操作</th>
          </tr>
        </thead>
        <tbody>
          {tokens.map((token) => (
            <tr key={token.id} style={styles.tr}>
              <td style={styles.td}>{token.name}</td>
              <td style={styles.td}>
                <div style={styles.tokenCell}>
                  <code style={styles.code}>{token.token}</code>
                  <button onClick={() => handleCopy(token.token)} style={styles.copyBtn}>复制</button>
                </div>
              </td>
              <td style={styles.td}>{new Date(token.created_at).toLocaleDateString()}</td>
              <td style={styles.td}>{token.max_requests > 0 ? `${token.max_requests}次` : '-'}</td>
              <td style={styles.td}>{token.hourly_limit ? `${token.hourly_requests}次/5小时` : '-'}</td>
              <td style={styles.td}>{token.weekly_limit ? `${token.weekly_requests}次/周` : '-'}</td>
              <td style={styles.td}>
                <span style={{
                  ...styles.status,
                  background: token.enabled ? '#4CAF50' : '#999',
                  color: '#fff',
                }}>
                  {token.enabled ? '启用' : '禁用'}
                </span>
              </td>
              <td style={styles.td}>
                <div style={styles.actions}>
                  <button onClick={() => navigate(`/tokens/${token.id}`)} style={{ ...styles.actionBtn, background: '#FF9800', color: '#fff' }}>
                    编辑
                  </button>
                  <button onClick={() => handleToggle(token)} style={styles.actionBtn}>
                    {token.enabled ? '禁用' : '启用'}
                  </button>
                  <button onClick={() => handleDelete(token.id)} style={{ ...styles.actionBtn, background: '#f44336', color: '#fff' }}>
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
  loading: {
    textAlign: 'center',
    padding: '2rem',
    color: '#666',
  },
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
    display: 'inline-block',
  },
  table: {
    width: '100%',
    borderCollapse: 'collapse',
    background: '#fff',
    border: '1px solid #ddd',
    borderRadius: '8px',
    overflow: 'hidden',
  },
  th: {
    background: '#f5f5f5',
    padding: '12px',
    textAlign: 'left',
    fontWeight: 600,
    borderBottom: '2px solid #ddd',
    color: '#333',
  },
  td: {
    padding: '12px',
    borderBottom: '1px solid #eee',
    color: '#333',
  },
  tr: {
    borderBottom: '1px solid #eee',
  },
  tokenCell: {
    display: 'flex',
    alignItems: 'center',
    gap: '0.5rem',
  },
  code: {
    background: '#f5f5f5',
    padding: '0.25rem 0.5rem',
    borderRadius: '4px',
    fontSize: '0.75rem',
    border: '1px solid #ddd',
  },
  copyBtn: {
    background: '#2196F3',
    color: '#fff',
    border: 'none',
    padding: '0.25rem 0.5rem',
    borderRadius: '4px',
    cursor: 'pointer',
    fontSize: '0.75rem',
  },
  status: {
    padding: '0.25rem 0.5rem',
    borderRadius: '4px',
    fontSize: '0.75rem',
  },
  actions: {
    display: 'flex',
    gap: '0.5rem',
    flexWrap: 'wrap',
  },
  actionBtn: {
    background: '#e0e0e0',
    color: '#333',
    border: 'none',
    padding: '0.25rem 0.5rem',
    borderRadius: '4px',
    cursor: 'pointer',
    fontSize: '0.75rem',
  },
  empty: {
    textAlign: 'center',
    padding: '2rem',
    color: '#666',
  },
}