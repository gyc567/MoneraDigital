import type { VercelRequest, VercelResponse } from '@vercel/node';
import { TwoFactorService } from '../../../src/lib/two-factor-service.js';
import { verifyToken } from '../../../src/lib/auth-middleware.js';
import logger from '../../../src/lib/logger.js';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method not allowed' });
  }

  const user = verifyToken(req);
  if (!user) {
    return res.status(401).json({ error: 'Unauthorized' });
  }

  try {
    const data = await TwoFactorService.setup(user.userId, user.email);
    return res.status(200).json(data);
  } catch (error: any) {
    logger.error({ error, userId: user.userId }, '2FA setup failed');
    return res.status(500).json({ error: 'Failed to setup 2FA' });
  }
}
