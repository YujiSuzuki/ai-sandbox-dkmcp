const express = require('express');
const authMiddleware = require('../middleware/auth');
const EncryptionService = require('../services/encryption');

const router = express.Router();

// In-memory storage (in production, use a database)
let notes = [];
let nextId = 1;

// Apply auth middleware to all routes
router.use(authMiddleware);

// GET /api/notes - List all notes for current user
router.get('/', (req, res) => {
  if (!req.app.locals.secretsLoaded) {
    return res.status(500).json({ error: 'Cannot decrypt notes without secrets' });
  }

  const encryptionService = new EncryptionService(req.app.locals.encryptionKey);

  const userNotes = notes
    .filter(note => note.userId === req.user.id)
    .map(note => {
      try {
        return {
          id: note.id,
          title: note.title,
          content: encryptionService.decrypt(note.encryptedContent),
          createdAt: note.createdAt,
          updatedAt: note.updatedAt
        };
      } catch (error) {
        return {
          id: note.id,
          title: note.title,
          content: '[Decryption failed]',
          createdAt: note.createdAt,
          updatedAt: note.updatedAt
        };
      }
    });

  res.json(userNotes);
});

// GET /api/notes/:id - Get specific note
router.get('/:id', (req, res) => {
  const noteId = parseInt(req.params.id);
  const note = notes.find(n => n.id === noteId && n.userId === req.user.id);

  if (!note) {
    return res.status(404).json({ error: 'Note not found' });
  }

  if (!req.app.locals.secretsLoaded) {
    return res.status(500).json({ error: 'Cannot decrypt note without secrets' });
  }

  const encryptionService = new EncryptionService(req.app.locals.encryptionKey);

  try {
    res.json({
      id: note.id,
      title: note.title,
      content: encryptionService.decrypt(note.encryptedContent),
      createdAt: note.createdAt,
      updatedAt: note.updatedAt
    });
  } catch (error) {
    res.status(500).json({ error: 'Decryption failed' });
  }
});

// POST /api/notes - Create new note
router.post('/', (req, res) => {
  const { title, content } = req.body;

  if (!title || !content) {
    return res.status(400).json({ error: 'Title and content required' });
  }

  if (!req.app.locals.secretsLoaded) {
    return res.status(500).json({ error: 'Cannot encrypt notes without secrets' });
  }

  const encryptionService = new EncryptionService(req.app.locals.encryptionKey);

  const note = {
    id: nextId++,
    userId: req.user.id,
    title,
    encryptedContent: encryptionService.encrypt(content),
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString()
  };

  notes.push(note);

  res.status(201).json({
    id: note.id,
    title: note.title,
    content,
    createdAt: note.createdAt,
    updatedAt: note.updatedAt
  });
});

// PUT /api/notes/:id - Update note
router.put('/:id', (req, res) => {
  const noteId = parseInt(req.params.id);
  const { title, content } = req.body;

  const noteIndex = notes.findIndex(n => n.id === noteId && n.userId === req.user.id);

  if (noteIndex === -1) {
    return res.status(404).json({ error: 'Note not found' });
  }

  if (!req.app.locals.secretsLoaded) {
    return res.status(500).json({ error: 'Cannot encrypt notes without secrets' });
  }

  const encryptionService = new EncryptionService(req.app.locals.encryptionKey);

  if (title) notes[noteIndex].title = title;
  if (content) notes[noteIndex].encryptedContent = encryptionService.encrypt(content);
  notes[noteIndex].updatedAt = new Date().toISOString();

  res.json({
    id: notes[noteIndex].id,
    title: notes[noteIndex].title,
    content: content || encryptionService.decrypt(notes[noteIndex].encryptedContent),
    createdAt: notes[noteIndex].createdAt,
    updatedAt: notes[noteIndex].updatedAt
  });
});

// DELETE /api/notes/:id - Delete note
router.delete('/:id', (req, res) => {
  const noteId = parseInt(req.params.id);
  const noteIndex = notes.findIndex(n => n.id === noteId && n.userId === req.user.id);

  if (noteIndex === -1) {
    return res.status(404).json({ error: 'Note not found' });
  }

  notes.splice(noteIndex, 1);
  res.status(204).send();
});

module.exports = router;
