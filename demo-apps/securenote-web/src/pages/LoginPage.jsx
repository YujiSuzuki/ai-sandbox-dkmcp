import { useState } from 'react'
import apiService from '../services/apiService'

function LoginPage({ onLogin }) {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState(null)
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e) => {
    e.preventDefault()
    setError(null)
    setLoading(true)

    try {
      const data = await apiService.login(username, password)
      onLogin(data.token, data.user)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="container">
      <div className="header">
        <h1>🔒 SecureNote</h1>
        <p className="subtitle">Encrypted note-taking with AI-safe secret management</p>
      </div>

      <div className="demo-info">
        <h3>🎯 Demo Application</h3>
        <p>This app demonstrates <strong>DockMCP</strong> - a tool for safely using AI coding assistants with Docker containers.</p>
        <p><strong>Key Feature:</strong> API secrets are encrypted and hidden from AI, but the app works perfectly!</p>
        <p><strong>Demo Credentials:</strong> <code>demo</code> / <code>demo123</code> or <code>alice</code> / <code>alice123</code></p>
      </div>

      <div className="card">
        <h2>Login</h2>
        {error && <div className="error">{error}</div>}
        <form onSubmit={handleSubmit} className="login-form">
          <div className="form-group">
            <label htmlFor="username">Username</label>
            <input
              id="username"
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder="Enter username"
              required
            />
          </div>
          <div className="form-group">
            <label htmlFor="password">Password</label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Enter password"
              required
            />
          </div>
          <button type="submit" className="btn btn-primary" disabled={loading}>
            {loading ? 'Logging in...' : 'Login'}
          </button>
        </form>
      </div>
    </div>
  )
}

export default LoginPage
