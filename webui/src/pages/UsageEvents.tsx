import { useEffect, useState } from 'react'
import { usageApi, UsageRecord, tokenApi, Token } from '../api/client'

export function UsageEvents() {
  const [records, setRecords] = useState<UsageRecord[]>([])
  const [tokens, setTokens] = useState<Token[]>([])
  const [selectedToken, setSelectedToken] = useState<string>('')
  const [loading, setLoading] = useState(true)
  const [total, setTotal] = useState(0)

  useEffect(() => {
    tokenApi.list().then((res) => setTokens(res.data.tokens))
  }, [])

  useEffect(() => {
    fetchEvents()
  }, [selectedToken])

  const fetchEvents = () => {
    setLoading(true)
    const params = selectedToken ? { token_id: selectedToken } : {}
    usageApi.events(params).then((res) => {
      setRecords(res.data.records)
      setTotal(res.data.total)
      setLoading(false)
    })
  }

  const formatTime = (timestamp: string) => {
    return new Date(timestamp).toLocaleString()
  }

  const maskToken = (tokenId: string) => {
    const token = tokens.find((t) => t.id === tokenId)
    if (token) {
      return token.name || token.token.substring(0, 12) + '...'
    }
    return tokenId.substring(0, 12) + '...'
  }

  if (loading) return <div style={styles.loading}>加载中...</div>

  return (
    <div>
      <div style={styles.header}>
        <h1>使用事件</h1>
        <div style={styles.filter}>
          <label style={styles.label}>筛选 Token：</label>
          <select
            value={selectedToken}
            onChange={(e) => setSelectedToken(e.target.value)}
            style={styles.select}
          >
            <option value="">全部</option>
            {tokens.map((token) => (
              <option key={token.id} value={token.id}>
                {token.name || token.token.substring(0, 20)}
              </option>
            ))}
          </select>
        </div>
      </div>

      <div style={styles.summary}>
        共 {total} 条记录
      </div>

      <table style={styles.table}>
        <thead>
          <tr>
            <th style={styles.th}>时间</th>
            <th style={styles.th}>来源 Token</th>
            <th style={styles.th}>模型</th>
            <th style={styles.th}>结果</th>
            <th style={styles.th}>输入</th>
            <th style={styles.th}>输出</th>
            <th style={styles.th}>缓存</th>
            <th style={styles.th}>总计</th>
          </tr>
        </thead>
        <tbody>
          {records.map((record) => (
            <tr key={record.id} style={styles.tr}>
              <td style={styles.td}>{formatTime(record.timestamp)}</td>
              <td style={styles.td}>{maskToken(record.token_id)}</td>
              <td style={styles.td}>{record.model || '-'}</td>
              <td style={styles.td}>
                <span style={{
                  ...styles.badge,
                  background: record.success ? '#4CAF50' : '#f44336',
                  color: '#fff',
                }}>
                  {record.success ? '成功' : '失败'}
                </span>
                {!record.success && record.error_message && (
                  <div style={styles.errorMsg}>{record.error_message}</div>
                )}
              </td>
              <td style={styles.td}>{record.input_tokens > 0 ? record.input_tokens.toLocaleString() : '-'}</td>
              <td style={styles.td}>{record.output_tokens > 0 ? record.output_tokens.toLocaleString() : '-'}</td>
              <td style={styles.td}>{record.cache_tokens > 0 ? record.cache_tokens.toLocaleString() : '-'}</td>
              <td style={styles.td}>{record.total_tokens > 0 ? record.total_tokens.toLocaleString() : '-'}</td>
            </tr>
          ))}
        </tbody>
      </table>

      {records.length === 0 && (
        <p style={styles.empty}>暂无使用记录</p>
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
    marginBottom: '1rem',
  },
  filter: {
    display: 'flex',
    alignItems: 'center',
    gap: '0.5rem',
  },
  label: {
    color: '#333',
    fontWeight: 500,
  },
  select: {
    padding: '0.5rem',
    borderRadius: '4px',
    border: '1px solid #ccc',
    fontSize: '0.875rem',
    minWidth: '200px',
  },
  summary: {
    marginBottom: '1rem',
    color: '#666',
    fontSize: '0.875rem',
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
    fontSize: '0.875rem',
  },
  td: {
    padding: '12px',
    borderBottom: '1px solid #eee',
    color: '#333',
    fontSize: '0.875rem',
  },
  tr: {
    borderBottom: '1px solid #eee',
  },
  badge: {
    padding: '0.25rem 0.5rem',
    borderRadius: '4px',
    fontSize: '0.75rem',
  },
  errorMsg: {
    marginTop: '4px',
    fontSize: '0.75rem',
    color: '#f44336',
    maxWidth: '200px',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },
  empty: {
    textAlign: 'center',
    padding: '2rem',
    color: '#666',
  },
}