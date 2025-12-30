import type { VercelRequest, VercelResponse } from '@vercel/node';
import { AuthService } from '../../src/lib/auth-service.js';
import { rateLimit } from '../../src/lib/rate-limit.js';
import { ZodError } from 'zod';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method not allowed' });
  }

  const ip = (req.headers['x-forwarded-for'] as string) || '127.0.0.1';
  if (!rateLimit(ip, 5, 60000)) { // 每分钟最多 5 次尝试
    return res.status(429).json({ error: 'Too many requests' });
  }

  const { email, password } = req.body;

  try {
    const user = await AuthService.register(email, password);
    return res.status(201).json({ message: 'User created successfully', user });
  } catch (error: any) {
    if (error instanceof ZodError) {
      return res.status(400).json({ error: error.errors[0].message });
    }
    return res.status(400).json({ error: error.message });
  }
}
