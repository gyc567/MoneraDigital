import { z } from 'zod';
import logger from './logger.js';
import { tokenManager, type TokenPair } from './token-manager.js';

/**
 * Auth Service
 *
 * KISS: Simple API client for authentication operations
 * All database operations are handled by Go backend
 */

const loginSchema = z.object({
  email: z.string().email('Invalid email format'),
  password: z.string().min(1, 'Password is required'),
});

const registerSchema = z.object({
  email: z.string().email('Invalid email format'),
  password: z.string().min(8, 'Password must be at least 8 characters'),
});

export class AuthService {
  /**
   * Login via API
   */
  static async login(email: string, password: string) {
    const validated = loginSchema.parse({ email, password });

    logger.info({ email }, 'Attempting login');

    const response = await fetch('/api/auth/login', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(validated),
    });

    const data = await response.json();

    if (!response.ok) {
      throw new Error(data.error || 'Login failed');
    }

    const tokenPair: TokenPair = {
      accessToken: data.accessToken || data.token,
      refreshToken: data.refreshToken || '',
      tokenType: data.tokenType || 'Bearer',
      expiresIn: data.expiresIn || 86400,
    };

    tokenManager.setTokens(tokenPair);

    if (data.user) {
      localStorage.setItem('user', JSON.stringify(data.user));
    }

    logger.info({ email }, 'Login successful');
    return data;
  }

  /**
   * Register via API
   */
  static async register(email: string, password: string) {
    const validated = registerSchema.parse({ email, password });

    logger.info({ email }, 'Attempting registration');

    const response = await fetch('/api/auth/register', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(validated),
    });

    const data = await response.json();

    if (!response.ok) {
      throw new Error(data.error || 'Registration failed');
    }

    logger.info({ email }, 'Registration successful');
    return data;
  }

  /**
   * Get current user via API
   */
  static async getCurrentUser() {
    const token = tokenManager.getAccessToken();
    if (!token) {
      return null;
    }

    const response = await fetch('/api/auth/me', {
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    });

    if (!response.ok) {
      tokenManager.clearTokens();
      return null;
    }

    return response.json();
  }

  /**
   * Logout
   */
  static logout() {
    tokenManager.clearTokens();
    logger.info('User logged out');
  }

  /**
   * Check if user is authenticated
   */
  static isAuthenticated(): boolean {
    return tokenManager.isAuthenticated();
  }
}
