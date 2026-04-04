import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { Login } from './pages/Login'
import { Dashboard } from './pages/Dashboard'
import { TokenList } from './pages/TokenList'
import { TokenCreate } from './pages/TokenCreate'
import { TokenEdit } from './pages/TokenEdit'
import { ApiKeyList } from './pages/ApiKeyList'
import { ApiKeyCreate } from './pages/ApiKeyCreate'
import { UsageEvents } from './pages/UsageEvents'
import { UserPanel } from './pages/UserPanel'
import { Navbar } from './components/Navbar'

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />

        {/* Admin routes */}
        <Route
          path="/*"
          element={
            <>
              <Navbar />
              <Routes>
                <Route path="/" element={<Dashboard />} />
                <Route path="/tokens" element={<TokenList />} />
                <Route path="/tokens/create" element={<TokenCreate />} />
                <Route path="/tokens/:id" element={<TokenEdit />} />
                <Route path="/keys" element={<ApiKeyList />} />
                <Route path="/keys/create" element={<ApiKeyCreate />} />
                <Route path="/usage" element={<UsageEvents />} />
              </Routes>
            </>
          }
        />

        {/* User routes */}
        <Route path="/user/*" element={<UserPanel />} />
      </Routes>
    </BrowserRouter>
  )
}

export default App
