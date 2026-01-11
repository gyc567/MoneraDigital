import { db } from './db';
import { walletCreationRequests } from '../db/schema';
import { eq } from 'drizzle-orm';
import { z } from 'zod';
import { ZodError } from 'zod';

// Request validation schema
const createWalletSchema = z.object({
  request_id: z.string().uuid(),
  user_id: z.number().int().positive(),
});

const getWalletStatusSchema = z.object({
  user_id: z.number().int().positive(),
});

const getDepositAddressesSchema = z.object({
  user_id: z.number().int().positive(),
  chain: z.string().optional(),
});

export type WalletCreationResult = {
  success: boolean;
  walletId?: string;
  address?: string;
  status: 'success' | 'creating' | 'failed';
  message: string;
};

export type WalletStatusResult = {
  is_opened: boolean;
  walletId?: string;
  address?: string;
  status?: 'success' | 'creating' | 'failed' | 'none';
};

export type DepositAddress = {
  chain: string;
  address: string;
  memo: string | null;
};

export type DepositAddressesResult = {
  addresses: DepositAddress[];
};

export class WalletService {
  async createWallet(userId: number, requestId: string): Promise<WalletCreationResult> {
    try {
      createWalletSchema.parse({ request_id: requestId, user_id: userId });

      const existingRequests = await db
        .select()
        .from(walletCreationRequests)
        .where(eq(walletCreationRequests.userId, userId));

      const pendingRequest = existingRequests.find(r => r.requestId === requestId);
      const successRequest = existingRequests.find(r => r.status === 'SUCCESS');

      if (successRequest) {
        return {
          success: true,
          walletId: successRequest.walletId || undefined,
          address: successRequest.address || undefined,
          status: 'success',
          message: 'Wallet already exists',
        };
      }

      if (pendingRequest && pendingRequest.status === 'CREATING') {
        return {
          success: false,
          status: 'creating',
          message: 'Wallet creation in progress',
        };
      }

      await db.insert(walletCreationRequests).values({
        requestId,
        userId,
        status: 'CREATING',
      });

      try {
        const safeheronResponse = await this.callSafeheronCreateWallet(userId);

        await db
          .update(walletCreationRequests)
          .set({
            status: 'SUCCESS',
            walletId: safeheronResponse.walletId,
            address: safeheronResponse.address,
            addresses: JSON.stringify(safeheronResponse.addresses),
            updatedAt: new Date(),
          })
          .where(eq(walletCreationRequests.requestId, requestId));

        return {
          success: true,
          walletId: safeheronResponse.walletId,
          address: safeheronResponse.address,
          status: 'success',
          message: 'Wallet created successfully',
        };
      } catch (error) {
        await db
          .update(walletCreationRequests)
          .set({
            status: 'FAILED',
            errorMessage: error instanceof Error ? error.message : 'Unknown error',
            updatedAt: new Date(),
          })
          .where(eq(walletCreationRequests.requestId, requestId));

        return {
          success: false,
          status: 'failed',
          message: 'Failed to create wallet. Please try again.',
        };
      }
    } catch (error) {
      if (error instanceof ZodError) {
        return {
          success: false,
          status: 'failed',
          message: error.errors[0].message,
        };
      }
      throw error;
    }
  }

  async getWalletStatus(userId: number): Promise<WalletStatusResult> {
    try {
      getWalletStatusSchema.parse({ user_id: userId });

      const requests = await db
        .select()
        .from(walletCreationRequests)
        .where(eq(walletCreationRequests.userId, userId));

      if (requests.length === 0) {
        return { is_opened: false, status: 'none' };
      }

      const latestRequest = requests[requests.length - 1];

      switch (latestRequest.status) {
        case 'SUCCESS':
          return {
            is_opened: true,
            walletId: latestRequest.walletId || undefined,
            address: latestRequest.address || undefined,
            status: 'success',
          };
        case 'CREATING':
          return {
            is_opened: false,
            status: 'creating',
          };
        case 'FAILED':
          return {
            is_opened: false,
            status: 'failed',
          };
        default:
          return { is_opened: false, status: 'none' };
      }
    } catch (error) {
      if (error instanceof ZodError) {
        throw error;
      }
      throw error;
    }
  }

  async getDepositAddresses(userId: number, chain?: string): Promise<DepositAddressesResult> {
    try {
      getDepositAddressesSchema.parse({ user_id: userId, chain });

      const requests = await db
        .select()
        .from(walletCreationRequests)
        .where(eq(walletCreationRequests.userId, userId));

      const successRequest = requests.find(r => r.status === 'SUCCESS');
      if (!successRequest) {
        throw new Error('No wallet account found for this user');
      }

      const addresses: DepositAddress[] = successRequest.addresses
        ? JSON.parse(successRequest.addresses)
        : [];

      if (chain) {
        return {
          addresses: addresses.filter(a => a.chain.toUpperCase() === chain.toUpperCase()),
        };
      }

      return { addresses };
    } catch (error) {
      if (error instanceof ZodError) {
        throw error;
      }
      throw error;
    }
  }

  private async callSafeheronCreateWallet(userId: number): Promise<{
    walletId: string;
    address: string;
    addresses: DepositAddress[];
  }> {
    await new Promise(resolve => setTimeout(resolve, 100));

    const walletId = `wallet_sfh_${Date.now()}`;
    const address = `0x${Date.now().toString(16).padStart(40, '0')}`;

    return {
      walletId,
      address,
      addresses: [
        { chain: 'ETH', address, memo: null },
        { chain: 'BSC', address: `0x${(Date.now() + 1).toString(16).padStart(40, '0')}`, memo: null },
        { chain: 'POLYGON', address: `0x${(Date.now() + 2).toString(16).padStart(40, '0')}`, memo: null },
      ],
    };
  }
}
