/**
 * Development Configuration Validator
 *
 * Utilities to validate that the development environment is properly configured,
 * specifically focusing on Vite proxy and API endpoint routing.
 *
 * Used to catch configuration issues early during development.
 */

/**
 * Validates that Vite proxy is working correctly by testing the /api/health endpoint
 *
 * Returns: true if proxy is working, false otherwise
 */
export async function validateViteProxy(): Promise<boolean> {
  if (import.meta.env.PROD) {
    return true; // Skip in production
  }

  try {
    const response = await fetch("/api/health", {
      method: "GET",
      headers: { "Content-Type": "application/json" },
    });

    if (!response.ok) {
      console.warn(
        "[Dev Config] Vite proxy health check failed with status:",
        response.status
      );
      return false;
    }

    console.debug("[Dev Config] ✓ Vite proxy is working correctly");
    return true;
  } catch (error) {
    console.warn(
      "[Dev Config] Failed to validate Vite proxy:",
      error instanceof Error ? error.message : String(error)
    );
    return false;
  }
}

/**
 * Validates that the development environment is properly configured
 *
 * Checks:
 * 1. Vite proxy is working (API calls route to backend)
 * 2. API base URL is not hardcoded in development
 *
 * Returns: true if environment is valid, false otherwise
 */
export async function validateDevEnvironment(): Promise<boolean> {
  if (import.meta.env.PROD) {
    return true; // Skip in production
  }

  console.debug("[Dev Config] Validating development environment...");

  // Check 1: Vite proxy
  const proxyWorking = await validateViteProxy();
  if (!proxyWorking) {
    console.error(
      "[Dev Config] ✗ Vite proxy validation failed. " +
      "This may cause 401 errors when accessing 2FA endpoints. " +
      "Ensure: " +
      "1. Frontend is running on http://localhost:5001 " +
      "2. Backend is running on http://localhost:8081 " +
      "3. Vite config has proxy: { '/api': { target: 'http://localhost:8081' } }"
    );
    return false;
  }

  // Check 2: API base URL
  const apiBase = import.meta.env.VITE_API_BASE_URL;
  if (apiBase && apiBase !== "") {
    console.warn(
      "[Dev Config] VITE_API_BASE_URL is set to:",
      apiBase,
      "In development, this should be empty to use Vite proxy"
    );
  }

  console.debug("[Dev Config] ✓ Development environment validation passed");
  return true;
}

/**
 * Logs current development configuration for debugging
 */
export function logDevConfig(): void {
  if (import.meta.env.PROD) {
    return; // Skip in production
  }

  console.debug("[Dev Config] Current configuration:");
  console.debug("  - Mode:", import.meta.env.MODE);
  console.debug("  - API Base URL:", import.meta.env.VITE_API_BASE_URL || "(using Vite proxy)");
  console.debug("  - Window location:", window.location.origin);
}
