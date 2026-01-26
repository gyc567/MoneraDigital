import { z } from 'zod';

/**
 * Environment variable validation schema
 * Ensures all required environment variables are present and valid
 */
const envSchema = z.object({
  VITE_API_BASE_URL: z.string().url('VITE_API_BASE_URL must be a valid URL'),
});

/**
 * Validate environment variables at runtime
 * Logs error and returns false if validation fails
 *
 * @returns {boolean} True if environment is valid, false otherwise
 */
export function validateEnv(): boolean {
  try {
    envSchema.parse({
      VITE_API_BASE_URL: import.meta.env.VITE_API_BASE_URL,
    });
    return true;
  } catch (error: unknown) {
    if (error instanceof z.ZodError) {
      console.error('Environment validation failed:');
      error.errors.forEach((err) => {
        console.error(`  - ${err.path.join('.')}: ${err.message}`);
      });
    } else {
      console.error('Environment validation failed:', error);
    }
    return false;
  }
}

/**
 * Type-safe environment variables
 * Use this import to access environment variables with type safety
 */
export const env = envSchema.parse({
  VITE_API_BASE_URL: import.meta.env.VITE_API_BASE_URL,
});

/**
 * Export individual environment variables for convenience
 */
export const VITE_API_BASE_URL = env.VITE_API_BASE_URL;
