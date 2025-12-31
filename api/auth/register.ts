import type { VercelRequest, VercelResponse } from '@vercel/node';
import { AuthService } from '../../src/lib/auth-service';
import { rateLimit } from '../../src/lib/rate-limit';
import { ZodError } from 'zod';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  console.log('Register request received:', { email: req.body?.email, method: req.method });

  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method not allowed' });
  }

  try {
    const ip = (req.headers['x-forwarded-for'] as string) || '127.0.0.1';
    if (!rateLimit(ip, 5, 60000)) {
      return res.status(429).json({ error: 'Too many requests' });
    }

    const { email, password } = req.body;
    const user = await AuthService.register(email, password);
    console.log('User registered successfully:', email);
    return res.status(201).json({ message: 'User created successfully', user });
  } catch (error: any) {
    console.error('Register handler error:', error);
    if (error instanceof ZodError) {
      return res.status(400).json({ error: error.errors[0].message });
    }
    return res.status(400).json({ error: error.message || 'Registration failed' });
  }
}
