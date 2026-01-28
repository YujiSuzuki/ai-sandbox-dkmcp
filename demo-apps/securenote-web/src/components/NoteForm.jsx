import { useState } from 'react'

function NoteForm({ onSubmit, onCancel }) {
  const [title, setTitle] = useState('')
  const [content, setContent] = useState('')

  const handleSubmit = (e) => {
    e.preventDefault()
    if (title.trim() && content.trim()) {
      onSubmit(title, content)
      setTitle('')
      setContent('')
    }
  }

  return (
    <form onSubmit={handleSubmit} className="note-form">
      <div className="form-group">
        <label htmlFor="title">Title</label>
        <input
          id="title"
          type="text"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="Note title"
          required
        />
      </div>
      <div className="form-group">
        <label htmlFor="content">Content</label>
        <textarea
          id="content"
          value={content}
          onChange={(e) => setContent(e.target.value)}
          placeholder="Write your note here... (will be encrypted)"
          required
        />
      </div>
      <div style={{ display: 'flex', gap: '10px' }}>
        <button type="submit" className="btn btn-primary">
          Save Note
        </button>
        <button type="button" onClick={onCancel} className="btn btn-secondary">
          Cancel
        </button>
      </div>
    </form>
  )
}

export default NoteForm
