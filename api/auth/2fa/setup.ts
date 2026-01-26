import type { VercelRequest, VercelResponse } from '@vercel/node';
import { TwoFactorService } from '../../../src/lib/two-factor-service.js';
import { verifyToken } from '../../../src/lib/auth-middleware.js';
import { TwoFactorSetupResponseSchema } from '../../../src/lib/two-factor-schemas.js';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method Not Allowed' });
  }

  try {
    // Verify JWT authentication
    const user = verifyToken(req);
    if (!user) {
      return res.status(401).json({
        code: 'AUTH_REQUIRED',
        message: 'Authentication required'
      });
    }

    // Call service to setup 2FA
    const result = await TwoFactorService.setup(user.userId, user.email);

    // Validate response structure
    const validated = TwoFactorSetupResponseSchema.safeParse({
      secret: result.secret,
      otpauth: result.otpauth,
      qrCodeUrl: result.qrCodeUrl,
      backupCodes: result.backupCodes,
    });

    if (!validated.success) {
      return res.status(500).json({
        error: 'Internal Server Error',
        message: 'Invalid response from setup service'
      });
    }

    return res.status(200).json(validated.data);
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    console.error('2FA Setup error:', errorMessage);
    return res.status(500).json({
      error: 'Internal Server Error',
      message: errorMessage
    });
  }
}
