import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { tokenApi, Token } from '../api/client'

export function Dashboard() {
  const [tokens, setTokens] = useState<Token[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    tokenApi.list()
      .then((res) => setTokens(res.data.tokens || []))
      .finally(() => setLoading(false))
  }, [])

  if (loading) return <div style={styles.loading}>加载中...</div>

  return (
    <div>
      <div style={styles.header}>
        <h1 style={styles.title}>仪表盘</h1>
        <Link to="/tokens/create" style={styles.createBtn}>+ 新建 Token</Link>
      </div>

      <h2 style={styles.subtitle}>Token 列表</h2>
      {tokens.length === 0 ? (
        <div style={styles.empty}>
          <p>暂无 Token</p>
          <Link to="/tokens/create" style={styles.createLink}>创建一个</Link>
        </div>
      ) : (
        <table style={styles.table}>
          <thead>
            <tr>
              <th style={styles.th}>名称</th>
              <th style={styles.th}>Token</th>
              <th style={styles.th}>总使用量</th>
              <th style={styles.th}>5小时使用量</th>
              <th style={styles.th}>周使用量</th>
              <th style={styles.th}>状态</th>
            </tr>
          </thead>
          <tbody>
            {tokens.map((token) => (
              <tr key={token.id} style={styles.tr}>
                <td style={styles.td}>{token.name}</td>
                <td style={styles.td}>
                  <div style={styles.tokenCell}>
                    <code style={styles.code}>{token.token}</code>
                  </div>
                </td>
                <td style={styles.td}>
                  {token.used_requests} / {token.max_requests || '∞'}
                </td>
                <td style={styles.td}>
                  {token.hourly_limit ? `${token.hourly_used} / ${token.hourly_requests}` : '-'}
                </td>
                <td style={styles.td}>
                  {token.weekly_limit ? `${token.weekly_used} / ${token.weekly_requests}` : '-'}
                </td>
                <td style={styles.td}>
                  <span style={{
                    ...styles.status,
                    background: token.enabled ? '#4CAF50' : '#999',
                    color: '#fff',
                  }}>
                    {token.enabled ? '启用' : '禁用'}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
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
  title: {
    margin: 0,
    color: '#333',
  },
  subtitle: {
    marginBottom: '1rem',
    color: '#333',
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
    fontSize: '0.75rem',
    background: '#f5f5f5',
    padding: '0.25rem 0.5rem',
    borderRadius: '4px',
    border: '1px solid #ddd',
  },
  status: {
    padding: '0.25rem 0.5rem',
    borderRadius: '4px',
    fontSize: '0.75rem',
  },
  empty: {
    background: '#fff',
    padding: '2rem',
    borderRadius: '8px',
    textAlign: 'center',
    color: '#666',
    border: '1px solid #ddd',
  },
  createLink: {
    color: '#4CAF50',
    textDecoration: 'none',
  },
}