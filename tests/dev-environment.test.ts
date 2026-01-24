/**
 * Development Environment Configuration Tests
 *
 * Tests that ensure development configuration is consistent across:
 * - Vite config (vite.config.ts)
 * - Start scripts (scripts/start-*.sh)
 * - Playwright config (playwright.config.ts)
 *
 * Purpose: Prevent port configuration drift and ensure proper proxy setup
 */

import { describe, it, expect } from "vitest";
import fs from "fs";
import path from "path";

const PROJECT_ROOT = process.cwd();
const EXPECTED_PORT = 5001;
const EXPECTED_BACKEND_PORT = 8081;

describe("Development Configuration Consistency", () => {
  it("should have consistent frontend port across configs", () => {
    // Read vite.config.ts
    const viteConfigPath = path.join(PROJECT_ROOT, "vite.config.ts");
    const viteConfig = fs.readFileSync(viteConfigPath, "utf-8");

    // Vite should be configured for port 5001
    expect(viteConfig).toMatch(/port:\s*5001/);
  });

  it("should not have port overrides in start scripts", () => {
    // Read scripts/start-dev.sh
    const startDevPath = path.join(PROJECT_ROOT, "scripts", "start-dev.sh");
    const startDev = fs.readFileSync(startDevPath, "utf-8");

    // Should not have --port override
    expect(startDev).not.toMatch(/npm run dev -- --port \d+/);
  });

  it("should not have port overrides in start-frontend.sh", () => {
    // Read scripts/start-frontend.sh
    const startFrontendPath = path.join(
      PROJECT_ROOT,
      "scripts",
      "start-frontend.sh"
    );
    const startFrontend = fs.readFileSync(startFrontendPath, "utf-8");

    // Should not have --port override
    expect(startFrontend).not.toMatch(/npm run dev -- --port \d+/);
  });

  it("should have Vite proxy configured for backend", () => {
    const viteConfigPath = path.join(PROJECT_ROOT, "vite.config.ts");
    const viteConfig = fs.readFileSync(viteConfigPath, "utf-8");

    // Should proxy /api requests to backend
    expect(viteConfig).toMatch(/\/api/);
    expect(viteConfig).toMatch(/target.*8081/);
  });

  it("should have consistent port in playwright.config.ts", () => {
    const playwrightConfigPath = path.join(
      PROJECT_ROOT,
      "playwright.config.ts"
    );
    const playwrightConfig = fs.readFileSync(playwrightConfigPath, "utf-8");

    // Should reference port 5001
    expect(playwrightConfig).toMatch(/5001/);
  });

  it("should not have hardcoded port 5000 in playwright config", () => {
    const playwrightConfigPath = path.join(
      PROJECT_ROOT,
      "playwright.config.ts"
    );
    const playwrightConfig = fs.readFileSync(playwrightConfigPath, "utf-8");

    // Should not have port 5000 (deprecated)
    expect(playwrightConfig).not.toMatch(/baseURL:.*5000/);
  });

  it("should have backend on port 8081", () => {
    const viteConfigPath = path.join(PROJECT_ROOT, "vite.config.ts");
    const viteConfig = fs.readFileSync(viteConfigPath, "utf-8");

    // Backend proxy should target 8081
    expect(viteConfig).toMatch(/8081/);
  });
});

describe("Development Configuration Documentation", () => {
  it("should document frontend port in scripts", () => {
    const startDevPath = path.join(PROJECT_ROOT, "scripts", "start-dev.sh");
    const startDev = fs.readFileSync(startDevPath, "utf-8");

    // Should mention port 5001
    expect(startDev).toMatch(/5001/);
  });

  it("should document Vite proxy in scripts", () => {
    const startFrontendPath = path.join(
      PROJECT_ROOT,
      "scripts",
      "start-frontend.sh"
    );
    const startFrontend = fs.readFileSync(startFrontendPath, "utf-8");

    // Should mention proxy or backend
    expect(startFrontend).toMatch(/(proxy|api|8081)/i);
  });
});
