import type { VercelRequest, VercelResponse } from '@vercel/node';
import { AuthService } from '../../src/lib/auth-service';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method not allowed' });
  }

  const { email, password } = req.body;

  try {
    const user = await AuthService.register(email, password);
    return res.status(201).json({ message: 'User created successfully', user });
  } catch (error: any) {
    return res.status(400).json({ error: error.message });
  }
}
