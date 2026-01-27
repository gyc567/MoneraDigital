import type { VercelRequest, VercelResponse } from '@vercel/node';
import logger from '../../../src/lib/logger.js';

/**
 * POST /api/auth/2fa/skip
 *
 * Allow users to skip 2FA verification during login.
 * Issues JWT token without completing 2FA verification.
 *
 * This is a convenience feature - user can enable 2FA again anytime.
 * All skip attempts are logged for audit purposes.
 */
export default async function handler(req: VercelRequest, res: VercelResponse) {
  const backendUrl = process.env.BACKEND_URL;

  if (!backendUrl) {
    return res.status(500).json({
      error: 'Server configuration error',
      message: 'Backend URL not configured',
    });
  }

  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method Not Allowed' });
  }

  try {
    const { userId } = req.body;

    // Validate userId
    if (!userId || typeof userId !== 'number') {
      return res.status(400).json({
        error: 'Invalid request',
        message: 'userId is required and must be a number',
      });
    }

    // Call Go backend to issue token and log skip attempt
    const response = await fetch(`${backendUrl}/api/auth/2fa/skip`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ userId }),
    });

    const data = await response.json();

    if (!response.ok) {
      logger.warn({ userId, error: data.error }, '2FA skip failed');
      return res.status(response.status).json(data);
    }

    logger.info({ userId }, '2FA verification skipped during login');

    return res.status(200).json(data);
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    logger.error({ error: errorMessage }, 'Skip 2FA error');
    return res.status(500).json({
      error: 'Internal Server Error',
      message: 'Failed to skip 2FA verification',
    });
  }
}
