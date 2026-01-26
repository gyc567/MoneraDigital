import type { VercelRequest, VercelResponse } from '@vercel/node';
import { TwoFactorService } from '../../../src/lib/two-factor-service.js';
import { verifyToken } from '../../../src/lib/auth-middleware.js';
import { TwoFactorDisableRequestSchema } from '../../../src/lib/two-factor-schemas.js';
import logger from '../../../src/lib/logger.js';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method Not Allowed' });
  }

  try {
    // Verify JWT authentication
    const user = verifyToken(req);
    if (!user) {
      return res.status(401).json({
        error: 'AUTH_REQUIRED',
        message: 'Authentication required'
      });
    }

    // Validate request body
    const validated = TwoFactorDisableRequestSchema.safeParse(req.body);
    if (!validated.success) {
      return res.status(400).json({
        error: 'Invalid request',
        message: 'Token must be a 6-digit number'
      });
    }

    const { token } = validated.data;

    // Call service to disable 2FA
    await TwoFactorService.disable(user.userId, token);

    return res.status(200).json({
      success: true,
      message: '2FA disabled successfully',
    });
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';

    if (errorMessage.includes('Invalid verification code')) {
      return res.status(400).json({
        error: 'Invalid code',
        message: errorMessage
      });
    }

    if (errorMessage.includes('not enabled')) {
      return res.status(400).json({
        error: 'Not enabled',
        message: errorMessage
      });
    }

    logger.error({ error: errorMessage }, '2FA Disable error');
    return res.status(500).json({
      error: 'Internal Server Error',
      message: errorMessage
    });
  }
}
