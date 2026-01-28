const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api'

class ApiService {
  constructor() {
    this.token = null
  }

  setToken(token) {
    this.token = token
  }

  async request(endpoint, options = {}) {
    const headers = {
      'Content-Type': 'application/json',
      ...options.headers
    }

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`
    }

    const response = await fetch(`${API_BASE_URL}${endpoint}`, {
      ...options,
      headers
    })

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Network error' }))
      throw new Error(error.error || 'Request failed')
    }

    if (response.status === 204) {
      return null
    }

    return response.json()
  }

  // Auth
  async login(username, password) {
    return this.request('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password })
    })
  }

  // Notes
  async getNotes() {
    return this.request('/notes')
  }

  async getNote(id) {
    return this.request(`/notes/${id}`)
  }

  async createNote(title, content) {
    return this.request('/notes', {
      method: 'POST',
      body: JSON.stringify({ title, content })
    })
  }

  async updateNote(id, title, content) {
    return this.request(`/notes/${id}`, {
      method: 'PUT',
      body: JSON.stringify({ title, content })
    })
  }

  async deleteNote(id) {
    return this.request(`/notes/${id}`, {
      method: 'DELETE'
    })
  }

  // Demo
  async getSecretsStatus() {
    return this.request('/demo/secrets-status')
  }
}

export default new ApiService()
