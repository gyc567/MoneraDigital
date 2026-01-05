import type { VercelRequest, VercelResponse } from '@vercel/node';
import { AddressWhitelistService } from '../../../src/lib/address-whitelist-service.js';
import { verifyToken } from '../../../src/lib/auth-middleware.js';
import logger from '../../../src/lib/logger.js';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'DELETE') {
    return res.status(405).json({ error: 'Method not allowed' });
  }

  const user = verifyToken(req);
  if (!user) {
    return res.status(401).json({ error: 'Unauthorized' });
  }

  const { id } = req.query;

  if (!id) {
    return res.status(400).json({ error: 'Address ID required' });
  }

  try {
    const deactivated = await AddressWhitelistService.deactivateAddress(user.userId, Number(id));

    return res.status(200).json({
      id: deactivated.id,
      address: deactivated.address,
      deactivatedAt: deactivated.deactivatedAt,
      message: 'Address deactivated successfully',
    });
  } catch (error: any) {
    logger.error({ error: error.message, userId: user.userId }, 'DELETE /api/addresses/:id failed');

    if (error.message.includes('not found')) {
      return res.status(404).json({ error: error.message });
    }

    return res.status(500).json({ error: 'Failed to deactivate address' });
  }
}