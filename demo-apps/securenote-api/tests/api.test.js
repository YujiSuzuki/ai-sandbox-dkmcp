const request = require('supertest');
const app = require('../src/server');

describe('SecureNote API Tests', () => {
  let authToken;

  // Test health endpoint
  test('GET /api/health should return ok', async () => {
    const response = await request(app).get('/api/health');
    expect(response.status).toBe(200);
    expect(response.body.status).toBe('ok');
  });

  // Test login
  test('POST /api/auth/login with valid credentials', async () => {
    const response = await request(app)
      .post('/api/auth/login')
      .send({ username: 'demo', password: 'demo123' });

    expect(response.status).toBe(200);
    expect(response.body).toHaveProperty('token');
    expect(response.body.user.username).toBe('demo');

    authToken = response.body.token;
  });

  test('POST /api/auth/login with invalid credentials', async () => {
    const response = await request(app)
      .post('/api/auth/login')
      .send({ username: 'demo', password: 'wrongpassword' });

    expect(response.status).toBe(401);
  });

  // Test notes CRUD
  test('POST /api/notes should create encrypted note', async () => {
    const response = await request(app)
      .post('/api/notes')
      .set('Authorization', `Bearer ${authToken}`)
      .send({
        title: 'Test Note',
        content: 'This is a secret message'
      });

    expect(response.status).toBe(201);
    expect(response.body.title).toBe('Test Note');
    expect(response.body.content).toBe('This is a secret message');
  });

  test('GET /api/notes should return decrypted notes', async () => {
    const response = await request(app)
      .get('/api/notes')
      .set('Authorization', `Bearer ${authToken}`);

    expect(response.status).toBe(200);
    expect(Array.isArray(response.body)).toBe(true);
  });

  // Test demo endpoint
  test('GET /api/demo/secrets-status should show secrets are loaded', async () => {
    const response = await request(app).get('/api/demo/secrets-status');

    expect(response.status).toBe(200);
    expect(response.body.secretsLoaded).toBe(true);
    expect(response.body.proof.jwtSecretLoaded).toBe(true);
  });
});
