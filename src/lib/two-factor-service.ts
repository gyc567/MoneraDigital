import { z } from 'zod';
import logger from './logger.js';

/**
 * Two Factor Service
 * 
 * KISS: Simple API client for 2FA operations
 * All database operations are handled by Go backend
 */

const verify2FASchema = z.object({
  token: z.string().length(6, '2FA code must be 6 digits'),
});

const enable2FASchema = z.object({
  token: z.string().length(6, '2FA code must be 6 digits'),
});

const disable2FASchema = z.object({
  token: z.string().length(6, '2FA code must be 6 digits'),
});

export class TwoFactorService {
  /**
   * Setup 2FA via API
   */
  static async setup2FA() {
    logger.info('Setting up 2FA');

    const token = localStorage.getItem('token');
    if (!token) {
      throw new Error('Not authenticated');
    }

    const response = await fetch('/api/auth/2fa/setup', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to setup 2FA');
    }

    const data = await response.json();
    logger.info('2FA setup successful');

    return data;
  }

  /**
   * Enable 2FA via API
   */
  static async enable2FA(token: string) {
    const validated = enable2FASchema.parse({ token });

    logger.info('Enabling 2FA');

    const authToken = localStorage.getItem('token');
    if (!authToken) {
      throw new Error('Not authenticated');
    }

    const response = await fetch('/api/auth/2fa/enable', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${authToken}`,
      },
      body: JSON.stringify({ token: validated.token }),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to enable 2FA');
    }

    logger.info('2FA enabled successfully');
    return response.json();
  }

  /**
   * Disable 2FA via API
   */
  static async disable2FA(token: string) {
    const validated = disable2FASchema.parse({ token });

    logger.info('Disabling 2FA');

    const authToken = localStorage.getItem('token');
    if (!authToken) {
      throw new Error('Not authenticated');
    }

    const response = await fetch('/api/auth/2fa/disable', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${authToken}`,
      },
      body: JSON.stringify({ token: validated.token }),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to disable 2FA');
    }

    logger.info('2FA disabled successfully');
    return response.json();
  }

  /**
   * Get 2FA status via API
   */
  static async get2FAStatus() {
    const token = localStorage.getItem('token');
    if (!token) {
      return { enabled: false };
    }

    const response = await fetch('/api/auth/2fa/status', {
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    });

    if (!response.ok) {
      return { enabled: false };
    }

    return response.json();
  }

  /**
   * Verify 2FA token via API
   */
  static async verify2FAToken(token: string) {
    const validated = verify2FASchema.parse({ token });

    logger.info('Verifying 2FA token');

    const authToken = localStorage.getItem('token');
    if (!authToken) {
      throw new Error('Not authenticated');
    }

    const response = await fetch('/api/auth/2fa/verify', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${authToken}`,
      },
      body: JSON.stringify({ token: validated.token }),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Invalid 2FA token');
    }

    return response.json();
  }
}
