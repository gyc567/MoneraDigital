import { describe, it, expect, vi, beforeEach } from 'vitest';

// Mock localStorage
const localStorageMock = {
  getItem: vi.fn(),
  setItem: vi.fn(),
  clear: vi.fn(),
  removeItem: vi.fn(),
};
Object.defineProperty(global, 'localStorage', {
  value: localStorageMock,
  writable: true,
});

describe('WalletService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorageMock.getItem.mockReturnValue('mock-token');
  });

  describe('createWallet validation', () => {
    it('should throw validation error for negative user_id', async () => {
      const { WalletService } = await import('./wallet-service');
      const validProductCode = 'X_FINANCE';
      await expect(WalletService.createWallet(-1, validProductCode, 'USDT_TRC20'))
        .rejects.toThrow('greater than 0');
    });

    it('should throw validation error for invalid currency format', async () => {
      const { WalletService } = await import('./wallet-service');
      const validProductCode = 'X_FINANCE';
      await expect(WalletService.createWallet(1, validProductCode, 'INVALID_CURRENCY'))
        .rejects.toThrow('Currency must be one of');
    });

    it('should throw validation error for empty product code', async () => {
      const { WalletService } = await import('./wallet-service');
      await expect(WalletService.createWallet(1, '', 'USDT_TRC20'))
        .rejects.toThrow('String must contain at least 1 character');
    });

    it('should throw validation error for empty currency', async () => {
      const { WalletService } = await import('./wallet-service');
      const validProductCode = 'X_FINANCE';
      await expect(WalletService.createWallet(1, validProductCode, ''))
        .rejects.toThrow('String must contain at least 1 character');
    });

    it('should accept valid currency format (USDT_TRC20)', async () => {
      const { WalletService } = await import('./wallet-service');
      const validProductCode = 'X_FINANCE';
      try {
        await WalletService.createWallet(1, validProductCode, 'USDT_TRC20');
      } catch (e: any) {
        expect(e.message).not.toContain('Currency must be one of');
      }
    });

    it('should accept all valid currency formats', async () => {
      const { WalletService } = await import('./wallet-service');
      const validProductCode = 'X_FINANCE';
      const validCurrencies = [
        'USDT_ERC20',
        'USDT_TRC20',
        'USDT_BEP20_BINANCE_SMART_CHAIN_MAINNET',
        'USDC_ERC20',
        'USDC_TRC20',
        'USDC_BEP20_BINANCE_SMART_CHAIN_MAINNET'
      ];

      for (const currency of validCurrencies) {
        try {
          await WalletService.createWallet(1, validProductCode, currency);
        } catch (e: any) {
          expect(e.message).not.toContain('Currency must be one of');
        }
      }
    });

    it('should accept various valid product codes', async () => {
      const { WalletService } = await import('./wallet-service');
      const productCodes = ['X_FINANCE', 'DEFAULT', 'TEST', 'prod-123'];

      for (const productCode of productCodes) {
        try {
          await WalletService.createWallet(1, productCode, 'USDT_TRC20');
        } catch (e: any) {
          expect(e.message).not.toContain('String must contain');
        }
      }
    });
  });
});
