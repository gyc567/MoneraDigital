import { describe, it, expect, vi, beforeEach } from 'vitest';

describe('WalletService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('createWallet validation', () => {
    it('should return validation error for invalid request_id format', async () => {
      // Import here to avoid module resolution issues
      const { WalletService } = await import('./wallet-service');
      const result = await WalletService.createWallet(1, 'invalid-uuid');
      expect(result.success).toBe(false);
      expect(result.message).toContain('Invalid uuid');
      expect(result.status).toBe('failed');
    });

    it('should return validation error for negative user_id', async () => {
      const { WalletService } = await import('./wallet-service');
      const validId = '550e8400-e29b-41d4-a716-446655440000';
      const result = await WalletService.createWallet(-1, validId);
      expect(result.success).toBe(false);
      expect(result.message).toContain('greater than 0');
      expect(result.status).toBe('failed');
    });
  });
});
