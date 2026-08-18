import { describe, it, expect } from "vitest";
import { validateEnv, env, VITE_API_BASE_URL } from "./env-validator";

describe("env-validator", () => {
  describe("validateEnv", () => {
    it("should return true when environment is configured", () => {
      const result = validateEnv();
      expect(result).toBe(true);
    });
  });

  describe("env", () => {
    it("should parse VITE_API_BASE_URL correctly", () => {
      expect(env.VITE_API_BASE_URL).toBeDefined();
      expect(typeof env.VITE_API_BASE_URL).toBe("string");
    });

    it("should have a valid URL format", () => {
      expect(env.VITE_API_BASE_URL).toMatch(/^https?:\/\/.+/);
    });

    it("should use the isolated test backend URL", () => {
      expect(env.VITE_API_BASE_URL).toBe("http://127.0.0.1:8081");
    });
  });

  describe("VITE_API_BASE_URL export", () => {
    it("should export VITE_API_BASE_URL as a string", () => {
      expect(typeof VITE_API_BASE_URL).toBe("string");
    });

    it("should match URL format", () => {
      expect(VITE_API_BASE_URL).toMatch(/^https?:\/\/.+/);
    });

    it("should equal the backend URL from env object", () => {
      expect(VITE_API_BASE_URL).toBe(env.VITE_API_BASE_URL);
    });
  });
});
