import type { VercelRequest, VercelResponse } from '@vercel/node';
import { AuthService } from '../../../src/lib/auth-service.js';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method not allowed' });
  }

  const { userId, token } = req.body;
  if (!userId || !token) {
    return res.status(400).json({ error: 'Missing required fields' });
  }

  try {
    const result = await AuthService.verify2FAAndLogin(Number(userId), token);
    return res.status(200).json(result);
  } catch (error: any) {
    return res.status(401).json({ error: error.message });
  }
}
