import { z } from 'zod';
import { db } from './db.js';
import { withdrawals, withdrawalAddresses } from '../db/schema.js';
import { eq, and } from 'drizzle-orm';
import logger from './logger.js';
import { safeheronService } from './safeheron-service.js';
import { FeeCalculationService } from './fee-calculation-service.js';

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
   * Initiate a withdrawal
   */
  static async initiateWithdrawal(
    userId: number,
    addressId: number,
    amount: string,
    asset: 'BTC' | 'ETH' | 'USDC' | 'USDT',
    chain?: string
  ) {
    const validated = withdrawalSchema.parse({
      addressId,
      amount,
      asset,
      chain,
    });

    logger.info({ userId, addressId, amount, asset, chain }, 'Initiating withdrawal');

    try {
      // Verify the address belongs to user and is verified
      const [address] = await db
        .select()
        .from(withdrawalAddresses)
        .where(
          and(
            eq(withdrawalAddresses.id, validated.addressId),
            eq(withdrawalAddresses.userId, userId),
            eq(withdrawalAddresses.isVerified, true)
          )
        );

      if (!address || address.deactivatedAt) {
        throw new Error('Address not found or not verified');
      }

      const selectedChain = chain || FeeCalculationService.getDefaultChain(asset);
      const feeResult = await FeeCalculationService.calculate(
        validated.asset,
        validated.amount,
        selectedChain
      );

      // Create withdrawal record with PENDING status
      const [withdrawal] = await db
        .insert(withdrawals)
        .values({
          userId,
          fromAddressId: validated.addressId,
          amount: validated.amount,
          asset: validated.asset,
          toAddress: address.address,
          status: 'PENDING',
          feeAmount: feeResult.fee,
          receivedAmount: feeResult.receivedAmount,
          chain: selectedChain,
        })
        .returning();

      logger.info({ withdrawalId: withdrawal.id, userId, addressId }, 'Withdrawal initiated successfully');

      this.processWithdrawalAsync(withdrawal.id);

      return {
        ...withdrawal,
        txHash: null,
      };
    } catch (error: any) {
      logger.error({ error: error.message, userId, addressId }, 'Failed to initiate withdrawal');
      throw error;
    }
  }

  /**
   * Process withdrawal asynchronously via Safeheron
   */
  private static async processWithdrawalAsync(withdrawalId: number) {
    try {
      await db
        .update(withdrawals)
        .set({ status: 'PROCESSING' })
        .where(eq(withdrawals.id, withdrawalId));

      const [withdrawal] = await db
        .select()
        .from(withdrawals)
        .where(eq(withdrawals.id, withdrawalId));

      if (!withdrawal) {
        throw new Error('Withdrawal not found');
      }

      const vaultId = process.env.SAFEHERON_VAULT_ID || 'default_vault';
      const assetId = safeheronService.getAssetId(withdrawal.asset, withdrawal.chain || undefined);

      const result = await safeheronService.coinOut(
        vaultId,
        assetId,
        withdrawal.amount,
        withdrawal.toAddress,
        `Withdrawal #${withdrawalId}`
      );

      if (result) {
        await db
          .update(withdrawals)
          .set({
            status: result.status === 'COMPLETED' ? 'COMPLETED' : 'PROCESSING',
            txHash: result.txHash,
            safeheronTxId: result.txId,
            completedAt: result.status === 'COMPLETED' ? new Date() : null,
          })
          .where(eq(withdrawals.id, withdrawalId));

        logger.info(
          { withdrawalId, safeheronTxId: result.txId, status: result.status },
          'Withdrawal processed via Safeheron'
        );
      }
    } catch (error: any) {
      logger.error(
        { error: error.message, withdrawalId },
        'Failed to process withdrawal via Safeheron'
      );

      await db
        .update(withdrawals)
        .set({
          status: 'FAILED',
          failureReason: error.message,
        })
        .where(eq(withdrawals.id, withdrawalId));
    }
  }

  /**
   * Initiate withdrawal with 2FA verification
   */
  static async initiateWithdrawalWith2FA(
    userId: number,
    addressId: number,
    amount: string,
    asset: 'BTC' | 'ETH' | 'USDC' | 'USDT',
    chain: string,
    twoFactorCode: string
  ) {
    const { TwoFactorService } = await import('./two-factor-service.js');
    const isValid = await TwoFactorService.verify(userId, twoFactorCode);

    if (!isValid) {
      throw new Error('Invalid 2FA code');
    }

    return this.initiateWithdrawal(userId, addressId, amount, asset, chain);
  }

  /**
   * Get withdrawal history for user
   */
  static async getWithdrawalHistory(
    userId: number,
    filters?: {
      status?: 'PENDING' | 'PROCESSING' | 'COMPLETED' | 'FAILED';
      asset?: string;
      limit?: number;
      offset?: number;
    }
  ) {
    logger.info({ userId, filters }, 'Fetching withdrawal history');

    try {
      const query = db.select().from(withdrawals).where(eq(withdrawals.userId, userId));

      // Apply filters if provided
      if (filters?.status) {
        // Note: Drizzle doesn't support chaining directly, so we'd need to build the query differently
        // For now, we fetch all and filter in-app
      }

      let historyResults = await db
        .select()
        .from(withdrawals)
        .where(eq(withdrawals.userId, userId));

      // Apply status filter if provided
      if (filters?.status) {
        historyResults = historyResults.filter((w) => w.status === filters.status);
      }

      // Apply asset filter if provided
      if (filters?.asset) {
        historyResults = historyResults.filter((w) => w.asset === filters.asset);
      }

      // Apply pagination
      const limit = filters?.limit || 50;
      const offset = filters?.offset || 0;
      const total = historyResults.length;
      const paginatedResults = historyResults.slice(offset, offset + limit);

      return {
        withdrawals: paginatedResults,
        total,
        hasMore: offset + limit < total,
      };
    } catch (error: any) {
      logger.error({ error: error.message, userId }, 'Failed to fetch withdrawal history');
      throw error;
    }
  }

  /**
   * Get withdrawal details by ID
   */
  static async getWithdrawalDetails(userId: number, withdrawalId: number) {
    logger.info({ userId, withdrawalId }, 'Fetching withdrawal details');

    try {
      const [withdrawal] = await db
        .select()
        .from(withdrawals)
        .where(
          and(
            eq(withdrawals.id, withdrawalId),
            eq(withdrawals.userId, userId)
          )
        );

      if (!withdrawal) {
        throw new Error('Withdrawal not found');
      }

      // Get address information
      const [address] = await db
        .select()
        .from(withdrawalAddresses)
        .where(eq(withdrawalAddresses.id, withdrawal.fromAddressId));

      return {
        ...withdrawal,
        fromAddress: address,
      };
    } catch (error: any) {
      logger.error({ error: error.message, userId, withdrawalId }, 'Failed to fetch withdrawal details');
      throw error;
    }
  }

  /**
   * Check if user has a default/primary address
   */
  static async hasPrimaryAddress(userId: number): Promise<boolean> {
    try {
      const [primary] = await db
        .select()
        .from(withdrawalAddresses)
        .where(
          and(
            eq(withdrawalAddresses.userId, userId),
            eq(withdrawalAddresses.isPrimary, true),
            eq(withdrawalAddresses.isVerified, true)
          )
        );

      return !!primary && !primary.deactivatedAt;
    } catch (error: any) {
      logger.error({ error: error.message, userId }, 'Failed to check primary address');
      return false;
    }
  }

  /**
   * Get primary withdrawal address
   */
  static async getPrimaryAddress(userId: number) {
    try {
      const [primary] = await db
        .select()
        .from(withdrawalAddresses)
        .where(
          and(
            eq(withdrawalAddresses.userId, userId),
            eq(withdrawalAddresses.isPrimary, true),
            eq(withdrawalAddresses.isVerified, true)
          )
        );

      if (!primary || primary.deactivatedAt) {
        return null;
      }

      return primary;
    } catch (error: any) {
      logger.error({ error: error.message, userId }, 'Failed to get primary address');
      return null;
    }
  }
}
