import type { VercelRequest, VercelResponse } from '@vercel/node';
import { AddressWhitelistService } from '../../../src/lib/address-whitelist-service.js';
import { verifyToken } from '../../../src/lib/auth-middleware.js';
import logger from '../../../src/lib/logger.js';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'POST') {
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
    const updated = await AddressWhitelistService.setPrimaryAddress(user.userId, Number(id));

    return res.status(200).json({
      id: updated.id,
      address: updated.address,
      addressType: updated.addressType,
      label: updated.label,
      isPrimary: updated.isPrimary,
      message: 'Primary address set successfully',
    });
  } catch (error: any) {
    logger.error({ error: error.message, userId: user.userId }, 'POST /api/addresses/:id/primary failed');

    if (error.message.includes('not found') || error.message.includes('verified')) {
      return res.status(400).json({ error: error.message });
    }

    return res.status(500).json({ error: 'Failed to set primary address' });
  }
}
