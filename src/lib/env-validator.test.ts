import { describe, it, expect } from 'vitest';
import { validateEnv, env, VITE_API_BASE_URL } from './env-validator';

// We'll test with actual environment variables that exist
// Since Vite environment variables are build-time constants, we'll test with real values

describe('env-validator', () => {
  describe('validateEnv', () => {
    it('should return true when environment is configured', () => {
      const result = validateEnv();
      // In test environment, VITE_API_BASE_URL should be set from .env
      expect(result).toBe(true);
    });
  });

  describe('env', () => {
    it('should parse VITE_API_BASE_URL correctly', () => {
      expect(env.VITE_API_BASE_URL).toBeDefined();
      expect(typeof env.VITE_API_BASE_URL).toBe('string');
    });

    it('should have a valid URL format', () => {
      expect(env.VITE_API_BASE_URL).toMatch(/^https?:\/\/.+/);
    });

    it('should point to the correct backend URL', () => {
      expect(env.VITE_API_BASE_URL).toBe('https://monera-digital--gyc567.replit.app');
    });
  });

  describe('VITE_API_BASE_URL export', () => {
    it('should export VITE_API_BASE_URL as a string', () => {
      expect(typeof VITE_API_BASE_URL).toBe('string');
    });

    it('should match URL format', () => {
      expect(VITE_API_BASE_URL).toMatch(/^https?:\/\/.+/);
    });

    it('should equal the backend URL from env object', () => {
      expect(VITE_API_BASE_URL).toBe(env.VITE_API_BASE_URL);
    });
  });
});
