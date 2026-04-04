import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { apiKeyApi, APIKey } from '../api/client'

export function ApiKeyList() {
  const [keys, setKeys] = useState<APIKey[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    apiKeyApi.list()
      .then((res) => setKeys(res.data.keys))
      .finally(() => setLoading(false))
  }, [])

  const handleDelete = async (id: string) => {
    if (!confirm('确定删除此 API Key?')) return
    await apiKeyApi.delete(id)
    setKeys(keys.filter((k) => k.id !== id))
  }

  const handleToggle = async (key: APIKey) => {
    if (key.enabled) {
      await apiKeyApi.disable(key.id)
    } else {
      await apiKeyApi.enable(key.id)
    }
    setKeys(keys.map((k) => (k.id === key.id ? { ...k, enabled: !k.enabled } : k)))
  }

  const maskKey = (key: string) => {
    if (key.length <= 10) return key.slice(0, 3) + '***'
    return key.slice(0, 6) + '***' + key.slice(-4)
  }

  if (loading) return <div style={styles.loading}>加载中...</div>

  return (
    <div>
      <div style={styles.header}>
        <h1>API Key 管理</h1>
        <Link to="/keys/create" style={styles.createBtn}>+ 新建 API Key</Link>
      </div>

      <table style={styles.table}>
        <thead>
          <tr>
            <th style={styles.th}>名称</th>
            <th style={styles.th}>Key</th>
            <th style={styles.th}>可用模型</th>
            <th style={styles.th}>健康状态</th>
            <th style={styles.th}>失败次数</th>
            <th style={styles.th}>创建时间</th>
            <th style={styles.th}>操作</th>
          </tr>
        </thead>
        <tbody>
          {keys.map((key) => (
            <tr key={key.id} style={styles.tr}>
              <td style={styles.td}>{key.name}</td>
              <td style={styles.td}>
                <code style={styles.code}>{maskKey(key.key)}</code>
              </td>
              <td style={styles.td}>
                {key.allowed_models ? (
                  <span style={styles.models}>{key.allowed_models}</span>
                ) : (
                  <span style={styles.noModels}>-</span>
                )}
              </td>
              <td style={styles.td}>
                <span style={{
                  ...styles.status,
                  background: key.healthy ? '#4CAF50' : '#f44336',
                  color: '#fff',
                }}>
                  {key.healthy ? '健康' : '异常'}
                </span>
              </td>
              <td style={styles.td}>{key.fail_count}</td>
              <td style={styles.td}>{new Date(key.created_at).toLocaleDateString()}</td>
              <td style={styles.td}>
                <div style={styles.actions}>
                  <button onClick={() => handleToggle(key)} style={styles.actionBtn}>
                    {key.enabled ? '禁用' : '启用'}
                  </button>
                  <Link to={`/keys/${key.id}`} style={{ ...styles.actionBtn, background: '#FF9800', color: '#fff', textDecoration: 'none' }}>
                    编辑
                  </Link>
                  <button onClick={() => handleDelete(key.id)} style={{ ...styles.actionBtn, background: '#f44336', color: '#fff' }}>
                    删除
                  </button>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {keys.length === 0 && (
        <p style={styles.empty}>暂无 API Key，点击"新建 API Key"创建</p>
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
  code: {
    background: '#f5f5f5',
    padding: '0.25rem 0.5rem',
    borderRadius: '4px',
    fontSize: '0.85rem',
    border: '1px solid #ddd',
  },
  status: {
    padding: '0.25rem 0.5rem',
    borderRadius: '4px',
    fontSize: '0.75rem',
  },
  models: {
    fontSize: '0.75rem',
    color: '#1976D2',
  },
  noModels: {
    color: '#999',
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