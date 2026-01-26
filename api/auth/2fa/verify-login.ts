import type { VercelRequest, VercelResponse } from '@vercel/node';
import { TwoFactorVerifyLoginRequestSchema } from '../../../src/lib/two-factor-schemas.js';

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
        message: 'Token and sessionId are required'
      });
    }

    // TODO: Implement proper session tracking:
    // 1. Look up pending login session by sessionId (Redis or database)
    // 2. Get userId from session
    // 3. Call TwoFactorService.verify(userId, token)
    // 4. If valid, clear session and return JWT token
    // For now, return error indicating implementation needed

    return res.status(400).json({
      error: 'Not implemented',
      message: 'Session tracking required for login 2FA verification'
    });
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    console.error('2FA Verify Login error:', errorMessage);
    return res.status(500).json({
      error: 'Internal Server Error',
      message: errorMessage
    });
  }
}
