import { describe, it, expect, beforeEach } from "vitest";
import { 
  getDaysRemaining, 
  formatNumber, 
  calculateInterest, 
  calculateDepositDates, 
  validateAmount,
  generateRequestId,
  isSameBusinessDay,
  type InterestCalculationResult,
  type DateCalculationResult
} from "../lib/wealth-utils";

describe("wealth-utils", () => {
  describe("getDaysRemaining", () => {
    it("returns positive days when end date is in the future", () => {
      const futureDate = new Date();
      futureDate.setDate(futureDate.getDate() + 10);
      const result = getDaysRemaining(futureDate.toISOString());
      expect(result).toBe(10);
    });

    it("returns 0 when end date is in the past", () => {
      const pastDate = new Date();
      pastDate.setDate(pastDate.getDate() - 5);
      const result = getDaysRemaining(pastDate.toISOString());
      expect(result).toBe(0);
    });

    it("returns 0 when end date is today", () => {
      const today = new Date();
      const result = getDaysRemaining(today.toISOString());
      expect(result).toBe(0);
    });

    it("returns 1 when end date is tomorrow", () => {
      const tomorrow = new Date();
      tomorrow.setDate(tomorrow.getDate() + 1);
      const result = getDaysRemaining(tomorrow.toISOString());
      expect(result).toBe(1);
    });

    it("handles large date ranges", () => {
      const futureDate = new Date();
      futureDate.setDate(futureDate.getDate() + 365);
      const result = getDaysRemaining(futureDate.toISOString());
      expect(result).toBe(365);
    });

    it("handles Date objects", () => {
      const futureDate = new Date();
      futureDate.setDate(futureDate.getDate() + 7);
      const result = getDaysRemaining(futureDate);
      expect(result).toBe(7);
    });
  });

  describe("formatNumber", () => {
    it("formats integer numbers correctly without trailing zeros", () => {
      expect(formatNumber(1000)).toBe("1,000");
    });

    it("formats decimal numbers correctly up to 7 decimal places", () => {
      expect(formatNumber(1234.5678)).toBe("1,234.5678");
    });

    it("formats string numbers correctly without trailing zeros", () => {
      expect(formatNumber("5000")).toBe("5,000");
    });

    it("formats decimal strings correctly", () => {
      expect(formatNumber("123.456")).toBe("123.456");
    });

    it("handles zero correctly without decimal point", () => {
      expect(formatNumber(0)).toBe("0");
    });

    it("handles large numbers correctly without trailing zeros", () => {
      expect(formatNumber(1000000)).toBe("1,000,000");
    });

    it("formats numbers with 7 decimal places correctly", () => {
      expect(formatNumber(123.1234567)).toBe("123.1234567");
    });

    it("removes trailing zeros from decimal numbers", () => {
      expect(formatNumber(100.1230000)).toBe("100.123");
    });

    it("removes all decimal places when they are zero", () => {
      expect(formatNumber(100.0)).toBe("100");
      expect(formatNumber("50.000")).toBe("50");
    });

    it("handles very small numbers correctly", () => {
      expect(formatNumber(0.0000001)).toBe("0.0000001");
    });

    it("handles invalid input gracefully", () => {
      expect(formatNumber(NaN)).toBe("0");
      expect(formatNumber(Infinity)).toBe("0");
      expect(formatNumber(-Infinity)).toBe("0");
      expect(formatNumber("invalid")).toBe("0");
    });

    it("respects custom decimal places", () => {
      expect(formatNumber(123.456789, 3)).toBe("123.457");
      expect(formatNumber(123.456, 5)).toBe("123.456");
      expect(formatNumber(123, 2, 2)).toBe("123.00");
    });
  });

  describe("calculateDepositDates", () => {
    it("calculates tomorrow start date correctly", () => {
      const now = new Date();
      const result = calculateDepositDates(30, 'tomorrow');
      
      expect(result.actualDays).toBe(30);
      expect(result.calendarDays).toBe(30);
      
      const expectedStart = new Date(now);
      expectedStart.setDate(expectedStart.getDate() + 1);
      expectedStart.setHours(0, 0, 0, 0);
      
      expect(result.startDate.toISOString()).toBe(expectedStart.toISOString());
    });

    it("calculates today start date correctly", () => {
      const result = calculateDepositDates(15, 'today');
      
      expect(result.actualDays).toBe(15);
      expect(result.calendarDays).toBe(15);
      
      const expectedStart = new Date();
      expectedStart.setHours(0, 0, 0, 0);
      expect(result.startDate.toISOString()).toBe(expectedStart.toISOString());
    });

    it("finds next business day correctly", () => {
      const result = calculateDepositDates(7, 'business_day');
      
      // The start date should be a weekday (1-5), not weekend (0 or 6)
      const startDay = result.startDate.getDay();
      expect(startDay).toBeGreaterThanOrEqual(1);
      expect(startDay).toBeLessThanOrEqual(5);
    });
  });

  describe("calculateInterest", () => {
    it("calculates simple interest correctly", () => {
      const result = calculateInterest(1000, 0.05, 365, 'simple');
      
      expect(result.principal).toBe(1000);
      expect(result.annualRate).toBe(0.05);
      expect(result.days).toBe(365);
      expect(result.interest).toBeCloseTo(50, 6);
      expect(result.totalAmount).toBeCloseTo(1050, 6);
    });

    it("calculates act_365 interest correctly", () => {
      const result = calculateInterest(1000, 0.05, 365, 'act_365');
      
      expect(result.interest).toBeCloseTo(50, 6);
      expect(result.totalAmount).toBeCloseTo(1050, 6);
    });

    it("calculates act_360 interest correctly", () => {
      const result = calculateInterest(1000, 0.05, 360, 'act_360');
      
      expect(result.interest).toBeCloseTo(50, 6);
      expect(result.totalAmount).toBeCloseTo(1050, 6);
    });

    it("calculates compound interest correctly", () => {
      const result = calculateInterest(1000, 0.05, 365, 'compound');
      
      expect(result.interest).toBeGreaterThan(50); // Compound should be higher
      expect(result.totalAmount).toBeGreaterThan(1050);
    });

    it("throws error for invalid parameters", () => {
      expect(() => calculateInterest(-100, 0.05, 365)).toThrow();
      expect(() => calculateInterest(1000, -0.05, 365)).toThrow();
      expect(() => calculateInterest(1000, 0.05, -365)).toThrow();
    });

    it("calculates partial year correctly", () => {
      const result = calculateInterest(1000, 0.06, 182, 'act_365'); // ~6 months
      expect(result.interest).toBeCloseTo(30, 6); // 1000 * 0.06 * (182/365) ≈ 30
    });
  });

  describe("validateAmount", () => {
    it("validates amount within range correctly", () => {
      const result = validateAmount(500, 100, 1000, 10000, undefined);
      expect(result.valid).toBe(true);
    });

    it("rejects amount below minimum", () => {
      const result = validateAmount(50, 100, 1000, 10000, undefined);
      expect(result.valid).toBe(false);
      expect(result.error).toContain('below minimum');
    });

    it("rejects amount above maximum", () => {
      const result = validateAmount(1500, 100, 1000, 10000, undefined);
      expect(result.valid).toBe(false);
      expect(result.error).toContain('above maximum');
    });

    it("rejects amount above available balance", () => {
      const result = validateAmount(500, 100, 1000, 200, undefined);
      expect(result.valid).toBe(false);
      expect(result.error).toContain('Insufficient balance');
    });

    it("rejects invalid input", () => {
      expect(validateAmount('invalid', 100, 1000, 10000, undefined).valid).toBe(false);
      expect(validateAmount(-100, 100, 1000, 10000, undefined).valid).toBe(false);
      expect(validateAmount(0, 100, 1000, 10000, undefined).valid).toBe(false);
    });
  });

  describe("generateRequestId", () => {
    it("generates unique request IDs", () => {
      const id1 = generateRequestId();
      const id2 = generateRequestId();
      expect(id1).not.toBe(id2);
    });

    it("generates IDs with correct format", () => {
      const id = generateRequestId();
      expect(id).toMatch(/^request_\d+_[a-z0-9]+$/);
    });
  });

  describe("isSameBusinessDay", () => {
    it("returns true for same day", () => {
      const date1 = new Date('2024-01-01T12:00:00Z');
      const date2 = new Date('2024-01-01T15:30:00Z');
      expect(isSameBusinessDay(date1, date2)).toBe(true);
    });

    it("returns false for different days", () => {
      const date1 = new Date('2024-01-01T12:00:00Z');
      const date2 = new Date('2024-01-02T12:00:00Z');
      expect(isSameBusinessDay(date1, date2)).toBe(false);
    });

    it("handles weekend dates", () => {
      const saturday = new Date('2024-01-06T12:00:00Z'); // Saturday
      const sunday = new Date('2024-01-07T12:00:00Z'); // Sunday
      expect(isSameBusinessDay(saturday, sunday)).toBe(false);
    });
  });

  describe("Enhanced error handling", () => {
    it("handles edge cases gracefully", () => {
      expect(formatNumber('')).toBe('0');
      expect(formatNumber(null as any)).toBe('0');
      expect(formatNumber(undefined as any)).toBe('0');
      
      expect(getDaysRemaining('')).toBe(0);
      expect(getDaysRemaining('invalid-date')).toBe(0);
    });
  });
});
