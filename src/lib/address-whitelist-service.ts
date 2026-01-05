import { z } from 'zod';
import { db } from './db.js';
import { withdrawalAddresses, addressVerifications, withdrawals } from '../db/schema.js';
import { eq, and } from 'drizzle-orm';
import { encrypt, decrypt } from './encryption.js';
import crypto from 'crypto';
import logger from './logger.js';

// Address type validation
const BTC_REGEX = /^(1[a-km-zA-HJ-NP-Z1-9]{25,34}|3[a-km-zA-HJ-NP-Z1-9]{25,34}|bc1[a-z0-9]{39,59})$/;
const ETH_REGEX = /^0x[a-fA-F0-9]{40}$/;

export const addressSchema = z.object({
  address: z.string().min(26, 'Address too short'),
  addressType: z.enum(['BTC', 'ETH', 'USDC', 'USDT']),
  label: z.string().min(1, 'Label required').max(50, 'Label too long'),
});

export class AddressWhitelistService {
  /**
   * Validate address format based on type
   */
  static validateAddressFormat(address: string, type: 'BTC' | 'ETH' | 'USDC' | 'USDT'): boolean {
    if (type === 'BTC') {
      return BTC_REGEX.test(address);
    }
    if (type === 'ETH' || type === 'USDC' || type === 'USDT') {
      return ETH_REGEX.test(address);
    }
    return false;
  }

  /**
   * Add a new withdrawal address
   */
  static async addAddress(
    userId: number,
    address: string,
    type: 'BTC' | 'ETH' | 'USDC' | 'USDT',
    label: string
  ) {
    // Validate input
    const validated = addressSchema.parse({ address, addressType: type, label });

    // Validate address format
    if (!this.validateAddressFormat(validated.address, validated.addressType)) {
      throw new Error(`Invalid ${validated.addressType} address format`);
    }

    logger.info({ userId, address: validated.address, type }, 'Adding withdrawal address');

    try {
      // Check if address already exists for this user
      const existing = await db
        .select()
        .from(withdrawalAddresses)
        .where(
          and(
            eq(withdrawalAddresses.userId, userId),
            eq(withdrawalAddresses.address, validated.address),
            eq(withdrawalAddresses.addressType, validated.addressType)
          )
        );

      if (existing.length > 0 && !existing[0].deactivatedAt) {
        throw new Error('Address already exists');
      }

      // Create new address
      const [newAddr] = await db
        .insert(withdrawalAddresses)
        .values({
          userId,
          address: validated.address,
          addressType: validated.addressType,
          label: validated.label,
          isVerified: false,
          isPrimary: false,
        })
        .returning();

      logger.info({ addressId: newAddr.id, userId }, 'Address created successfully');
      return newAddr;
    } catch (error: any) {
      logger.error({ error: error.message, userId }, 'Failed to add address');
      throw error;
    }
  }

  /**
   * Generate verification token and save to database
   */
  static async generateVerificationToken(addressId: number): Promise<string> {
    // Generate a random token
    const token = crypto.randomBytes(32).toString('hex');

    // Set expiration to 24 hours from now
    const expiresAt = new Date();
    expiresAt.setHours(expiresAt.getHours() + 24);

    // Encrypt token for storage
    const encryptedToken = encrypt(token);

    await db
      .insert(addressVerifications)
      .values({
        addressId,
        token: encryptedToken,
        expiresAt,
      });

    logger.info({ addressId }, 'Verification token generated');
    return token;
  }

  /**
   * Get all addresses for a user
   */
  static async getAddresses(userId: number) {
    logger.info({ userId }, 'Fetching addresses for user');

    try {
      const addresses = await db
        .select()
        .from(withdrawalAddresses)
        .where(eq(withdrawalAddresses.userId, userId));

      return addresses;
    } catch (error: any) {
      logger.error({ error: error.message, userId }, 'Failed to fetch addresses');
      throw error;
    }
  }

