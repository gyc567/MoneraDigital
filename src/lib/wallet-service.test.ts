import { describe, it, expect, vi, beforeEach } from 'vitest';

describe('WalletService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('createWallet validation', () => {
    it('should throw ZodError for invalid request_id format', async () => {
      // Import here to avoid module resolution issues
      const { WalletService } = await import('./wallet-service');
      await expect(WalletService.createWallet(1, 'invalid-uuid'))
        .rejects.toThrow();
    });

    it('should throw ZodError for negative user_id', async () => {
      const { WalletService } = await import('./wallet-service');
      const validId = '550e8400-e29b-41d4-a716-446655440000';
      await expect(WalletService.createWallet(-1, validId))
        .rejects.toThrow();
    });
  });
});
