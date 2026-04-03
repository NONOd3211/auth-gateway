import { useAuthStore } from '../store/auth'
import { useNavigate, Link } from 'react-router-dom'

export function Navbar() {
  const { logout } = useAuthStore()
  const navigate = useNavigate()

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  return (
    <nav style={styles.nav}>
      <div style={styles.logo}>
        <Link to="/" style={styles.logoLink}>🔐 Auth Gateway</Link>
      </div>
      <div style={styles.links}>
        <Link to="/" style={styles.link}>仪表盘</Link>
        <Link to="/tokens" style={styles.link}>Token 管理</Link>
        <Link to="/keys" style={styles.link}>API Keys</Link>
        <button onClick={handleLogout} style={styles.logoutBtn}>退出</button>
      </div>
    </nav>
  )
}

const styles: Record<string, React.CSSProperties> = {
  nav: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: '1rem 0',
    marginBottom: '2rem',
    borderBottom: '1px solid #ddd',
  },
  logo: {
    fontSize: '1.25rem',
    fontWeight: 'bold',
  },
  logoLink: {
    color: '#333',
    textDecoration: 'none',
  },
  links: {
    display: 'flex',
    gap: '1rem',
    alignItems: 'center',
  },
  link: {
    color: '#1976D2',
    textDecoration: 'none',
    padding: '0.5rem 1rem',
    borderRadius: '4px',
  },
  logoutBtn: {
    background: '#dc3545',
    color: '#fff',
    border: 'none',
    padding: '0.5rem 1rem',
    borderRadius: '4px',
    cursor: 'pointer',
  },
}