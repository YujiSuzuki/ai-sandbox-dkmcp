import { useState, useEffect } from 'react'
import NoteForm from '../components/NoteForm'
import NoteList from '../components/NoteList'
import apiService from '../services/apiService'

function NotesPage({ user, onLogout }) {
  const [notes, setNotes] = useState([])
  const [loading, setLoading] = useState(true)
  const [showForm, setShowForm] = useState(false)

  useEffect(() => {
    loadNotes()
  }, [])

  const loadNotes = async () => {
    try {
      const data = await apiService.getNotes()
      setNotes(data)
    } catch (err) {
      console.error('Failed to load notes:', err)
    } finally {
      setLoading(false)
    }
  }

  const handleCreateNote = async (title, content) => {
    try {
      await apiService.createNote(title, content)
      await loadNotes()
      setShowForm(false)
    } catch (err) {
      alert('Failed to create note: ' + err.message)
    }
  }

  const handleDeleteNote = async (id) => {
    if (!confirm('Delete this note?')) return

    try {
      await apiService.deleteNote(id)
      await loadNotes()
    } catch (err) {
      alert('Failed to delete note: ' + err.message)
    }
  }

  return (
    <div className="container">
      <div className="header">
        <h1>🔒 SecureNote</h1>
        <p className="subtitle">Your encrypted notes</p>
      </div>

      <div className="card">
        <div className="notes-header">
          <h2>My Notes</h2>
          <div className="user-info">
            <span>👤 {user.username}</span>
            <button onClick={onLogout} className="btn btn-secondary">
              Logout
            </button>
          </div>
        </div>

        {loading ? (
          <div className="loading">Loading notes...</div>
        ) : (
          <>
            {!showForm && (
              <button onClick={() => setShowForm(true)} className="btn btn-primary" style={{marginBottom: '20px'}}>
                ➕ New Note
              </button>
            )}

            {showForm && (
              <NoteForm
                onSubmit={handleCreateNote}
                onCancel={() => setShowForm(false)}
              />
            )}

            <NoteList notes={notes} onDelete={handleDeleteNote} />
          </>
        )}
      </div>
    </div>
  )
}

export default NotesPage
