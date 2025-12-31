import type { VercelRequest, VercelResponse } from '@vercel/node';
import { AuthService } from '../../src/lib/auth-service.js';
import { rateLimit } from '../../src/lib/rate-limit.js';
import { ZodError } from 'zod';
import logger from '../../src/lib/logger.js';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  const requestId = Math.random().toString(36).substring(7);
  const log = logger.child({ requestId, endpoint: '/api/auth/register' });

  log.info({ method: req.method, email: req.body?.email }, 'Register request received');

  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method not allowed' });
  }

  try {
    const ip = (req.headers['x-forwarded-for'] as string) || '127.0.0.1';
    const isAllowed = await rateLimit(ip, 5, 60000);
    
    if (!isAllowed) {
      log.warn({ ip }, 'Rate limit exceeded');
      return res.status(429).json({ error: 'Too many requests' });
    }

    const { email, password } = req.body;
    const user = await AuthService.register(email, password);
    
    return res.status(201).json({ message: 'User created successfully', user });
  } catch (error: any) {
    if (error instanceof ZodError) {
      log.warn({ errors: error.errors }, 'Validation error');
      return res.status(400).json({ error: error.errors[0].message });
    }
    
    log.error({ error: error.message, stack: error.stack }, 'Registration handler failed');
    return res.status(400).json({ error: error.message || 'Registration failed' });
  }
}