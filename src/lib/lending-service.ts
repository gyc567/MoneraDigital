import { z } from 'zod';
import logger from './logger.js';

/**
 * Lending Service
 * 
 * KISS: Simple API client for lending operations
 * All database operations are handled by Go backend
 */

const applyLendingSchema = z.object({
  asset: z.enum(['BTC', 'ETH', 'USDT', 'USDC', 'SOL']),
  amount: z.string().refine((val) => !isNaN(parseFloat(val)) && parseFloat(val) > 0, 'Invalid amount'),
  durationDays: z.number().int().min(30).max(360),
});

export class LendingService {
  /**
   * Apply for lending via API
   */
  static async applyForLending(
    asset: string,
    amount: string,
    durationDays: number
  ) {
    const validated = applyLendingSchema.parse({
      asset,
      amount,
      durationDays,
    });

    logger.info({ asset, amount, durationDays }, 'Applying for lending');

    const token = localStorage.getItem('token');
    if (!token) {
      throw new Error('Not authenticated');
    }

    const response = await fetch('/api/lending/apply', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
      },
      body: JSON.stringify(validated),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to apply for lending');
    }

    const data = await response.json();
    logger.info({ positionId: data.position?.id }, 'Lending application successful');

    return data;
  }

  /**
   * Get user lending positions via API
   */
  static async getUserPositions() {
    const token = localStorage.getItem('token');
    if (!token) {
      throw new Error('Not authenticated');
    }

    const response = await fetch('/api/lending/positions', {
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    });

    if (!response.ok) {
      throw new Error('Failed to fetch lending positions');
    }

    return response.json();
  }

  /**
   * Calculate APY
   */
  static calculateAPY(asset: string, durationDays: number): number {
    const baseRates: Record<string, number> = {
      BTC: 4.5,
      ETH: 5.0,
      USDT: 10.0,
      USDC: 9.5,
      SOL: 7.0,
    };

    const baseRate = baseRates[asset] || 5.0;
    const durationBonus = Math.min((durationDays / 360) * 2, 2.0);
    
    return parseFloat((baseRate + durationBonus).toFixed(2));
  }

  /**
   * Calculate estimated yield
   */
  static calculateEstimatedYield(amount: string, apy: number, durationDays: number): string {
    const principal = parseFloat(amount);
    const yieldValue = principal * (apy / 100) * (durationDays / 365);
    return yieldValue.toFixed(8);
  }
}
