import { describe, it, expect, vi, beforeEach } from 'vitest';
import { LendingService } from './lending-service.js';

describe('LendingService', () => {
  describe('calculateAPY', () => {
    it('should return correct base APY for USDT', () => {
      const apy = LendingService.calculateAPY('USDT', 30);
      expect(apy).toBe(8.5);
    });

    it('should apply multiplier for long terms', () => {
      const shortApy = LendingService.calculateAPY('BTC', 30);
      const longApy = LendingService.calculateAPY('BTC', 360);
      expect(longApy).toBeGreaterThan(shortApy);
      expect(longApy).toBe(4.5 * 1.5);
    });

    it('should return default rate for unknown assets', () => {
      const apy = LendingService.calculateAPY('UNKNOWN', 30);
      expect(apy).toBe(5.0);
    });
  });

  describe('calculateEstimatedYield', () => {
    it('should calculate correct yield for 1 year', () => {
      // 10000 * 10% * 365 / 365 = 1000
      const yield_ = LendingService.calculateEstimatedYield(10000, 10, 365);
      expect(yield_).toBe(1000);
    });

    it('should calculate correct yield for 6 months', () => {
      const yield_ = LendingService.calculateEstimatedYield(10000, 10, 182.5);
      expect(yield_).toBe(500);
    });
  });
});
