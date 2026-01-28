const express = require('express');
const cors = require('cors');
const bodyParser = require('body-parser');
const fs = require('fs');
const path = require('path');
require('dotenv').config();

const authRoutes = require('./routes/auth');
const notesRoutes = require('./routes/notes');
const demoRoutes = require('./routes/demo');

const app = express();
const PORT = process.env.PORT || 8080;

// Middleware
app.use(cors());
app.use(bodyParser.json());

// Load secrets at startup
const loadSecrets = () => {
  const jwtSecretPath = process.env.JWT_SECRET_PATH || path.join(__dirname, '../secrets/jwt-secret.key');
  const encryptionKeyPath = process.env.ENCRYPTION_KEY_PATH || path.join(__dirname, '../secrets/encryption.key');

  try {
    const jwtSecret = fs.readFileSync(jwtSecretPath, 'utf8').trim();
    const encryptionKey = fs.readFileSync(encryptionKeyPath, 'utf8').trim();

    if (!jwtSecret || !encryptionKey) {
      throw new Error('Secrets are empty');
    }

    // ============================================================
    // DEMO ONLY: Logging secrets for DockMCP masking demonstration
    // デモ専用: DockMCP のマスキング機能を体験するためのログ出力
    //
    // In production, NEVER log secrets (even partially)!
    // 本番環境では、シークレットをログ出力しないでください（一部でも）！
    //
    // DockMCP masks "JWT Secret:" but check if "Encryption Key:" is also masked.
    // See: dkmcp/configs/dkmcp.example.yaml → output_masking.patterns
    // ============================================================
    console.log('✅ Secrets loaded successfully');
    console.log(`   JWT Secret: ${jwtSecret.substring(0, 10)}... (${jwtSecret.length} chars)`);
    console.log(`   Encryption Key: ${encryptionKey.substring(0, 10)}... (${encryptionKey.length} chars)`);

    // Store in app locals for access by routes
    app.locals.jwtSecret = jwtSecret;
    app.locals.encryptionKey = encryptionKey;
    app.locals.secretsLoaded = true;
  } catch (error) {
    console.error('❌ Failed to load secrets:', error.message);
    console.error('   This API will not function properly without secrets.');
    app.locals.secretsLoaded = false;
  }
};

loadSecrets();

// Routes
app.use('/api/auth', authRoutes);
app.use('/api/notes', notesRoutes);
app.use('/api/demo', demoRoutes);

// Health check
app.get('/api/health', (req, res) => {
  res.json({
    status: 'ok',
    timestamp: new Date().toISOString(),
    secretsLoaded: app.locals.secretsLoaded
  });
});

// Root endpoint
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

// Error handling
app.use((err, req, res, next) => {
  console.error('Error:', err);
  res.status(500).json({
    error: 'Internal server error',
    message: err.message
  });
});

// Start server
app.listen(PORT, '0.0.0.0', () => {
  console.log(`🚀 SecureNote API listening on port ${PORT}`);
  console.log(`   Environment: ${process.env.NODE_ENV}`);
  console.log(`   Health check: http://localhost:${PORT}/api/health`);
});

module.exports = app;
