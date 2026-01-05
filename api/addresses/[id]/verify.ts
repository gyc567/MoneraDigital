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
  const { token } = req.body;

  if (!id || !token) {
    return res.status(400).json({ error: 'Missing required fields' });
  }

  try {
    const verified = await AddressWhitelistService.verifyAddress(user.userId, token);

    return res.status(200).json({
      id: verified.id,
      address: verified.address,
      addressType: verified.addressType,
      label: verified.label,
      isVerified: verified.isVerified,
      verifiedAt: verified.verifiedAt,
      message: 'Address verified successfully',
    });
  } catch (error: any) {
    logger.error({ error: error.message, userId: user.userId }, 'POST /api/addresses/:id/verify failed');

    if (error.message.includes('Invalid') || error.message.includes('expired')) {
      return res.status(400).json({ error: error.message });
    }

    return res.status(500).json({ error: 'Failed to verify address' });
  }
}
