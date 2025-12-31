import type { VercelRequest, VercelResponse } from '@vercel/node';
import { LendingService } from '../../src/lib/lending-service.js';
import { verifyToken } from '../../src/lib/auth-middleware.js';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'GET') {
    return res.status(405).json({ error: 'Method not allowed' });
  }

  const user = verifyToken(req);
  if (!user) {
    return res.status(401).json({ error: 'Unauthorized' });
  }

  try {
    const positions = await LendingService.getUserPositions(user.userId);
    return res.status(200).json(positions);
  } catch (error: any) {
    return res.status(500).json({ error: 'Internal server error' });
  }
}
