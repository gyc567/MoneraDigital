import path from "path";

import { configDefaults, defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react-swc";

export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: "jsdom",
    setupFiles: ["./vitest.setup.ts"],
    css: true,
    include: [
      "src/**/*.test.{ts,tsx}",
      "api/**/*.test.ts",
      "tests/dev-environment.test.ts",
      "vitest.config.test.ts",
    ],
    exclude: [...configDefaults.exclude],
    env: {
      VITE_API_BASE_URL: "http://127.0.0.1:8081",
    },
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
});
