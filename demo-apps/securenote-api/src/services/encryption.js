const crypto = require('crypto');

class EncryptionService {
  constructor(encryptionKey) {
    // Ensure key is 32 bytes for AES-256
    this.key = Buffer.from(encryptionKey.padEnd(32, '0').substring(0, 32));
    this.algorithm = 'aes-256-cbc';
  }

  encrypt(text) {
    const iv = crypto.randomBytes(16);
    const cipher = crypto.createCipheriv(this.algorithm, this.key, iv);

    let encrypted = cipher.update(text, 'utf8', 'hex');
    encrypted += cipher.final('hex');

    // Return IV + encrypted data
    return iv.toString('hex') + ':' + encrypted;
  }

  decrypt(encryptedText) {
    const parts = encryptedText.split(':');
    const iv = Buffer.from(parts[0], 'hex');
    const encrypted = parts[1];

    const decipher = crypto.createDecipheriv(this.algorithm, this.key, iv);

    let decrypted = decipher.update(encrypted, 'hex', 'utf8');
    decrypted += decipher.final('utf8');

    return decrypted;
  }

  hash(text) {
    return crypto.createHash('sha256').update(text).digest('hex');
  }
}

module.exports = EncryptionService;
