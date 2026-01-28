import { useState, useEffect } from 'react'
import LoginPage from './pages/LoginPage'
import NotesPage from './pages/NotesPage'
import apiService from './services/apiService'

function App() {
  const [token, setToken] = useState(null)
  const [user, setUser] = useState(null)

  useEffect(() => {
    // Check if already logged in
    const savedToken = localStorage.getItem('token')
    const savedUser = localStorage.getItem('user')
    if (savedToken && savedUser) {
      setToken(savedToken)
      setUser(JSON.parse(savedUser))
      apiService.setToken(savedToken)
    }
  }, [])

  const handleLogin = (loginToken, loginUser) => {
    setToken(loginToken)
    setUser(loginUser)
    localStorage.setItem('token', loginToken)
    localStorage.setItem('user', JSON.stringify(loginUser))
    apiService.setToken(loginToken)
  }

  const handleLogout = () => {
    setToken(null)
    setUser(null)
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    apiService.setToken(null)
  }

  return (
    <div className="app">
      {!token ? (
        <LoginPage onLogin={handleLogin} />
      ) : (
        <NotesPage user={user} onLogout={handleLogout} />
      )}
    </div>
  )
}

export default App
