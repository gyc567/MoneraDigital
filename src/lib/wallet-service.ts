import { z } from 'zod';
import logger from './logger.js';

/**
 * Wallet Service
 * 
 * KISS: Simple API client for wallet operations
 * All database operations are handled by Go backend
 */

const createWalletSchema = z.object({
  userId: z.number().int().positive(),
  productCode: z.string().min(1),
  currency: z.string().min(1),
});

export class WalletService {
  /**
   * Create wallet via API
   */
  static async createWallet(userId: number, productCode: string, currency: string) {
    const validated = createWalletSchema.parse({
      userId,
      productCode,
      currency,
    });

    logger.info({ userId, productCode, currency }, 'Creating wallet');

    const token = localStorage.getItem('token');
    if (!token) {
      throw new Error('Not authenticated');
    }

    const response = await fetch('/api/wallet/create', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
      },
      body: JSON.stringify(validated),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to create wallet');
    }

    const data = await response.json();
    logger.info({ walletId: data.wallet?.id }, 'Wallet created successfully');

    return data;
  }

  /**
   * Get wallet info via API
   */
  static async getWalletInfo(walletId: string) {
    const token = localStorage.getItem('token');
    if (!token) {
      throw new Error('Not authenticated');
    }

    const response = await fetch(`/api/wallet/${walletId}`, {
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    });

    if (!response.ok) {
      throw new Error('Failed to fetch wallet info');
    }

    return response.json();
  }
}
