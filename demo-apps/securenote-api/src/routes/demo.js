const express = require('express');
const router = express.Router();

// GET /api/demo/secrets-status
// This endpoint demonstrates that the API has access to secrets
// but AI assistants in DevContainer cannot read them directly
router.get('/secrets-status', (req, res) => {
  const secretsLoaded = req.app.locals.secretsLoaded;
  const jwtSecret = req.app.locals.jwtSecret;
  const encryptionKey = req.app.locals.encryptionKey;

  if (!secretsLoaded) {
    return res.json({
      message: 'Secrets are NOT loaded',
      secretsLoaded: false,
      explanation: 'This API cannot function without secrets. Check if secrets/ directory is mounted correctly.'
    });
  }

  res.json({
    message: 'This API has access to secrets',
    secretsLoaded: true,
    proof: {
      jwtSecretLoaded: !!jwtSecret,
      jwtSecretPreview: jwtSecret ? `${jwtSecret.substring(0, 10)}***` : null,
      jwtSecretLength: jwtSecret ? jwtSecret.length : 0,

      encryptionKeyLoaded: !!encryptionKey,
      encryptionKeyPreview: encryptionKey ? `${encryptionKey.substring(0, 10)}***` : null,
      encryptionKeyLength: encryptionKey ? encryptionKey.length : 0
    },
    explanation: {
      container: 'This API container CAN read secrets from /app/secrets/',
      devContainer: 'AI assistants in DevContainer CANNOT read these secrets (directory is hidden)',
      security: 'This demonstrates secure development with AI - secrets are isolated but functionality is preserved'
    }
  });
});

module.exports = router;
