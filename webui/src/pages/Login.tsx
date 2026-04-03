import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '../store/auth'

export function Login() {
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const { login } = useAuthStore()
  const navigate = useNavigate()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      const res = await fetch('/api/admin/tokens', {
        headers: { Authorization: `Bearer ${password}` },
      })
      if (res.ok) {
        login(password)
        navigate('/')
      } else {
        setError('密码错误')
      }
    } catch {
      setError('连接失败，请检查网关是否运行')
    }
  }

  return (
    <div style={styles.container}>
      <div style={styles.card}>
        <h1 style={styles.title}>🔐 Auth Gateway</h1>
        <p style={styles.subtitle}>请输入管理员密码</p>
        <form onSubmit={handleSubmit} style={styles.form}>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="管理员密码"
            style={styles.input}
          />
          {error && <p style={styles.error}>{error}</p>}
          <button type="submit" style={styles.button}>登录</button>
        </form>
      </div>
    </div>
  )
}

const styles: Record<string, React.CSSProperties> = {
  container: {
    minHeight: '100vh',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
  },
  card: {
    background: '#333',
    padding: '2rem',
    borderRadius: '8px',
    width: '100%',
    maxWidth: '400px',
  },
  title: {
    textAlign: 'center',
    marginBottom: '0.5rem',
  },
  subtitle: {
    textAlign: 'center',
    color: '#888',
    marginBottom: '1.5rem',
  },
  form: {
    display: 'flex',
    flexDirection: 'column',
    gap: '1rem',
  },
  input: {
    padding: '0.75rem',
    borderRadius: '4px',
    border: '1px solid #555',
    background: '#222',
    color: '#fff',
    fontSize: '1rem',
  },
  button: {
    padding: '0.75rem',
    borderRadius: '4px',
    border: 'none',
    background: '#4CAF50',
    color: '#fff',
    fontSize: '1rem',
    cursor: 'pointer',
  },
  error: {
    color: '#f44336',
    textAlign: 'center',
    fontSize: '0.875rem',
  },
}
