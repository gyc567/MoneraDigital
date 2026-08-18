// @vitest-environment node

import { describe, expect, it } from "vitest";
import { configDefaults } from "vitest/config";

import vitestConfig from "./vitest.config";

interface VitestConfigShape {
  test?: {
    include?: string[];
    exclude?: string[];
  };
}

describe("Vitest configuration", () => {
  it("collects only project unit tests and preserves dependency exclusions", () => {
    const config = vitestConfig as VitestConfigShape;

    expect(config.test?.include).toEqual([
      "src/**/*.test.{ts,tsx}",
      "api/**/*.test.ts",
      "tests/dev-environment.test.ts",
      "vitest.config.test.ts",
    ]);
    expect(config.test?.exclude).toEqual(expect.arrayContaining(configDefaults.exclude));
  });
});
