import { useState, useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { tokenApi } from '../api/client'

export function TokenEdit() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [loading, setLoading] = useState(true)
  const [form, setForm] = useState({
    name: '',
    max_requests: 0,
    expires_at: '',
    user_id: '',
    description: '',
    enabled: true,
  })

  useEffect(() => {
    if (!id) return
    tokenApi.get(id).then((res) => {
      const token = res.data.token
      setForm({
        name: token.name || '',
        max_requests: token.max_requests || 0,
        expires_at: token.expires_at ? new Date(token.expires_at).toISOString().slice(0, 16) : '',
        user_id: token.user_id || '',
        description: token.description || '',
        enabled: token.enabled ?? true,
      })
    }).finally(() => setLoading(false))
  }, [id])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!id) return
    const data: any = {
      name: form.name,
      max_requests: Number(form.max_requests),
      user_id: form.user_id,
      description: form.description,
      enabled: form.enabled,
    }
    if (form.expires_at) {
      data.expires_at = new Date(form.expires_at).toISOString()
    }

    await tokenApi.update(id, data)
    navigate('/tokens')
  }

  if (loading) return <div style={styles.loading}>加载中...</div>

  return (
    <div style={styles.container}>
      <h1 style={styles.title}>✏️ 编辑 Token</h1>
      <form onSubmit={handleSubmit} style={styles.form}>
        <div style={styles.field}>
          <label style={styles.label}>名称 *</label>
          <input
            type="text"
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
            placeholder="例如：测试环境"
            required
            style={styles.input}
          />
        </div>

        <div style={styles.field}>
          <label style={styles.label}>最大请求次数</label>
          <input
            type="number"
            value={form.max_requests}
            onChange={(e) => setForm({ ...form, max_requests: Number(e.target.value) })}
            placeholder="0 = 不限制"
            style={styles.input}
          />
          <small style={styles.hint}>设置为 0 表示不限制请求次数</small>
        </div>

        <div style={styles.field}>
          <label style={styles.label}>过期时间</label>
          <input
            type="datetime-local"
            value={form.expires_at}
            onChange={(e) => setForm({ ...form, expires_at: e.target.value })}
            style={styles.input}
          />
          <small style={styles.hint}>留空表示永不过期</small>
        </div>

        <div style={styles.field}>
          <label style={styles.label}>用户 ID</label>
          <input
            type="text"
            value={form.user_id}
            onChange={(e) => setForm({ ...form, user_id: e.target.value })}
            placeholder="可选，用于区分用户"
            style={styles.input}
          />
        </div>

        <div style={styles.field}>
          <label style={styles.label}>描述</label>
          <textarea
            value={form.description}
            onChange={(e) => setForm({ ...form, description: e.target.value })}
            placeholder="可选，备注信息"
            style={{ ...styles.input, minHeight: '80px' }}
          />
        </div>

        <div style={styles.field}>
          <label style={styles.checkboxLabel}>
            <input
              type="checkbox"
              checked={form.enabled}
              onChange={(e) => setForm({ ...form, enabled: e.target.checked })}
            />
            启用 Token
          </label>
        </div>

        <div style={styles.buttons}>
          <button type="submit" style={styles.submitBtn}>保存</button>
          <button type="button" onClick={() => navigate('/tokens')} style={styles.cancelBtn}>取消</button>
        </div>
      </form>
    </div>
  )
}

const styles: Record<string, React.CSSProperties> = {
  loading: {
    textAlign: 'center',
    padding: '2rem',
    color: '#666',
  },
  container: {
    maxWidth: '600px',
  },
  title: {
    marginBottom: '1.5rem',
    color: '#333',
  },
  form: {
    background: '#fff',
    padding: '1.5rem',
    borderRadius: '8px',
    border: '1px solid #ddd',
  },
  field: {
    marginBottom: '1rem',
    display: 'flex',
    flexDirection: 'column',
    gap: '0.5rem',
  },
  label: {
    fontWeight: 500,
    color: '#333',
  },
  input: {
    padding: '0.75rem',
    borderRadius: '4px',
    border: '1px solid #ccc',
    background: '#fff',
    color: '#333',
    fontSize: '1rem',
  },
  hint: {
    color: '#888',
    fontSize: '0.75rem',
  },
  checkboxLabel: {
    display: 'flex',
    alignItems: 'center',
    gap: '0.5rem',
    cursor: 'pointer',
    color: '#333',
  },
  buttons: {
    display: 'flex',
    gap: '1rem',
    marginTop: '1rem',
  },
  submitBtn: {
    flex: 1,
    padding: '0.75rem',
    borderRadius: '4px',
    border: 'none',
    background: '#4CAF50',
    color: '#fff',
    fontSize: '1rem',
    cursor: 'pointer',
  },
  cancelBtn: {
    flex: 1,
    padding: '0.75rem',
    borderRadius: '4px',
    border: 'none',
    background: '#e0e0e0',
    color: '#333',
    fontSize: '1rem',
    cursor: 'pointer',
  },
}