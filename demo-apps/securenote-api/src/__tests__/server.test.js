const express = require('express');
const cors = require('cors');
const bodyParser = require('body-parser');
const request = require('supertest');
const authRoutes = require('../routes/auth');
const notesRoutes = require('../routes/notes');
const demoRoutes = require('../routes/demo');

describe('Server', () => {
  let app;
  const jwtSecret = 'test-secret-key';
  const encryptionKey = 'test-encryption-key-32-bytes';

  beforeEach(() => {
    app = express();
    app.use(cors());
    app.use(bodyParser.json());
    app.locals.jwtSecret = jwtSecret;
    app.locals.encryptionKey = encryptionKey;
    app.locals.secretsLoaded = true;

    app.use('/api/auth', authRoutes);
    app.use('/api/notes', notesRoutes);
    app.use('/api/demo', demoRoutes);

    app.get('/api/health', (req, res) => {
      res.json({
        status: 'ok',
        timestamp: new Date().toISOString(),
        secretsLoaded: app.locals.secretsLoaded
      });
    });

    app.get('/', (req, res) => {
      res.json({
        name: 'SecureNote API',
        version: '1.0.0',
        description: 'Demo API for DockMCP - Secure Docker access for AI assistants',
        endpoints: {
          health: '/api/health',
          auth: '/api/auth/login',
          notes: '/api/notes',
          demo: '/api/demo/secrets-status'
        }
      });
    });
  });

  describe('GET /', () => {
    it('should return API info', async () => {
      const response = await request(app).get('/');

      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('name');
      expect(response.body).toHaveProperty('version');
      expect(response.body).toHaveProperty('description');
      expect(response.body).toHaveProperty('endpoints');
      expect(response.body.name).toBe('SecureNote API');
    });

    it('should have correct endpoints listed', async () => {
      const response = await request(app).get('/');

      expect(response.body.endpoints).toHaveProperty('health');
      expect(response.body.endpoints).toHaveProperty('auth');
      expect(response.body.endpoints).toHaveProperty('notes');
      expect(response.body.endpoints).toHaveProperty('demo');
    });
  });

  describe('GET /api/health', () => {
    it('should return health status', async () => {
      const response = await request(app).get('/api/health');

      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('status');
      expect(response.body).toHaveProperty('timestamp');
      expect(response.body).toHaveProperty('secretsLoaded');
      expect(response.body.status).toBe('ok');
    });

    it('should return ISO timestamp', async () => {
      const response = await request(app).get('/api/health');

      expect(response.body.timestamp).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/);
    });

    it('should indicate secrets status', async () => {
      const response = await request(app).get('/api/health');

      expect(typeof response.body.secretsLoaded).toBe('boolean');
    });
  });

  describe('Error handling', () => {
    it('should return 404 for non-existent routes', async () => {
      const response = await request(app).get('/non-existent-route');

      expect(response.status).toBe(404);
    });

    it('should handle CORS', async () => {
      const response = await request(app)
        .get('/api/health')
        .set('Origin', 'http://localhost:3000');

      expect(response.headers['access-control-allow-origin']).toBeDefined();
    });
  });
});
