import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { apiKeyApi } from '../api/client'

export function ApiKeyCreate() {
  const navigate = useNavigate()
  const [form, setForm] = useState({
    key: '',
    name: '',
  })

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    await apiKeyApi.create({
      key: form.key,
      name: form.name,
    })
    navigate('/keys')
  }

  return (
    <div style={styles.container}>
      <h1 style={styles.title}>+ 新建 API Key</h1>
      <form onSubmit={handleSubmit} style={styles.form}>
        <div style={styles.field}>
          <label style={styles.label}>Key *</label>
          <input
            type="text"
            value={form.key}
            onChange={(e) => setForm({ ...form, key: e.target.value })}
            placeholder="请输入 API Key"
            required
            style={styles.input}
          />
        </div>

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

        <div style={styles.buttons}>
          <button type="submit" style={styles.submitBtn}>创建</button>
          <button type="button" onClick={() => navigate('/keys')} style={styles.cancelBtn}>取消</button>
        </div>
      </form>
    </div>
  )
}

const styles: Record<string, React.CSSProperties> = {
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