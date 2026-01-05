import type { VercelRequest, VercelResponse } from '@vercel/node';
import { WithdrawalService } from '../../src/lib/withdrawal-service.js';
import { EmailService } from '../../src/lib/email-service.js';
import { verifyToken } from '../../src/lib/auth-middleware.js';
import logger from '../../src/lib/logger.js';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method === 'GET') {
    // Get withdrawal history
    const user = verifyToken(req);
    if (!user) {
      return res.status(401).json({ error: 'Unauthorized' });
    }

    try {
      const { status, asset, limit, offset } = req.query;
      const result = await WithdrawalService.getWithdrawalHistory(user.userId, {
        status: status as 'PENDING' | 'PROCESSING' | 'COMPLETED' | 'FAILED' | undefined,
        asset: asset as string | undefined,
        limit: limit ? Number(limit) : undefined,
        offset: offset ? Number(offset) : undefined,
      });

      return res.status(200).json(result);
    } catch (error: any) {
      logger.error({ error: error.message, userId: user.userId }, 'GET /api/withdrawals failed');
      return res.status(500).json({ error: 'Failed to fetch withdrawal history' });
    }
  }

  if (req.method === 'POST') {
    // Initiate withdrawal
    const user = verifyToken(req);
    if (!user) {
      return res.status(401).json({ error: 'Unauthorized' });
    }

    const { addressId, amount, asset } = req.body;

    if (!addressId || !amount || !asset) {
      return res.status(400).json({ error: 'Missing required fields' });
    }

    try {
      const withdrawal = await WithdrawalService.initiateWithdrawal(
        user.userId,
        addressId,
        amount,
        asset
      );

      // Send confirmation email (don't block on failure)
      try {
        await EmailService.sendWithdrawalConfirmation(user.email, {
          amount,
          asset,
          toAddress: withdrawal.toAddress,
          status: withdrawal.status,
          txHash: withdrawal.txHash,
        });
      } catch (emailError: any) {
        logger.warn({ error: emailError.message }, 'Failed to send withdrawal confirmation email');
      }

      return res.status(201).json({
        id: withdrawal.id,
        status: withdrawal.status,
        amount: withdrawal.amount,
        asset: withdrawal.asset,
        toAddress: withdrawal.toAddress,
        createdAt: withdrawal.createdAt,
        message: 'Withdrawal initiated successfully',
      });
    } catch (error: any) {
      logger.error({ error: error.message, userId: user.userId }, 'POST /api/withdrawals failed');

      if (error.message.includes('not found') || error.message.includes('verified')) {
        return res.status(400).json({ error: error.message });
      }

      return res.status(500).json({ error: 'Failed to initiate withdrawal' });
    }
  }

  return res.status(405).json({ error: 'Method not allowed' });
}
