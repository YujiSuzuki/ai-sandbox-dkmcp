function NoteList({ notes, onDelete }) {
  if (notes.length === 0) {
    return (
      <div className="empty-state">
        <p>📝 No notes yet</p>
        <p style={{ fontSize: '0.9rem', color: '#ccc' }}>Create your first encrypted note!</p>
      </div>
    )
  }

  return (
    <div className="notes-list">
      {notes.map(note => (
        <div key={note.id} className="note-item">
          <h3>{note.title}</h3>
          <p>{note.content}</p>
          <div className="note-meta">
            Created: {new Date(note.createdAt).toLocaleString()}
          </div>
          <div className="note-actions">
            <button onClick={() => onDelete(note.id)} className="btn btn-danger">
              Delete
            </button>
          </div>
        </div>
      ))}
    </div>
  )
}

export default NoteList
