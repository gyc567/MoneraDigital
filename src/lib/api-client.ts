/**
 * API Client for Monera Digital
 *
 * This client provides a unified way to make API requests with proper base URL configuration.
 * It supports both development (Vite proxy) and production (direct backend URL) environments.
 */

import { useNavigate } from "react-router-dom";

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
 * API Error class with structured error information
 */
export class ApiError extends Error {
  constructor(
    message: string,
    public readonly code: string,
    public readonly status: number,
    public readonly originalResponse?: unknown
  ) {
    super(message);
    this.name = 'ApiError';
  }
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

  // Parse response body
  let responseData: unknown;
  const contentType = response.headers.get('content-type');
  const isJson = contentType?.includes('application/json') || contentType?.includes('text/json');

  if (isJson) {
    try {
      responseData = await response.json();
    } catch {
      responseData = null;
    }
  }

  // Handle error responses
  if (!response.ok) {
    const status = response.status;
    const statusText = response.statusText;

    // Extract error message from response
    let errorMessage = statusText;
    let errorCode = 'UNKNOWN_ERROR';

    if (isJson && responseData && typeof responseData === 'object') {
      const data = responseData as Record<string, unknown>;
      errorMessage = (data.message as string) || statusText;
      errorCode = (data.code as string) || `HTTP_${status}`;
    }

    // Handle 401 Unauthorized specifically
    if (status === 401) {
      // Clear invalid token
      if (typeof window !== 'undefined') {
        localStorage.removeItem('token');
      }

      // Create error with specific code for 401
      errorCode = 'UNAUTHORIZED';
      errorMessage = 'Authentication required. Please log in again.';
    }

    throw new ApiError(errorMessage, errorCode, status, responseData);
  }

  // Return parsed JSON or empty object
  if (isJson && responseData) {
    return responseData as T;
  }

  return {} as T;
}

/**
 * Get API base URL (useful for debugging)
 */
export function getApiBaseUrl(): string {
  return API_BASE_URL || '(Vite Proxy - /api)';
}
