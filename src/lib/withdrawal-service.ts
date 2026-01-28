import { z } from 'zod';
import logger from './logger.js';

/**
 * Withdrawal Service
 * 
 * KISS: Simple API client for withdrawal operations
 * All database operations are handled by Go backend
 */

export const withdrawalSchema = z.object({
  addressId: z.number().int().positive('Invalid address'),
  amount: z.string().refine((val) => !isNaN(parseFloat(val)) && parseFloat(val) > 0, 'Invalid amount'),
  asset: z.enum(['BTC', 'ETH', 'USDC', 'USDT']),
  chain: z.string().optional(),
});

export const withdrawalWith2FASchema = withdrawalSchema.extend({
  twoFactorCode: z.string().length(6, 'Invalid 2FA code'),
  chain: z.string(),
});

export class WithdrawalService {
  /**
   * Initiate a withdrawal via API
   */
  static async initiateWithdrawal(
    addressId: number,
    amount: string,
    asset: 'BTC' | 'ETH' | 'USDC' | 'USDT',
    twoFactorToken?: string
  ) {
    const validated = withdrawalSchema.parse({
      addressId,
      amount,
      asset,
    });

    logger.info({ addressId, amount, asset }, 'Initiating withdrawal');

    const token = localStorage.getItem('token');
    if (!token) {
      throw new Error('Not authenticated');
    }

    const response = await fetch('/api/withdrawals', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
      },
      body: JSON.stringify({
        addressId: validated.addressId,
        amount: validated.amount,
        asset: validated.asset,
        twoFactorToken: twoFactorToken || '',
      }),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to initiate withdrawal');
    }

    const data = await response.json();
    logger.info({ withdrawalId: data.order?.id }, 'Withdrawal initiated successfully');

    return data.order;
  }

  /**
   * Get withdrawal history via API
   */
  static async getWithdrawalHistory() {
    const token = localStorage.getItem('token');
    if (!token) {
      throw new Error('Not authenticated');
    }

    const response = await fetch('/api/withdrawals', {
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    });

    if (!response.ok) {
      throw new Error('Failed to fetch withdrawal history');
    }

    return response.json();
  }

  /**
   * Get withdrawal fees via API
   */
  static async getWithdrawalFees(asset: string, amount: string, chain: string) {
    const token = localStorage.getItem('token');
    if (!token) {
      throw new Error('Not authenticated');
    }

    const params = new URLSearchParams({ asset, amount, chain });
    const response = await fetch(`/api/withdrawals/fees?${params}`, {
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    });

    if (!response.ok) {
      throw new Error('Failed to fetch withdrawal fees');
    }

    return response.json();
  }
}
