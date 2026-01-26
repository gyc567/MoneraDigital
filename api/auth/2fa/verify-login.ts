import type { VercelRequest, VercelResponse } from '@vercel/node';
import { TwoFactorService } from '../../../src/lib/two-factor-service.js';
import { SessionService } from '../../../src/lib/session-service.js';
import { TwoFactorVerifyLoginRequestSchema } from '../../../src/lib/two-factor-schemas.js';
import { generateToken } from '../../../src/lib/auth-service.js';
import { db } from '../../../src/lib/db.js';
import { users } from '../../../src/db/schema.js';
import { eq } from 'drizzle-orm';
import logger from '../../../src/lib/logger.js';

/**
 * POST /api/auth/2fa/verify-login
 *
 * Completes the 2FA login flow by verifying a TOTP token or backup code.
 *
 * Flow:
 * 1. Receive sessionId + TOTP/backup code from client
 * 2. Look up pending session by sessionId
 * 3. Verify the token/code via TwoFactorService
 * 4. If valid: clear session, generate JWT token
 * 5. Return JWT to client
 *
 * Error responses:
 * - 400: Session expired or invalid code
 * - 401: Bad request format
 * - 500: Server error
 */
export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method Not Allowed' });
  }

  try {
    // 1. Validate request format
    const validated = TwoFactorVerifyLoginRequestSchema.safeParse(req.body);
    if (!validated.success) {
      return res.status(400).json({
        error: 'Invalid request',
        message: 'Token is required'
      });
    }

    const { sessionId, token } = validated.data;

    // 2. Retrieve pending session
    const session = await SessionService.getPendingLoginSession(sessionId);
    if (!session) {
      logger.warn({ sessionId }, '2FA verify-login: session not found or expired');
      return res.status(400).json({
        error: 'Session expired',
        message: 'Your login session has expired. Please try logging in again.'
      });
    }

    const userId = session.userId;

    // 3. Verify TOTP token or backup code
    const isValid = await TwoFactorService.verify(userId, token);
    if (!isValid) {
      logger.warn({ userId, sessionId }, '2FA verify-login: invalid token');
      return res.status(400).json({
        error: 'Invalid code',
        message: 'The code you entered is invalid or has expired.'
      });
    }

    // 4. Clear session (prevent reuse)
    await SessionService.clearPendingLoginSession(sessionId);

    // 5. Generate JWT token
    const [user] = await db.select().from(users).where(eq(users.id, userId));
    if (!user) {
      logger.error({ userId }, '2FA verify-login: user not found after session validation');
      return res.status(500).json({
        error: 'Internal Server Error',
        message: 'Failed to complete login'
      });
    }

    const jwtToken = generateToken({ userId: user.id, email: user.email });

    logger.info({ userId, sessionId }, '2FA login verification successful');

    return res.status(200).json({
      success: true,
      token: jwtToken,
      message: '2FA verified successfully'
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
