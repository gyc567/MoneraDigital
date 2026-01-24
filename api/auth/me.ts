import type { VercelRequest, VercelResponse } from '@vercel/node';
import { verifyToken } from '../../src/lib/auth-middleware.js';
import { AuthService } from '../../src/lib/auth-service.js';
import logger from '../../src/lib/logger.js';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'GET') {
    return res.status(405).json({ error: 'Method Not Allowed' });
  }

  const user = verifyToken(req);
  if (!user) {
    return res.status(401).json({ code: 'MISSING_TOKEN', message: 'Authentication required' });
  }

  try {
    const userInfo = await AuthService.getUserById(user.userId);
    if (!userInfo) {
      return res.status(404).json({ error: 'User not found' });
    }

    res.status(200).json({
      user: {
        id: userInfo.id,
        email: userInfo.email,
      },
    });
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    logger.error({ error: errorMessage, userId: user.userId }, 'Failed to get user info');
    res.status(500).json({ error: 'Internal Server Error' });
  }
}