import type { VercelRequest, VercelResponse } from '@vercel/node';
import { TwoFactorService } from '../../../src/lib/two-factor-service.js';
import { verifyToken } from '../../../src/lib/auth-middleware.js';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method not allowed' });
  }

  const user = verifyToken(req);
  if (!user) {
    return res.status(401).json({ error: 'Unauthorized' });
  }

  const { token } = req.body;
  if (!token) {
    return res.status(400).json({ error: 'Verification code is required' });
  }

  try {
    await TwoFactorService.enable(user.userId, token);
    return res.status(200).json({ message: '2FA enabled successfully' });
  } catch (error: any) {
    return res.status(400).json({ error: error.message });
  }
}
