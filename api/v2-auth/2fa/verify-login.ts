import type { VercelRequest, VercelResponse } from '@vercel/node';
import { AuthService } from '../../../src/lib/auth-service.js';
import logger from '../../../src/lib/logger.js';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method Not Allowed' });
  }

  try {
    const { userId, token } = req.body;
    if (!userId || !token) {
      return res.status(400).json({ error: 'User ID and verification code are required' });
    }

    const result = await AuthService.verify2FAAndLogin(userId, token);

    res.status(200).json({
      user: result.user,
      token: result.token,
    });
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    logger.error({ error: errorMessage }, '2FA login verification failed');
    res.status(401).json({ error: errorMessage || 'Invalid verification code' });
  }
}
