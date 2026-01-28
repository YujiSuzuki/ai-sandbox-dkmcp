const request = require('supertest');
const express = require('express');
const bodyParser = require('body-parser');
const jwt = require('jsonwebtoken');
const authRoutes = require('../../routes/auth');

describe('Auth Routes', () => {
  let app;
  const jwtSecret = 'test-secret-key';

  beforeEach(() => {
    app = express();
    app.use(bodyParser.json());
    app.locals.jwtSecret = jwtSecret;
    app.locals.secretsLoaded = true;
    app.use('/auth', authRoutes);
  });

  describe('POST /login', () => {
    it('should return token for valid credentials', async () => {
      const response = await request(app)
        .post('/auth/login')
        .send({
          username: 'demo',
          password: 'demo123'
        });

      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('token');
      expect(response.body).toHaveProperty('user');
      expect(response.body.user.username).toBe('demo');
      expect(response.body.user.id).toBe(1);

      // Verify token is valid
      const decoded = jwt.verify(response.body.token, jwtSecret);
      expect(decoded.username).toBe('demo');
    });

    it('should return 401 for invalid credentials', async () => {
      const response = await request(app)
        .post('/auth/login')
        .send({
          username: 'demo',
          password: 'wrongpassword'
        });

      expect(response.status).toBe(401);
      expect(response.body).toEqual({ error: 'Invalid credentials' });
    });

    it('should return 401 for nonexistent user', async () => {
      const response = await request(app)
        .post('/auth/login')
        .send({
          username: 'nonexistent',
          password: 'anypassword'
        });

      expect(response.status).toBe(401);
      expect(response.body).toEqual({ error: 'Invalid credentials' });
    });

    it('should return 400 when username is missing', async () => {
      const response = await request(app)
        .post('/auth/login')
        .send({
          password: 'demo123'
        });

      expect(response.status).toBe(400);
      expect(response.body).toEqual({ error: 'Username and password required' });
    });

    it('should return 400 when password is missing', async () => {
      const response = await request(app)
        .post('/auth/login')
        .send({
          username: 'demo'
        });

      expect(response.status).toBe(400);
      expect(response.body).toEqual({ error: 'Username and password required' });
    });

    it('should return 400 when both username and password are missing', async () => {
      const response = await request(app)
        .post('/auth/login')
        .send({});

      expect(response.status).toBe(400);
      expect(response.body).toEqual({ error: 'Username and password required' });
    });

    it('should return 500 when secrets are not loaded', async () => {
      app.locals.secretsLoaded = false;

      const response = await request(app)
        .post('/auth/login')
        .send({
          username: 'demo',
          password: 'demo123'
        });

      expect(response.status).toBe(500);
      expect(response.body).toHaveProperty('error');
      expect(response.body).toHaveProperty('message');
    });

    it('should work with alice user', async () => {
      const response = await request(app)
        .post('/auth/login')
        .send({
          username: 'alice',
          password: 'alice123'
        });

      expect(response.status).toBe(200);
      expect(response.body.user.username).toBe('alice');
      expect(response.body.user.id).toBe(2);
    });

    it('should generate tokens with 24h expiration', async () => {
      const response = await request(app)
        .post('/auth/login')
        .send({
          username: 'demo',
          password: 'demo123'
        });

      const decoded = jwt.verify(response.body.token, jwtSecret);
      expect(decoded).toHaveProperty('exp');
      // Token expiration should be roughly 24 hours from now
      const expiresIn = decoded.exp - Math.floor(Date.now() / 1000);
      expect(expiresIn).toBeGreaterThan(86000); // ~24 hours
      expect(expiresIn).toBeLessThanOrEqual(86400); // Less than or equal to 24 hours
    });
  });
});
