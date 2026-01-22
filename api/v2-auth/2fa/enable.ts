import type { VercelRequest, VercelResponse } from '@vercel/node';
import { verifyToken } from '../../../src/lib/auth-middleware.js';
import { TwoFactorService } from '../../../src/lib/two-factor-service.js';
import logger from '../../../src/lib/logger.js';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method Not Allowed' });
  }

  try {
    const user = verifyToken(req);
    if (!user) {
      return res.status(401).json({ error: 'Unauthorized' });
    }

    const { token } = req.body;
    if (!token) {
      return res.status(400).json({ error: 'Verification code is required' });
    }

    await TwoFactorService.enable(user.userId, token);

    res.status(200).json({ success: true });
  } catch (error: any) {
    logger.error({ error: error.message }, '2FA enable failed');
    res.status(500).json({ error: error.message || 'Failed to enable 2FA' });
  }
}
