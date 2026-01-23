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

    const { secret, qrCodeUrl, backupCodes, otpauth } = await TwoFactorService.setup(user.userId, user.email);

    res.status(200).json({ secret, qrCodeUrl, backupCodes, otpauth });
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    logger.error({ error: errorMessage }, '2FA setup failed');
    res.status(500).json({ error: 'Failed to set up 2FA' });
  }
}
