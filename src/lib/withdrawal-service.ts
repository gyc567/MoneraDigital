import { z } from 'zod';
import { db } from './db.js';
import { withdrawals, withdrawalAddresses } from '../db/schema.js';
import { eq, and } from 'drizzle-orm';
import logger from './logger.js';

export const withdrawalSchema = z.object({
  addressId: z.number().int().positive('Invalid address'),
  amount: z.string().refine((val) => !isNaN(parseFloat(val)) && parseFloat(val) > 0, 'Invalid amount'),
  asset: z.enum(['BTC', 'ETH', 'USDC', 'USDT']),
});

export class WithdrawalService {
  /**
   * Initiate a withdrawal
   */
  static async initiateWithdrawal(
    userId: number,
    addressId: number,
    amount: string,
    asset: 'BTC' | 'ETH' | 'USDC' | 'USDT'
  ) {
    const validated = withdrawalSchema.parse({
      addressId,
      amount,
      asset,
    });

    logger.info({ userId, addressId, amount, asset }, 'Initiating withdrawal');

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

      // For simulated processing: generate a mock tx hash
      const mockTxHash = `0x${Math.random().toString(16).substring(2)}${Math.random().toString(16).substring(2)}`;

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
        })
        .returning();

      logger.info({ withdrawalId: withdrawal.id, userId, addressId }, 'Withdrawal initiated successfully');

      // Simulate async processing - update status to COMPLETED
      // In production, this would be handled by a background job
      setTimeout(async () => {
        try {
          await db
            .update(withdrawals)
            .set({
              status: 'COMPLETED',
              txHash: mockTxHash,
              completedAt: new Date(),
            })
            .where(eq(withdrawals.id, withdrawal.id));

          logger.info({ withdrawalId: withdrawal.id }, 'Withdrawal processed successfully');
        } catch (error: any) {
          logger.error({ error: error.message, withdrawalId: withdrawal.id }, 'Failed to process withdrawal');
        }
      }, 2000); // Simulate processing after 2 seconds

      return {
        ...withdrawal,
        txHash: null, // Initially null since it's being processed
      };
    } catch (error: any) {
      logger.error({ error: error.message, userId, addressId }, 'Failed to initiate withdrawal');
      throw error;
    }
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
      let query = db.select().from(withdrawals).where(eq(withdrawals.userId, userId));

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
