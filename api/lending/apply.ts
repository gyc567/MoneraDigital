import type { VercelRequest, VercelResponse } from '@vercel/node';
import { LendingService } from '../../src/lib/lending-service.js';
import { verifyToken } from '../../src/lib/auth-middleware.js';
import logger from '../../src/lib/logger.js';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method not allowed' });
  }

  const user = verifyToken(req);
  if (!user) {
    return res.status(401).json({ error: 'Unauthorized' });
  }

  const { asset, amount, durationDays } = req.body;

  if (!asset || !amount || !durationDays) {
    return res.status(400).json({ error: 'Missing required fields' });
  }

  try {
    const position = await LendingService.applyForLending(
      user.userId,
      asset,
      Number(amount),
      Number(durationDays)
    );

    return res.status(201).json(position);
  } catch (error: any) {
    logger.error({ error: error.message, userId: user.userId }, 'Lending application handler failed');
    return res.status(500).json({ error: 'Internal server error' });
  }
}
