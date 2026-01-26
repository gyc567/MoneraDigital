import type { VercelRequest, VercelResponse } from '@vercel/node';
import { TwoFactorService } from '../../../src/lib/two-factor-service.js';
import { verifyToken } from '../../../src/lib/auth-middleware.js';
import { TwoFactorStatusResponseSchema } from '../../../src/lib/two-factor-schemas.js';
import logger from '../../../src/lib/logger.js';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'GET') {
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

    // Get 2FA status
    const status = await TwoFactorService.getStatus(user.userId);

    // Validate response
    const validated = TwoFactorStatusResponseSchema.safeParse({
      enabled: status.enabled,
      remainingBackupCodes: status.remainingBackupCodes,
    });

    if (!validated.success) {
      return res.status(500).json({
        error: 'Internal Server Error'
      });
    }

    return res.status(200).json(validated.data);
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    logger.error({ error: errorMessage }, '2FA Status error');
    return res.status(500).json({
      error: 'Internal Server Error',
      message: errorMessage
    });
  }
}
