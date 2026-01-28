const request = require('supertest');
const express = require('express');
const bodyParser = require('body-parser');
const jwt = require('jsonwebtoken');
const notesRoutes = require('../../routes/notes');

describe('Notes Routes', () => {
  let app;
  const jwtSecret = 'test-secret-key';
  const encryptionKey = 'test-encryption-key-32-bytes';
  const user1Token = jwt.sign({ id: 1, username: 'user1' }, jwtSecret);
  const user2Token = jwt.sign({ id: 2, username: 'user2' }, jwtSecret);

  beforeEach(() => {
    app = express();
    app.use(bodyParser.json());
    app.locals.jwtSecret = jwtSecret;
    app.locals.encryptionKey = encryptionKey;
    app.locals.secretsLoaded = true;
    app.use('/notes', notesRoutes);
  });

  describe('GET /notes', () => {
    it('should return 401 when no token is provided', async () => {
      const response = await request(app).get('/notes');

      expect(response.status).toBe(401);
      expect(response.body).toEqual({ error: 'No token provided' });
    });

    it('should return empty array when user has no notes', async () => {
      const response = await request(app)
        .get('/notes')
        .set('Authorization', `Bearer ${user1Token}`);

      expect(response.status).toBe(200);
      expect(response.body).toEqual([]);
    });

    it('should return 500 when secrets are not loaded', async () => {
      app.locals.secretsLoaded = false;

      const response = await request(app)
        .get('/notes')
        .set('Authorization', `Bearer ${user1Token}`);

      expect(response.status).toBe(500);
      expect(response.body).toEqual({ error: 'Cannot decrypt notes without secrets' });
    });
  });

  describe('POST /notes', () => {
    it('should create a new note', async () => {
      const response = await request(app)
        .post('/notes')
        .set('Authorization', `Bearer ${user1Token}`)
        .send({
          title: 'Test Note',
          content: 'This is a test note'
        });

      expect(response.status).toBe(201);
      expect(response.body).toHaveProperty('id');
      expect(response.body.title).toBe('Test Note');
      expect(response.body.content).toBe('This is a test note');
      expect(response.body).toHaveProperty('createdAt');
      expect(response.body).toHaveProperty('updatedAt');
    });

    it('should return 400 when title is missing', async () => {
      const response = await request(app)
        .post('/notes')
        .set('Authorization', `Bearer ${user1Token}`)
        .send({
          content: 'This is a test note'
        });

      expect(response.status).toBe(400);
      expect(response.body).toEqual({ error: 'Title and content required' });
    });

    it('should return 400 when content is missing', async () => {
      const response = await request(app)
        .post('/notes')
        .set('Authorization', `Bearer ${user1Token}`)
        .send({
          title: 'Test Note'
        });

      expect(response.status).toBe(400);
      expect(response.body).toEqual({ error: 'Title and content required' });
    });

    it('should return 400 when both title and content are missing', async () => {
      const response = await request(app)
        .post('/notes')
        .set('Authorization', `Bearer ${user1Token}`)
        .send({});

      expect(response.status).toBe(400);
      expect(response.body).toEqual({ error: 'Title and content required' });
    });

    it('should return 401 when no token is provided', async () => {
      const response = await request(app)
        .post('/notes')
        .send({
          title: 'Test Note',
          content: 'Content'
        });

      expect(response.status).toBe(401);
      expect(response.body).toEqual({ error: 'No token provided' });
    });

    it('should return 500 when secrets are not loaded', async () => {
      app.locals.secretsLoaded = false;

      const response = await request(app)
        .post('/notes')
        .set('Authorization', `Bearer ${user1Token}`)
        .send({
          title: 'Test Note',
          content: 'Content'
        });

      expect(response.status).toBe(500);
      expect(response.body).toEqual({ error: 'Cannot encrypt notes without secrets' });
    });

    it('should handle long content', async () => {
      const longContent = 'A'.repeat(10000);

      const response = await request(app)
        .post('/notes')
        .set('Authorization', `Bearer ${user1Token}`)
        .send({
          title: 'Long Note',
          content: longContent
        });

      expect(response.status).toBe(201);
      expect(response.body.content).toBe(longContent);
    });

    it('should handle unicode content', async () => {
      const response = await request(app)
        .post('/notes')
        .set('Authorization', `Bearer ${user1Token}`)
        .send({
          title: 'Unicode Note',
          content: '你好世界 🚀 こんにちは'
        });

      expect(response.status).toBe(201);
      expect(response.body.content).toBe('你好世界 🚀 こんにちは');
    });
  });

  describe('GET /notes/:id', () => {
    let noteId;

    beforeEach(async () => {
      const response = await request(app)
        .post('/notes')
        .set('Authorization', `Bearer ${user1Token}`)
        .send({
          title: 'Test Note',
          content: 'Test content'
        });
      noteId = response.body.id;
    });

    it('should retrieve a specific note', async () => {
      const response = await request(app)
        .get(`/notes/${noteId}`)
        .set('Authorization', `Bearer ${user1Token}`);

      expect(response.status).toBe(200);
      expect(response.body.id).toBe(noteId);
      expect(response.body.title).toBe('Test Note');
      expect(response.body.content).toBe('Test content');
    });

    it('should return 404 when note does not exist', async () => {
      const response = await request(app)
        .get('/notes/99999')
        .set('Authorization', `Bearer ${user1Token}`);

      expect(response.status).toBe(404);
      expect(response.body).toEqual({ error: 'Note not found' });
    });

    it('should return 404 when user tries to access another user\'s note', async () => {
      const response = await request(app)
        .get(`/notes/${noteId}`)
        .set('Authorization', `Bearer ${user2Token}`);

      expect(response.status).toBe(404);
      expect(response.body).toEqual({ error: 'Note not found' });
    });

    it('should return 401 when no token is provided', async () => {
      const response = await request(app).get(`/notes/${noteId}`);

      expect(response.status).toBe(401);
      expect(response.body).toEqual({ error: 'No token provided' });
    });

    it('should return 500 when secrets are not loaded', async () => {
      app.locals.secretsLoaded = false;

      const response = await request(app)
        .get(`/notes/${noteId}`)
        .set('Authorization', `Bearer ${user1Token}`);

      expect(response.status).toBe(500);
      expect(response.body).toEqual({ error: 'Cannot decrypt note without secrets' });
    });
  });

  describe('PUT /notes/:id', () => {
    let noteId;

    beforeEach(async () => {
      const response = await request(app)
        .post('/notes')
        .set('Authorization', `Bearer ${user1Token}`)
        .send({
          title: 'Original Title',
          content: 'Original content'
        });
      noteId = response.body.id;
    });

    it('should update note title', async () => {
      const response = await request(app)
        .put(`/notes/${noteId}`)
        .set('Authorization', `Bearer ${user1Token}`)
        .send({
          title: 'Updated Title'
        });

      expect(response.status).toBe(200);
      expect(response.body.title).toBe('Updated Title');
    });

    it('should update note content', async () => {
      const response = await request(app)
        .put(`/notes/${noteId}`)
        .set('Authorization', `Bearer ${user1Token}`)
        .send({
          content: 'Updated content'
        });

      expect(response.status).toBe(200);
      expect(response.body.content).toBe('Updated content');
    });

    it('should update both title and content', async () => {
      const response = await request(app)
        .put(`/notes/${noteId}`)
        .set('Authorization', `Bearer ${user1Token}`)
        .send({
          title: 'New Title',
          content: 'New content'
        });

      expect(response.status).toBe(200);
      expect(response.body.title).toBe('New Title');
      expect(response.body.content).toBe('New content');
    });

    it('should return 404 when note does not exist', async () => {
      const response = await request(app)
        .put('/notes/99999')
        .set('Authorization', `Bearer ${user1Token}`)
        .send({
          title: 'Updated Title'
        });

      expect(response.status).toBe(404);
      expect(response.body).toEqual({ error: 'Note not found' });
    });

    it('should prevent user from updating another user\'s note', async () => {
      const response = await request(app)
        .put(`/notes/${noteId}`)
        .set('Authorization', `Bearer ${user2Token}`)
        .send({
          title: 'Updated Title'
        });

      expect(response.status).toBe(404);
      expect(response.body).toEqual({ error: 'Note not found' });
    });

    it('should return 401 when no token is provided', async () => {
      const response = await request(app)
        .put(`/notes/${noteId}`)
        .send({
          title: 'Updated Title'
        });

      expect(response.status).toBe(401);
      expect(response.body).toEqual({ error: 'No token provided' });
    });
  });

  describe('DELETE /notes/:id', () => {
    let noteId;

    beforeEach(async () => {
      const response = await request(app)
        .post('/notes')
        .set('Authorization', `Bearer ${user1Token}`)
        .send({
          title: 'Note to delete',
          content: 'This will be deleted'
        });
      noteId = response.body.id;
    });

    it('should delete a note', async () => {
      const response = await request(app)
        .delete(`/notes/${noteId}`)
        .set('Authorization', `Bearer ${user1Token}`);

      expect(response.status).toBe(204);

      // Verify note is deleted
      const getResponse = await request(app)
        .get(`/notes/${noteId}`)
        .set('Authorization', `Bearer ${user1Token}`);

      expect(getResponse.status).toBe(404);
    });

    it('should return 404 when note does not exist', async () => {
      const response = await request(app)
        .delete('/notes/99999')
        .set('Authorization', `Bearer ${user1Token}`);

      expect(response.status).toBe(404);
      expect(response.body).toEqual({ error: 'Note not found' });
    });

    it('should prevent user from deleting another user\'s note', async () => {
      const response = await request(app)
        .delete(`/notes/${noteId}`)
        .set('Authorization', `Bearer ${user2Token}`);

      expect(response.status).toBe(404);
      expect(response.body).toEqual({ error: 'Note not found' });
    });

    it('should return 401 when no token is provided', async () => {
      const response = await request(app).delete(`/notes/${noteId}`);

      expect(response.status).toBe(401);
      expect(response.body).toEqual({ error: 'No token provided' });
    });
  });
});
