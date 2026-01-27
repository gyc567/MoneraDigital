import { describe, it, expect } from "vitest";
import { getDaysRemaining, formatNumber } from "../lib/wealth-utils";

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
  });

  describe("formatNumber", () => {
    it("formats integer numbers correctly", () => {
      expect(formatNumber(1000)).toBe("1,000.00");
    });

    it("formats decimal numbers correctly", () => {
      expect(formatNumber(1234.5678)).toBe("1,234.5678");
    });

    it("formats string numbers correctly", () => {
      expect(formatNumber("5000")).toBe("5,000.00");
    });

    it("formats decimal strings correctly", () => {
      expect(formatNumber("123.456")).toBe("123.456");
    });

    it("handles zero correctly", () => {
      expect(formatNumber(0)).toBe("0.00");
    });

    it("handles large numbers correctly", () => {
      expect(formatNumber(1000000)).toBe("1,000,000.00");
    });
  });
});
