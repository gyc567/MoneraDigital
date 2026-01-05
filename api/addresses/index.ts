import type { VercelRequest, VercelResponse } from '@vercel/node';
import { AddressWhitelistService } from '../../src/lib/address-whitelist-service.js';
import { EmailService } from '../../src/lib/email-service.js';
import { verifyToken } from '../../src/lib/auth-middleware.js';
import logger from '../../src/lib/logger.js';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method === 'GET') {
    // Get all addresses for user
    const user = verifyToken(req);
    if (!user) {
      return res.status(401).json({ error: 'Unauthorized' });
    }

    try {
      const addresses = await AddressWhitelistService.getAddresses(user.userId);
      return res.status(200).json({ addresses });
    } catch (error: any) {
      logger.error({ error: error.message, userId: user.userId }, 'GET /api/addresses failed');
      return res.status(500).json({ error: 'Failed to fetch addresses' });
    }
  }

  if (req.method === 'POST') {
    // Add new address
    const user = verifyToken(req);
    if (!user) {
      return res.status(401).json({ error: 'Unauthorized' });
    }

    const { address, addressType, label } = req.body;

    if (!address || !addressType || !label) {
      return res.status(400).json({ error: 'Missing required fields' });
    }

    try {
      // Add the address
      const newAddress = await AddressWhitelistService.addAddress(
        user.userId,
        address,
        addressType,
        label
      );

      // Generate verification token
      const verificationToken = await AddressWhitelistService.generateVerificationToken(newAddress.id);

      // Send verification email
      try {
        await EmailService.sendVerificationEmail(user.email, verificationToken);
      } catch (emailError: any) {
        logger.warn({ error: emailError.message }, 'Failed to send verification email, but address was created');
      }

      return res.status(201).json({
        id: newAddress.id,
        address: newAddress.address,
        addressType: newAddress.addressType,
        label: newAddress.label,
        isVerified: newAddress.isVerified,
        message: 'Address added successfully. Verification email sent.',
      });
    } catch (error: any) {
      logger.error({ error: error.message, userId: user.userId }, 'POST /api/addresses failed');

      if (error.message.includes('Invalid')) {
        return res.status(400).json({ error: error.message });
      }

      return res.status(500).json({ error: 'Failed to add address' });
    }
  }

  return res.status(405).json({ error: 'Method not allowed' });
}
