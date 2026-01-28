const jwt = require('jsonwebtoken');
const authMiddleware = require('../../middleware/auth');

describe('authMiddleware', () => {
  const jwtSecret = 'test-secret-key';
  const testPayload = { id: 1, username: 'testuser' };
  let req, res, next;

  beforeEach(() => {
    req = {
      headers: {},
      app: {
        locals: {
          jwtSecret: jwtSecret
        }
      }
    };
    res = {
      status: jest.fn().mockReturnThis(),
      json: jest.fn()
    };
    next = jest.fn();
  });

  it('should call next() when valid token is provided', () => {
    const token = jwt.sign(testPayload, jwtSecret);
    req.headers.authorization = `Bearer ${token}`;

    authMiddleware(req, res, next);

    expect(next).toHaveBeenCalled();
    expect(res.status).not.toHaveBeenCalled();
    expect(req.user.id).toBe(1);
    expect(req.user.username).toBe('testuser');
  });

  it('should return 401 when no authorization header is provided', () => {
    authMiddleware(req, res, next);

    expect(res.status).toHaveBeenCalledWith(401);
    expect(res.json).toHaveBeenCalledWith({ error: 'No token provided' });
    expect(next).not.toHaveBeenCalled();
  });

  it('should return 401 when authorization header does not start with Bearer', () => {
    req.headers.authorization = 'Basic some-token';

    authMiddleware(req, res, next);

    expect(res.status).toHaveBeenCalledWith(401);
    expect(res.json).toHaveBeenCalledWith({ error: 'No token provided' });
    expect(next).not.toHaveBeenCalled();
  });

  it('should return 401 when token is invalid', () => {
    req.headers.authorization = 'Bearer invalid-token';

    authMiddleware(req, res, next);

    expect(res.status).toHaveBeenCalledWith(401);
    expect(res.json).toHaveBeenCalledWith({ error: 'Invalid token' });
    expect(next).not.toHaveBeenCalled();
  });

  it('should return 500 when jwt secret is not configured', () => {
    const validToken = jwt.sign(testPayload, jwtSecret);
    req.headers.authorization = `Bearer ${validToken}`;
    req.app.locals.jwtSecret = null;

    authMiddleware(req, res, next);

    expect(res.status).toHaveBeenCalledWith(500);
    expect(res.json).toHaveBeenCalledWith({ error: 'Server configuration error' });
    expect(next).not.toHaveBeenCalled();
  });

  it('should return 401 when token is signed with different secret', () => {
    const token = jwt.sign(testPayload, 'different-secret');
    req.headers.authorization = `Bearer ${token}`;

    authMiddleware(req, res, next);

    expect(res.status).toHaveBeenCalledWith(401);
    expect(res.json).toHaveBeenCalledWith({ error: 'Invalid token' });
    expect(next).not.toHaveBeenCalled();
  });

  it('should handle expired tokens', () => {
    const expiredToken = jwt.sign(testPayload, jwtSecret, { expiresIn: '-1h' });
    req.headers.authorization = `Bearer ${expiredToken}`;

    authMiddleware(req, res, next);

    expect(res.status).toHaveBeenCalledWith(401);
    expect(res.json).toHaveBeenCalledWith({ error: 'Invalid token' });
    expect(next).not.toHaveBeenCalled();
  });
});
