import type { VercelRequest, VercelResponse } from '@vercel/node';
import { WithdrawalService } from '../../../src/lib/withdrawal-service.js';
import { verifyToken } from '../../../src/lib/auth-middleware.js';
import logger from '../../../src/lib/logger.js';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'GET') {
    return res.status(405).json({ error: 'Method not allowed' });
  }

  const user = verifyToken(req);
  if (!user) {
    return res.status(401).json({ error: 'Unauthorized' });
  }

  const { id } = req.query;

  if (!id) {
    return res.status(400).json({ error: 'Withdrawal ID required' });
  }

  try {
    const withdrawal = await WithdrawalService.getWithdrawalDetails(user.userId, Number(id));

    return res.status(200).json({
      id: withdrawal.id,
      status: withdrawal.status,
      amount: withdrawal.amount,
      asset: withdrawal.asset,
      toAddress: withdrawal.toAddress,
      txHash: withdrawal.txHash,
      fromAddress: withdrawal.fromAddress,
      createdAt: withdrawal.createdAt,
      completedAt: withdrawal.completedAt,
      failureReason: withdrawal.failureReason,
    });
  } catch (error: any) {
    logger.error({ error: error.message, userId: user.userId }, 'GET /api/withdrawals/:id failed');

    if (error.message.includes('not found')) {
      return res.status(404).json({ error: error.message });
    }

    return res.status(500).json({ error: 'Failed to fetch withdrawal details' });
  }
}
