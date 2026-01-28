const request = require('supertest');
const express = require('express');
const demoRoutes = require('../../routes/demo');

describe('Demo Routes', () => {
  let app;

  beforeEach(() => {
    app = express();
    app.use('/demo', demoRoutes);
  });

  describe('GET /demo/secrets-status', () => {
    it('should show that secrets are not loaded when not configured', async () => {
      const response = await request(app).get('/demo/secrets-status');

      expect(response.status).toBe(200);
      expect(response.body).toEqual({
        message: 'Secrets are NOT loaded',
        secretsLoaded: false,
        explanation: 'This API cannot function without secrets. Check if secrets/ directory is mounted correctly.'
      });
    });

    it('should show secrets are loaded when configured', async () => {
      app.locals.secretsLoaded = true;
      app.locals.jwtSecret = 'test-jwt-secret-12345';
      app.locals.encryptionKey = 'test-encryption-key-32-bytes';

      const response = await request(app).get('/demo/secrets-status');

      expect(response.status).toBe(200);
      expect(response.body.message).toBe('This API has access to secrets');
      expect(response.body.secretsLoaded).toBe(true);
      expect(response.body.proof.jwtSecretLoaded).toBe(true);
      expect(response.body.proof.encryptionKeyLoaded).toBe(true);
      expect(response.body.proof.jwtSecretLength).toBe(21);
      expect(response.body.proof.encryptionKeyLength).toBe(28);
      expect(response.body.proof.jwtSecretPreview).toBe('test-jwt-s***');
      expect(response.body.explanation).toBeDefined();
    });

    it('should handle missing jwtSecret', async () => {
      app.locals.secretsLoaded = true;
      app.locals.jwtSecret = null;
      app.locals.encryptionKey = 'test-encryption-key';

      const response = await request(app).get('/demo/secrets-status');

      expect(response.status).toBe(200);
      expect(response.body.proof.jwtSecretLoaded).toBe(false);
      expect(response.body.proof.jwtSecretLength).toBe(0);
      expect(response.body.proof.jwtSecretPreview).toBeNull();
    });

    it('should handle missing encryptionKey', async () => {
      app.locals.secretsLoaded = true;
      app.locals.jwtSecret = 'test-secret';
      app.locals.encryptionKey = null;

      const response = await request(app).get('/demo/secrets-status');

      expect(response.status).toBe(200);
      expect(response.body.proof.encryptionKeyLoaded).toBe(false);
      expect(response.body.proof.encryptionKeyLength).toBe(0);
      expect(response.body.proof.encryptionKeyPreview).toBeNull();
    });

    it('should show preview of secrets without exposing full values', async () => {
      app.locals.secretsLoaded = true;
      app.locals.jwtSecret = 'this-is-a-very-long-secret-key-with-more-characters';
      app.locals.encryptionKey = 'encryption-key-also-very-long';

      const response = await request(app).get('/demo/secrets-status');

      expect(response.status).toBe(200);
      expect(response.body.proof.jwtSecretPreview).toMatch(/^this-is-a-\*\*\*$/);
      expect(response.body.proof.encryptionKeyPreview).toMatch(/^encryption\*\*\*$/);
      expect(response.body.proof.jwtSecretPreview).not.toContain('very-long-secret');
    });
  });
});
