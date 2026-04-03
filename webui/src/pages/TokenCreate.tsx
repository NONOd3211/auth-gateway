import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { tokenApi } from '../api/client'

export function TokenCreate() {
  const navigate = useNavigate()
  const [form, setForm] = useState({
    name: '',
    max_requests: 0,
    expires_at: '',
    user_id: '',
    description: '',
    hourly_limit: false,
    hourly_requests: 0,
    weekly_limit: false,
    weekly_requests: 0,
  })
  const [createdToken, setCreatedToken] = useState<string | null>(null)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    const data: any = {
      name: form.name,
      max_requests: Number(form.max_requests),
      user_id: form.user_id,
      description: form.description,
      hourly_limit: form.hourly_limit,
      weekly_limit: form.weekly_limit,
    }
    if (form.expires_at) {
      data.expires_at = new Date(form.expires_at).toISOString()
    }
    if (form.hourly_limit) {
      data.hourly_requests = Number(form.hourly_requests)
    }
    if (form.weekly_limit) {
      data.weekly_requests = Number(form.weekly_requests)
    }

    const res = await tokenApi.create(data)
    setCreatedToken(res.data.token.token)
  }

  if (createdToken) {
    return (
      <div style={styles.container}>
        <h1 style={styles.successTitle}>✅ Token 创建成功</h1>
        <div style={styles.successCard}>
          <p>请复制保存以下 Token，关闭后将无法再次查看完整 Token：</p>
          <code style={styles.tokenCode}>{createdToken}</code>
          <button
            onClick={() => {
              navigator.clipboard.writeText(createdToken).then(() => {
                alert('已复制到剪贴板')
              }).catch(() => {
                alert('复制失败，请手动复制')
              })
            }}
            style={styles.copyBtn}
          >
            复制 Token
          </button>
          <button onClick={() => navigate('/tokens')} style={styles.backBtn}>
            返回列表
          </button>
        </div>
      </div>
    )
  }

  return (
    <div style={styles.container}>
      <h1 style={styles.title}>➕ 新建 Token</h1>
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
              checked={form.hourly_limit}
              onChange={(e) => setForm({ ...form, hourly_limit: e.target.checked })}
            />
            启用小时请求限制
          </label>
          {form.hourly_limit && (
            <input
              type="number"
              value={form.hourly_requests}
              onChange={(e) => setForm({ ...form, hourly_requests: Number(e.target.value) })}
              placeholder="小时请求次数"
              style={styles.input}
            />
          )}
          <small style={styles.hint}>启用后，每小时请求次数达到限制后无法使用</small>
        </div>

        <div style={styles.field}>
          <label style={styles.checkboxLabel}>
            <input
              type="checkbox"
              checked={form.weekly_limit}
              onChange={(e) => setForm({ ...form, weekly_limit: e.target.checked })}
            />
            启用周请求限制
          </label>
          {form.weekly_limit && (
            <input
              type="number"
              value={form.weekly_requests}
              onChange={(e) => setForm({ ...form, weekly_requests: Number(e.target.value) })}
              placeholder="每周请求次数"
              style={styles.input}
            />
          )}
          <small style={styles.hint}>启用后，每周请求次数达到限制后无法使用</small>
        </div>

        <div style={styles.buttons}>
          <button type="submit" style={styles.submitBtn}>创建 Token</button>
          <button type="button" onClick={() => navigate('/tokens')} style={styles.cancelBtn}>取消</button>
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
  successTitle: {
    marginBottom: '1rem',
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
  successCard: {
    background: '#fff',
    padding: '1.5rem',
    borderRadius: '8px',
    textAlign: 'center',
    border: '1px solid #ddd',
  },
  tokenCode: {
    display: 'block',
    background: '#f5f5f5',
    padding: '1rem',
    borderRadius: '4px',
    margin: '1rem 0',
    wordBreak: 'break-all',
    fontSize: '1rem',
    border: '1px solid #ddd',
  },
  copyBtn: {
    background: '#2196F3',
    color: '#fff',
    border: 'none',
    padding: '0.75rem 1.5rem',
    borderRadius: '4px',
    cursor: 'pointer',
    marginRight: '0.5rem',
  },
  backBtn: {
    background: '#e0e0e0',
    color: '#333',
    border: 'none',
    padding: '0.75rem 1.5rem',
    borderRadius: '4px',
    cursor: 'pointer',
  },
}