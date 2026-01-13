import { describe, it, expect, vi, beforeEach } from 'vitest';

// Mock Safeheron service
vi.mock('./safeheron-service', () => ({
  safeheronService: {
    isConfigured: vi.fn().mockReturnValue(false),
    getAssetId: vi.fn((asset) => asset),
    estimateFee: vi.fn().mockResolvedValue(null),
  },
}));

import { FeeCalculationService, feeCalculationService } from './fee-calculation-service';

describe('FeeCalculationService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('calculate', () => {
    it('should calculate fees for BTC withdrawal', async () => {
      const result = await FeeCalculationService.calculate('BTC', '1.0', 'Bitcoin');

      expect(result.asset).toBe('BTC');
      expect(result.amount).toBe('1.0');
      expect(result.chain).toBe('Bitcoin');
      expect(parseFloat(result.fee)).toBeGreaterThan(0);
      expect(parseFloat(result.receivedAmount)).toBeLessThan(parseFloat(result.amount));
      expect(result.feePercentage).toBeGreaterThan(0);
    });

    it('should calculate fees for ETH withdrawal', async () => {
      const result = await FeeCalculationService.calculate('ETH', '10.0', 'Ethereum');

      expect(result.asset).toBe('ETH');
      expect(result.chain).toBe('Ethereum');
      expect(parseFloat(result.fee)).toBeGreaterThan(0);
      expect(parseFloat(result.receivedAmount)).toBeLessThan(parseFloat(result.amount));
    });

    it('should calculate fees for USDC on different chains', async () => {
      const ethResult = await FeeCalculationService.calculate('USDC', '1000', 'Ethereum');
      const arbResult = await FeeCalculationService.calculate('USDC', '1000', 'Arbitrum');

      expect(ethResult.chain).toBe('Ethereum');
      expect(arbResult.chain).toBe('Arbitrum');
      // Both should return valid fees
      expect(parseFloat(ethResult.fee)).toBeGreaterThan(0);
      expect(parseFloat(arbResult.fee)).toBeGreaterThan(0);
    });

    it('should calculate fees for USDT on Tron', async () => {
      const result = await FeeCalculationService.calculate('USDT', '500', 'Tron');

      expect(result.chain).toBe('Tron');
      expect(parseFloat(result.fee)).toBe(1);
    });

    it('should use default chain when chain is not provided', async () => {
      const result = await FeeCalculationService.calculate('BTC', '1.0');

      expect(result.chain).toBe('Bitcoin');
    });

    it('should calculate received amount correctly', async () => {
      const result = await FeeCalculationService.calculate('BTC', '1.0');

      const expectedReceived = parseFloat(result.amount) - parseFloat(result.fee);
      expect(parseFloat(result.receivedAmount)).toBeCloseTo(expectedReceived, 8);
    });
  });

  describe('getDefaultChain', () => {
    it('should return Bitcoin for BTC', () => {
      expect(FeeCalculationService.getDefaultChain('BTC')).toBe('Bitcoin');
    });

    it('should return Ethereum for ETH', () => {
      expect(FeeCalculationService.getDefaultChain('ETH')).toBe('Ethereum');
    });

    it('should return Ethereum for USDC', () => {
      expect(FeeCalculationService.getDefaultChain('USDC')).toBe('Ethereum');
    });

    it('should return Ethereum for USDT', () => {
      expect(FeeCalculationService.getDefaultChain('USDT')).toBe('Ethereum');
    });

    it('should return Ethereum for unknown assets', () => {
      expect(FeeCalculationService.getDefaultChain('UNKNOWN')).toBe('Ethereum');
    });
  });

  describe('validateAmount', () => {
    it('should return valid for sufficient balance', async () => {
      const result = await FeeCalculationService.validateAmount('BTC', '1.0', '10.0');

      expect(result.valid).toBe(true);
    });

    it('should return invalid for insufficient balance', async () => {
      const result = await FeeCalculationService.validateAmount('BTC', '10.0', '1.0');

      expect(result.valid).toBe(false);
      expect(result.error).toContain('Insufficient balance');
      expect(result.maxAmount).toBeDefined();
    });

    it('should return invalid for invalid amount', async () => {
      const result = await FeeCalculationService.validateAmount('BTC', 'invalid', '10.0');

      expect(result.valid).toBe(false);
      expect(result.error).toBe('Invalid amount');
    });

    it('should return invalid for negative amount', async () => {
      const result = await FeeCalculationService.validateAmount('BTC', '-1.0', '10.0');

      expect(result.valid).toBe(false);
      expect(result.error).toBe('Invalid amount');
    });

    it('should return invalid for amount below minimum', async () => {
      const result = await FeeCalculationService.validateAmount('BTC', '0.0001', '10.0');

      expect(result.valid).toBe(false);
      expect(result.error).toContain('Minimum withdrawal amount');
    });

    it('should use different minimum amounts per asset', async () => {
      const btcResult = await FeeCalculationService.validateAmount('BTC', '0.001', '10');
      const ethResult = await FeeCalculationService.validateAmount('ETH', '0.01', '10');
      const usdcResult = await FeeCalculationService.validateAmount('USDC', '1', '10');

      expect(btcResult.valid).toBe(true);
      expect(ethResult.valid).toBe(true);
      expect(usdcResult.valid).toBe(true);
    });
  });

  describe('calculateFeeOnly', () => {
    it('should return fee only without received amount', async () => {
      const result = await FeeCalculationService.calculateFeeOnly('ETH', '10.0');

      expect(result.fee).toBeDefined();
      expect(result.feePercentage).toBeGreaterThan(0);
    });
  });
});

describe('FeeCalculationService Schema Validation', () => {
  it('should throw error for invalid asset', async () => {
    await expect(
      FeeCalculationService.calculate('INVALID', '1.0', 'Bitcoin')
    ).rejects.toThrow();
  });

  it('should throw error for invalid amount', async () => {
    await expect(
      FeeCalculationService.calculate('BTC', 'invalid', 'Bitcoin')
    ).rejects.toThrow();
  });

  it('should throw error for zero amount', async () => {
    await expect(
      FeeCalculationService.calculate('BTC', '0', 'Bitcoin')
    ).rejects.toThrow();
  });

  it('should throw error for negative amount', async () => {
    await expect(
      FeeCalculationService.calculate('BTC', '-1.0', 'Bitcoin')
    ).rejects.toThrow();
  });
});
