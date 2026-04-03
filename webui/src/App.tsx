import { useState, useEffect } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { Login } from './pages/Login'
import { Dashboard } from './pages/Dashboard'
import { TokenList } from './pages/TokenList'
import { TokenCreate } from './pages/TokenCreate'
import { UsageStats } from './pages/UsageStats'
import { UserPanel } from './pages/UserPanel'
import { Navbar } from './components/Navbar'
import { useAuthStore } from './store/auth'

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = useAuthStore()
  return isAuthenticated ? <>{children}</> : <Navigate to="/login" />
}

function App() {
  const [isAdmin, setIsAdmin] = useState(false)

  useEffect(() => {
    // Check URL for admin code
    const params = new URLSearchParams(window.location.search)
    const code = params.get('code')
    if (code) {
      setIsAdmin(true)
    }
  }, [])

  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />

        {/* Admin routes - require ?code=xxx */}
        {isAdmin && (
          <Route
            path="/*"
            element={
              <ProtectedRoute>
                <Navbar />
                <Routes>
                  <Route path="/" element={<Dashboard />} />
                  <Route path="/tokens" element={<TokenList />} />
                  <Route path="/tokens/create" element={<TokenCreate />} />
                  <Route path="/usage" element={<UsageStats />} />
                </Routes>
              </ProtectedRoute>
            }
          />
        )}

        {/* User routes - default, no admin code needed */}
        {!isAdmin && (
          <Route path="/*" element={<UserPanel />} />
        )}
      </Routes>
    </BrowserRouter>
  )
}

export default App
