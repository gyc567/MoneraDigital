import type { VercelRequest, VercelResponse } from '@vercel/node';
import { TwoFactorVerifyLoginRequestSchema } from '../../../src/lib/two-factor-schemas.js';
import logger from '../../../src/lib/logger.js';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method Not Allowed' });
  }

  try {
    // Validate request body
    const validated = TwoFactorVerifyLoginRequestSchema.safeParse(req.body);
    if (!validated.success) {
      return res.status(400).json({
        error: 'Invalid request',
        message: 'Token is required'
      });
    }

    const { sessionId } = validated.data;

    // TODO: Implement proper session tracking:
    // 1. Look up pending login session by sessionId (Redis or database)
    // 2. Get userId from session
    // 3. Call TwoFactorService.verify(userId, token)
    // 4. If valid, clear session and return JWT token

    logger.warn({ sessionId }, 'verify-login not yet implemented - session tracking required');

    return res.status(501).json({
      error: 'Not Implemented',
      message: 'Two-factor login verification will be implemented in a future update'
    });
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    logger.error({ error: errorMessage }, '2FA Verify Login error');
    return res.status(500).json({
      error: 'Internal Server Error',
      message: 'An unexpected error occurred'
    });
  }
}
