import { describe, it, expect, vi, beforeEach } from "vitest";
import { LendingService } from "./lending-service.js";

describe("LendingService", () => {
  describe("calculateAPY", () => {
    it("should return correct base APY for USDT", () => {
      const apy = LendingService.calculateAPY("USDT", 30);
      expect(apy).toBe(10.17);
    });

    it("should apply multiplier for long terms", () => {
      const shortApy = LendingService.calculateAPY("BTC", 30);
      const longApy = LendingService.calculateAPY("BTC", 360);
      expect(longApy).toBeGreaterThan(shortApy);
      expect(longApy).toBe(6.5);
    });

    it("should return default rate for unknown assets", () => {
      const apy = LendingService.calculateAPY("UNKNOWN", 30);
      expect(apy).toBe(5.17);
    });
  });

  describe("calculateEstimatedYield", () => {
    it("should calculate correct yield for 1 year", () => {
      // 10000 * 10% * 365 / 365 = 1000
      const estimatedYield = LendingService.calculateEstimatedYield("10000", 10, 365);
      expect(estimatedYield).toBe("1000.00000000");
    });

    it("should calculate correct yield for 6 months", () => {
      const estimatedYield = LendingService.calculateEstimatedYield("10000", 10, 182.5);
      expect(estimatedYield).toBe("500.00000000");
    });
  });
});
