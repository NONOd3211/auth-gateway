import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { tokenApi, usageApi, UsageStats } from '../api/client'

export function Dashboard() {
  const [stats, setStats] = useState<UsageStats | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    usageApi.stats()
      .then((res) => setStats(res.data))
      .finally(() => setLoading(false))
  }, [])

  if (loading) return <div>加载中...</div>

  return (
    <div>
      <h1 style={styles.title}>📊 仪表盘</h1>

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
      </div>

      <div style={styles.quickActions}>
        <Link to="/tokens" style={styles.linkCard}>
          <h3>📝 管理 Token</h3>
          <p>创建、编辑、删除 API Token</p>
        </Link>
        <Link to="/usage" style={styles.linkCard}>
          <h3>📈 使用统计</h3>
          <p>查看详细使用报表</p>
        </Link>
      </div>
    </div>
  )
}

const styles: Record<string, React.CSSProperties> = {
  title: {
    marginBottom: '1.5rem',
  },
  grid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
    gap: '1rem',
    marginBottom: '2rem',
  },
  card: {
    background: '#333',
    padding: '1.5rem',
    borderRadius: '8px',
  },
  number: {
    fontSize: '2rem',
    fontWeight: 'bold',
    marginTop: '0.5rem',
  },
  quickActions: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(250px, 1fr))',
    gap: '1rem',
  },
  linkCard: {
    background: '#333',
    padding: '1.5rem',
    borderRadius: '8px',
    textDecoration: 'none',
    color: '#fff',
    display: 'block',
  },
}
