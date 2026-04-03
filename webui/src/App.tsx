import { useState } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { Login } from './pages/Login'
import { Dashboard } from './pages/Dashboard'
import { TokenList } from './pages/TokenList'
import { TokenCreate } from './pages/TokenCreate'
import { UsageStats } from './pages/UsageStats'
import { Navbar } from './components/Navbar'
import { useAuthStore } from './store/auth'

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = useAuthStore()
  return isAuthenticated ? <>{children}</> : <Navigate to="/login" />
}

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />
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
      </Routes>
    </BrowserRouter>
  )
}

export default App
