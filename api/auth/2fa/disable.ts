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

    const isValid = await TwoFactorService.verify(user.userId, token);
    if (!isValid) {
      return res.status(400).json({ error: 'Invalid verification code' });
    }

    const { db } = await import('../../../src/lib/db.js');
    const { users } = await import('../../../src/db/schema.js');
    const { eq } = await import('drizzle-orm');

    await db.update(users)
      .set({ twoFactorEnabled: false })
      .where(eq(users.id, user.userId));

    res.status(200).json({ success: true });
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    logger.error({ error: errorMessage }, '2FA disable failed');
    res.status(500).json({ error: 'Failed to disable 2FA' });
  }
}
