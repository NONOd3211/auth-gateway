import { useEffect, useState } from 'react'
import { usageApi } from '../api/client'

export function UsageStats() {
  const [stats, setStats] = useState<any>(null)
  const [daily, setDaily] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([
      usageApi.stats(),
      usageApi.daily(),
    ])
      .then(([statsRes, dailyRes]) => {
        setStats(statsRes.data)
        setDaily(dailyRes.data.daily)
      })
      .finally(() => setLoading(false))
  }, [])

  if (loading) return <div>加载中...</div>

  return (
    <div>
      <h1 style={styles.title}>📈 使用统计</h1>

      <div style={styles.grid}>
        <div style={styles.card}>
          <h3>总请求数</h3>
          <p style={styles.number}>{stats?.total_requests ?? 0}</p>
        </div>
        <div style={styles.card}>
          <h3>成功请求</h3>
          <p style={{ ...styles.number, color: '#4CAF50' }}>{stats?.success_count ?? 0}</p>
        </div>
        <div style={styles.card}>
          <h3>失败请求</h3>
          <p style={{ ...styles.number, color: '#f44336' }}>{stats?.failure_count ?? 0}</p>
        </div>
        <div style={styles.card}>
          <h3>总 Token 数</h3>
          <p style={styles.number}>{stats?.total_tokens ?? 0}</p>
        </div>
        <div style={styles.card}>
          <h3>输入 Token</h3>
          <p style={styles.number}>{stats?.input_tokens ?? 0}</p>
        </div>
        <div style={styles.card}>
          <h3>输出 Token</h3>
          <p style={styles.number}>{stats?.output_tokens ?? 0}</p>
        </div>
      </div>

      <h2 style={styles.subtitle}>每日统计</h2>
      <table style={styles.table}>
        <thead>
          <tr>
            <th>日期</th>
            <th>请求数</th>
            <th>总 Token</th>
          </tr>
        </thead>
        <tbody>
          {daily.map((d) => (
            <tr key={d.date}>
              <td>{d.date}</td>
              <td>{d.requests}</td>
              <td>{d.total_tokens}</td>
            </tr>
          ))}
        </tbody>
      </table>

      {daily.length === 0 && (
        <p style={styles.empty}>暂无统计数据</p>
      )}
    </div>
  )
}

const styles: Record<string, React.CSSProperties> = {
  title: {
    marginBottom: '1.5rem',
  },
  subtitle: {
    marginTop: '2rem',
    marginBottom: '1rem',
  },
  grid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))',
    gap: '1rem',
    marginBottom: '2rem',
  },
  card: {
    background: '#333',
    padding: '1.5rem',
    borderRadius: '8px',
    textAlign: 'center',
  },
  number: {
    fontSize: '1.75rem',
    fontWeight: 'bold',
    marginTop: '0.5rem',
  },
  table: {
    width: '100%',
    borderCollapse: 'collapse',
    background: '#333',
    borderRadius: '8px',
    overflow: 'hidden',
  },
  empty: {
    textAlign: 'center',
    padding: '2rem',
    color: '#888',
  },
}
