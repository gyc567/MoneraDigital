import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testMatch: 'tests/*.spec.ts',
  fullyParallel: false,
  workers: 1,
  reporter: 'list',
  use: {
    baseURL: 'http://127.0.0.1:5000',
    trace: 'on-first-retry',
  },
  webServer: [
    {
      command: 'go run cmd/server/main.go',
      url: 'http://127.0.0.1:8081/health',
      reuseExistingServer: !process.env.CI,
      stdout: 'pipe',
      stderr: 'pipe',
      env: { PORT: '8081' },
    },
    {
      command: 'npm run dev -- --port 5000',
      url: 'http://127.0.0.1:5000',
      reuseExistingServer: !process.env.CI,
      stdout: 'pipe',
      stderr: 'pipe',
    },
  ],
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
