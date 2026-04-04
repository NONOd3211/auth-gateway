import { useState, useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { apiKeyApi } from '../api/client'

export function ApiKeyEdit() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [loading, setLoading] = useState(true)
  const [form, setForm] = useState({
    name: '',
    allowed_models: '',
  })

  useEffect(() => {
    if (!id) return
    apiKeyApi.list().then((res) => {
      const key = res.data.keys.find((k: any) => k.id === id)
      if (key) {
        setForm({
          name: key.name || '',
          allowed_models: key.allowed_models || '',
        })
      }
    }).finally(() => setLoading(false))
  }, [id])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!id) return
    await apiKeyApi.update(id, {
      name: form.name,
      allowed_models: form.allowed_models,
    })
    navigate('/keys')
  }

  if (loading) return <div style={styles.loading}>加载中...</div>

  return (
    <div style={styles.container}>
      <h1 style={styles.title}>✏️ 编辑 API Key</h1>
      <form onSubmit={handleSubmit} style={styles.form}>
        <div style={styles.field}>
          <label style={styles.label}>名称</label>
          <input
            type="text"
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
            placeholder="例如：生产环境 Key"
            style={styles.input}
          />
        </div>

        <div style={styles.field}>
          <label style={styles.label}>允许的模型</label>
          <input
            type="text"
            value={form.allowed_models}
            onChange={(e) => setForm({ ...form, allowed_models: e.target.value })}
            placeholder="例如：MiniMax-M2.7,MiniMax-M2（逗号分隔，留空则不限制）"
            style={styles.input}
          />
          <small style={styles.hint}>多个模型用逗号分隔</small>
        </div>

        <div style={styles.buttons}>
          <button type="submit" style={styles.submitBtn}>保存</button>
          <button type="button" onClick={() => navigate('/keys')} style={styles.cancelBtn}>取消</button>
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