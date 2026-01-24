import type { VercelRequest, VercelResponse } from '@vercel/node';
import { AuthService } from '../../src/lib/auth-service.js';
import logger from '../../src/lib/logger.js';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method Not Allowed' });
  }

  try {
    const { email, password } = req.body;
    if (!email || !password) {
      return res.status(400).json({ error: 'Email and password are required' });
    }

    const result = await AuthService.login(email, password);

    if (result.requires2FA) {
      return res.status(200).json({
        requires2FA: true,
        userId: result.userId,
      });
    }

    return res.status(200).json({
      user: result.user,
      token: result.token,
    });
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    const errorStack = error instanceof Error ? error.stack : undefined;
    logger.error({ error: errorMessage, stack: errorStack }, 'Login failed');
    if (errorMessage === 'Invalid email or password') {
      return res.status(401).json({ code: 'INVALID_CREDENTIALS', error: errorMessage });
    }
    res.status(500).json({ error: 'Internal Server Error' });
  }
}