  /**
   * Verify address using token
   */
  static async verifyAddress(userId: number, token: string) {
    logger.info({ userId }, 'Verifying address');

    try {
      const encryptedToken = encrypt(token);

      // Find the verification record
      const [verification] = await db
        .select()
        .from(addressVerifications)
        .where(eq(addressVerifications.token, encryptedToken));

      if (!verification) {
        throw new Error('Invalid verification token');
      }

      // Check if token is expired
      if (new Date() > verification.expiresAt) {
        throw new Error('Verification token expired');
      }

      // Get the address and verify it belongs to the user
      const [address] = await db
        .select()
        .from(withdrawalAddresses)
        .where(eq(withdrawalAddresses.id, verification.addressId));

      if (!address || address.userId !== userId) {
        throw new Error('Address not found or unauthorized');
      }

      // Update address verification status
      await db
        .update(withdrawalAddresses)
        .set({
          isVerified: true,
          verifiedAt: new Date(),
        })
        .where(eq(withdrawalAddresses.id, verification.addressId));

      // Mark verification as completed
      await db
        .update(addressVerifications)
        .set({ verifiedAt: new Date() })
        .where(eq(addressVerifications.id, verification.id));

      logger.info({ addressId: verification.addressId, userId }, 'Address verified successfully');
      return address;
    } catch (error: any) {
      logger.error({ error: error.message, userId }, 'Failed to verify address');
      throw error;
    }
  }

  /**
   * Set an address as primary (only one per user)
   */
  static async setPrimaryAddress(userId: number, addressId: number) {
    logger.info({ userId, addressId }, 'Setting primary address');

    try {
      // Verify address belongs to user and is verified
      const [address] = await db
        .select()
        .from(withdrawalAddresses)
        .where(
          and(
            eq(withdrawalAddresses.id, addressId),
            eq(withdrawalAddresses.userId, userId)
          )
        );

      if (!address) {
        throw new Error('Address not found');
      }

      if (!address.isVerified) {
        throw new Error('Address must be verified before setting as primary');
      }

      // Remove primary status from all other addresses
      await db
        .update(withdrawalAddresses)
        .set({ isPrimary: false })
        .where(eq(withdrawalAddresses.userId, userId));

      // Set this address as primary
      const [updated] = await db
        .update(withdrawalAddresses)
        .set({ isPrimary: true })
        .where(eq(withdrawalAddresses.id, addressId))
        .returning();

      logger.info({ addressId, userId }, 'Primary address set successfully');
      return updated;
    } catch (error: any) {
      logger.error({ error: error.message, userId, addressId }, 'Failed to set primary address');
      throw error;
    }
  }

  /**
   * Deactivate an address
   */
  static async deactivateAddress(userId: number, addressId: number) {
    logger.info({ userId, addressId }, 'Deactivating address');

    try {
      // Verify address belongs to user
      const [address] = await db
        .select()
        .from(withdrawalAddresses)
        .where(
          and(
            eq(withdrawalAddresses.id, addressId),
            eq(withdrawalAddresses.userId, userId)
          )
        );

      if (!address) {
        throw new Error('Address not found');
      }

      // Update address with deactivation timestamp
      const [updated] = await db
        .update(withdrawalAddresses)
        .set({
          deactivatedAt: new Date(),
          isPrimary: false,
        })
        .where(eq(withdrawalAddresses.id, addressId))
        .returning();

      logger.info({ addressId, userId }, 'Address deactivated successfully');
      return updated;
    } catch (error: any) {
      logger.error({ error: error.message, userId, addressId }, 'Failed to deactivate address');
      throw error;
    }
  }

  /**
   * Get verified addresses for withdrawal (excluding deactivated)
   */
  static async getVerifiedAddressesForWithdrawal(userId: number) {
    logger.info({ userId }, 'Fetching verified addresses for withdrawal');

    try {
      const addresses = await db
        .select()
        .from(withdrawalAddresses)
        .where(
          and(
            eq(withdrawalAddresses.userId, userId),
            eq(withdrawalAddresses.isVerified, true)
          )
        );

      // Filter out deactivated addresses
      return addresses.filter((addr) => !addr.deactivatedAt);
    } catch (error: any) {
      logger.error({ error: error.message, userId }, 'Failed to fetch verified addresses');
      throw error;
    }
  }
}
