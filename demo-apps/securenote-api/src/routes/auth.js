const express = require('express');
const jwt = require('jsonwebtoken');
const EncryptionService = require('../services/encryption');

const router = express.Router();

/**
 * DEMO ONLY - Never use hardcoded credentials in production!
 * デモ専用 - 本番環境では絶対にハードコードしないでください！
 *
 * Production requirements / 本番環境での要件:
 * - Store users in database / データベースでユーザー管理
 * - Hash passwords with bcrypt / パスワードは bcrypt 等でハッシュ化
 * - Implement rate limiting / ブルートフォース対策（レート制限）
 */
const users = [
  { id: 1, username: 'demo', password: 'demo123' },
  { id: 2, username: 'alice', password: 'alice123' }
];

// POST /api/auth/login
router.post('/login', (req, res) => {
  const { username, password } = req.body;

  if (!username || !password) {
    return res.status(400).json({ error: 'Username and password required' });
  }

  // Check if secrets are loaded
  if (!req.app.locals.secretsLoaded) {
    return res.status(500).json({
      error: 'Server configuration error',
      message: 'Secrets not loaded. Cannot sign JWT tokens.'
    });
  }

  // Find user
  const user = users.find(u => u.username === username && u.password === password);

  if (!user) {
    return res.status(401).json({ error: 'Invalid credentials' });
  }

  // Generate JWT token
  const token = jwt.sign(
    { id: user.id, username: user.username },
    req.app.locals.jwtSecret,
    { expiresIn: '24h' }
  );

  res.json({
    token,
    user: {
      id: user.id,
      username: user.username
    }
  });
});

module.exports = router;
