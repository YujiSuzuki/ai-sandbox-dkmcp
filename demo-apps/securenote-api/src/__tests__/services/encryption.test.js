const EncryptionService = require('../../services/encryption');

describe('EncryptionService', () => {
  let encryptionService;
  const testKey = 'my-secret-encryption-key-32-bytes';

  beforeEach(() => {
    encryptionService = new EncryptionService(testKey);
  });

  describe('encrypt and decrypt', () => {
    it('should encrypt and decrypt text correctly', () => {
      const plaintext = 'Hello, World!';
      const encrypted = encryptionService.encrypt(plaintext);
      const decrypted = encryptionService.decrypt(encrypted);

      expect(decrypted).toBe(plaintext);
    });

    it('should produce different ciphertexts for same plaintext (due to random IV)', () => {
      const plaintext = 'Test message';
      const encrypted1 = encryptionService.encrypt(plaintext);
      const encrypted2 = encryptionService.encrypt(plaintext);

      expect(encrypted1).not.toBe(encrypted2);
    });

    it('should handle special characters', () => {
      const plaintext = 'Special chars: !@#$%^&*()_+-=[]{}|;:",.<>?/~`';
      const encrypted = encryptionService.encrypt(plaintext);
      const decrypted = encryptionService.decrypt(encrypted);

      expect(decrypted).toBe(plaintext);
    });

    it('should handle unicode characters', () => {
      const plaintext = 'Unicode: 你好世界 🚀 こんにちは';
      const encrypted = encryptionService.encrypt(plaintext);
      const decrypted = encryptionService.decrypt(encrypted);

      expect(decrypted).toBe(plaintext);
    });

    it('should handle empty string', () => {
      const plaintext = '';
      const encrypted = encryptionService.encrypt(plaintext);
      const decrypted = encryptionService.decrypt(encrypted);

      expect(decrypted).toBe(plaintext);
    });

    it('should handle long text', () => {
      const plaintext = 'A'.repeat(10000);
      const encrypted = encryptionService.encrypt(plaintext);
      const decrypted = encryptionService.decrypt(encrypted);

      expect(decrypted).toBe(plaintext);
    });
  });

  describe('hash', () => {
    it('should produce consistent hash for same input', () => {
      const text = 'Test input';
      const hash1 = encryptionService.hash(text);
      const hash2 = encryptionService.hash(text);

      expect(hash1).toBe(hash2);
    });

    it('should produce different hash for different inputs', () => {
      const hash1 = encryptionService.hash('Input 1');
      const hash2 = encryptionService.hash('Input 2');

      expect(hash1).not.toBe(hash2);
    });

    it('should produce 64-character hex string (SHA256)', () => {
      const hash = encryptionService.hash('Test');
      expect(hash).toMatch(/^[a-f0-9]{64}$/);
    });
  });
});
