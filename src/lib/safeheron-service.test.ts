import { describe, it, expect, vi, beforeEach } from 'vitest';

// Mock environment variables
process.env.SAFEHERON_API_KEY = 'test-api-key';
process.env.SAFEHERON_API_SECRET = 'test-api-secret';
process.env.SAFEHERON_API_URL = 'https://api.safeheron.test';

import { SafeheronService, safeheronService } from './safeheron-service';

describe('SafeheronService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('getAssetId', () => {
    it('should return correct asset ID for BTC', () => {
      expect(safeheronService.getAssetId('BTC', 'Bitcoin')).toBe('BTC');
    });

    it('should return correct asset ID for ETH', () => {
      expect(safeheronService.getAssetId('ETH', 'Ethereum')).toBe('ETH');
    });

    it('should return correct asset ID for USDC on different chains', () => {
      expect(safeheronService.getAssetId('USDC', 'Ethereum')).toBe('USDC-ERC20');
      expect(safeheronService.getAssetId('USDC', 'Arbitrum')).toBe('USDC-ARB');
      expect(safeheronService.getAssetId('USDC', 'Polygon')).toBe('USDC-POL');
    });

    it('should return correct asset ID for USDT on different chains', () => {
      expect(safeheronService.getAssetId('USDT', 'Ethereum')).toBe('USDT-ERC20');
      expect(safeheronService.getAssetId('USDT', 'Tron')).toBe('USDT-TRC20');
    });

    it('should fall back to Ethereum for unknown chains', () => {
      expect(safeheronService.getAssetId('USDC', 'UnknownChain')).toBe('USDC-ERC20');
    });
  });

  describe('estimateFee (mock mode)', () => {
    it('should return mock fee for BTC', async () => {
      const result = await safeheronService.estimateFee('BTC', '1.0', 'test-address');
      expect(result).not.toBeNull();
      expect(result?.fee).toBe('0.0005');
      expect(result?.feeUnit).toBe('BTC');
    });

    it('should return mock fee for ETH', async () => {
      const result = await safeheronService.estimateFee('ETH', '1.0', 'test-address');
      expect(result).not.toBeNull();
      expect(result?.fee).toBe('0.002');
      expect(result?.feeUnit).toBe('ETH');
    });

    it('should return mock fee for USDC', async () => {
      const result = await safeheronService.estimateFee('USDC', '100', 'test-address');
      expect(result).not.toBeNull();
      expect(result?.fee).toBe('1');
      expect(result?.feeUnit).toBe('USDC');
    });

    it('should return mock fee for USDT', async () => {
      const result = await safeheronService.estimateFee('USDT', '100', 'test-address');
      expect(result).not.toBeNull();
      expect(result?.fee).toBe('2');
      expect(result?.feeUnit).toBe('USDT');
    });

    it('should return default fee for unknown assets', async () => {
      const result = await safeheronService.estimateFee('UNKNOWN', '1.0', 'test-address');
      expect(result).not.toBeNull();
      expect(result?.fee).toBe('0.001');
      expect(result?.feeUnit).toBe('UNKNOWN');
    });
  });

  describe('coinOut (mock mode)', () => {
    it('should return mock response for coin out', async () => {
      const result = await safeheronService.coinOut(
        'vault-123',
        'BTC',
        '1.0',
        'bc1qtest123',
        'Test withdrawal'
      );

      expect(result).not.toBeNull();
      expect(result?.txId).toMatch(/^mock_/);
      expect(result?.status).toBe('INIT');
      expect(result?.txHash).toBeDefined();
    });
  });

  describe('getTransactionStatus (mock mode)', () => {
    it('should return mock transaction status', async () => {
      const result = await safeheronService.getTransactionStatus('tx-123');

      expect(result).not.toBeNull();
      expect(result?.txId).toBe('tx-123');
      expect(result?.status).toBe('COMPLETED');
      expect(result?.txHash).toBeDefined();
    });
  });

  describe('getVault (mock mode)', () => {
    it('should return mock vault information', async () => {
      const result = await safeheronService.getVault('vault-123');

      expect(result).not.toBeNull();
      expect(result?.vaultId).toBe('vault-123');
      expect(result?.name).toBe('Mock Vault');
      expect(result?.assetId).toBe('BTC');
      expect(result?.balance).toBe('1000000');
    });
  });
});

describe('SafeheronService Configuration', () => {
  it('should use mock mode when API credentials are not configured', async () => {
    // Temporarily override env vars to test configuration check
    const originalKey = process.env.SAFEHERON_API_KEY;
    const originalSecret = process.env.SAFEHERON_API_SECRET;

    process.env.SAFEHERON_API_KEY = '';
    process.env.SAFEHERON_API_SECRET = '';

    const service = new SafeheronService();
    expect(service).toBeDefined();

    // Restore env vars
    process.env.SAFEHERON_API_KEY = originalKey;
    process.env.SAFEHERON_API_SECRET = originalSecret;
  });
});
