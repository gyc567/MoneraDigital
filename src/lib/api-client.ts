/**
 * API Client for Monera Digital
 *
 * This client provides a unified way to make API requests with proper base URL configuration.
 * It supports both development (Vite proxy) and production (direct backend URL) environments.
 */

// API base URL configuration
// Development: Use Vite proxy (empty string means relative paths)
// Production: Set VITE_API_BASE_URL to your backend URL
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '';

// Helper function to build full API URLs
export function getApiUrl(path: string): string {
  // If base URL is configured (production), use it
  if (API_BASE_URL) {
    // Remove leading slash from path if present
    const cleanPath = path.startsWith('/') ? path.slice(1) : path;
    // Ensure base URL doesn't end with slash
    const cleanBase = API_BASE_URL.endsWith('/') ? API_BASE_URL.slice(0, -1) : API_BASE_URL;
    return `${cleanBase}/${cleanPath}`;
  }

  // Development: use relative path (Vite proxy handles it)
  return path;
}

/**
 * Type-safe fetch wrapper for API requests
 */
export async function apiRequest<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const url = getApiUrl(path);

  // Get token from localStorage for authenticated requests
  const token = typeof window !== 'undefined' ? localStorage.getItem('token') : null;

  const defaultOptions: RequestInit = {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { 'Authorization': `Bearer ${token}` } : {}),
      ...options.headers,
    },
  };

  const response = await fetch(url, defaultOptions);

  if (!response.ok) {
    const error = await response.json().catch(() => ({
      code: 'UNKNOWN_ERROR',
      message: response.statusText,
    }));
    throw new Error(error.message || 'Request failed');
  }

  return response.json() as Promise<T>;
}

/**
 * Get API base URL (useful for debugging)
 */
export function getApiBaseUrl(): string {
  return API_BASE_URL || '(Vite Proxy - /api)';
}
