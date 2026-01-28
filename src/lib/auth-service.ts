import { z } from 'zod';
import logger from './logger.js';

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

    // Store token
    if (data.accessToken || data.token) {
      localStorage.setItem('token', data.accessToken || data.token);
    }

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
    const token = localStorage.getItem('token');
    if (!token) {
      return null;
    }

    const response = await fetch('/api/auth/me', {
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    });

    if (!response.ok) {
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      return null;
    }

    return response.json();
  }

  /**
   * Logout
   */
  static logout() {
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    logger.info('User logged out');
  }

  /**
   * Check if user is authenticated
   */
  static isAuthenticated(): boolean {
    return !!localStorage.getItem('token');
  }
}
