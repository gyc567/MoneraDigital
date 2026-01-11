import type { VercelRequest, VercelResponse } from '@vercel/node';
import { AuthService } from '../../src/lib/auth-service.js';
import { TwoFactorService } from '../../src/lib/two-factor-service.js';
import { rateLimit } from '../../src/lib/rate-limit.js';
import { verifyToken } from '../../src/lib/auth-middleware.js';
import { db } from '../../src/lib/db.js';
import { users } from '../../src/db/schema.js';
import { eq } from 'drizzle-orm';
import { ZodError } from 'zod';
import logger from '../../src/lib/logger.js';

// Helper to handle rate limiting
async function checkRateLimit(req: VercelRequest, res: VercelResponse, log: any) {
  const ip = (req.headers['x-forwarded-for'] as string) || '127.0.0.1';
  const isAllowed = await rateLimit(ip, 5, 60000);
  if (!isAllowed) {
    log.warn({ ip }, 'Rate limit exceeded');
    res.status(429).json({ error: 'Too many requests' });
    return false;
  }
  return true;
}

// Route Handlers
const handlers: Record<string, (req: VercelRequest, res: VercelResponse, log: any) => Promise<any>> = {
  'login': async (req, res, log) => {
    if (req.method !== 'POST') return res.status(405).json({ error: 'Method not allowed' });
    if (!(await checkRateLimit(req, res, log))) return;
    
    const { email, password } = req.body;
    const result = await AuthService.login(email, password);
    return res.status(200).json(result);
  },

  'register': async (req, res, log) => {
    if (req.method !== 'POST') return res.status(405).json({ error: 'Method not allowed' });
    if (!(await checkRateLimit(req, res, log))) return;

    const { email, password } = req.body;
    const user = await AuthService.register(email, password);
    return res.status(201).json({ message: 'User created successfully', user });
  },

  'me': async (req, res) => {
    if (req.method !== 'GET') return res.status(405).json({ error: 'Method not allowed' });
    const authUser = verifyToken(req);
    if (!authUser) return res.status(401).json({ error: 'Unauthorized' });

    const [user] = await db.select({
      id: users.id,
      email: users.email,
      twoFactorEnabled: users.twoFactorEnabled,
    })
    .from(users)
    .where(eq(users.id, authUser.userId));

    if (!user) return res.status(404).json({ error: 'User not found' });
    return res.status(200).json(user);
  },

  '2fa/setup': async (req, res, log) => {
    if (req.method !== 'POST') return res.status(405).json({ error: 'Method not allowed' });
    const user = verifyToken(req);
    if (!user) return res.status(401).json({ error: 'Unauthorized' });

    const data = await TwoFactorService.setup(user.userId, user.email);
    return res.status(200).json(data);
  },

  '2fa/enable': async (req, res) => {
    if (req.method !== 'POST') return res.status(405).json({ error: 'Method not allowed' });
    const user = verifyToken(req);
    if (!user) return res.status(401).json({ error: 'Unauthorized' });

    const { token } = req.body;
    if (!token) return res.status(400).json({ error: 'Verification code is required' });

    await TwoFactorService.enable(user.userId, token);
    return res.status(200).json({ message: '2FA enabled successfully' });
  },

  '2fa/verify-login': async (req, res, log) => {
    if (req.method !== 'POST') return res.status(405).json({ error: 'Method not allowed' });
    if (!(await checkRateLimit(req, res, log))) return;

    const { userId, token } = req.body;
    if (!userId || !token) {
      return res.status(400).json({ error: 'Missing required fields: userId and token' });
    }

    // Pre-check user existence
    const [user] = await db.select({ id: users.id }).from(users).where(eq(users.id, Number(userId)));
    if (!user) {
      log.warn({ userId }, '2FA verify failed: user not found');
      return res.status(401).json({ code: 'INVALID_CREDENTIALS', message: 'Invalid user or token' });
    }

    try {
      const result = await AuthService.verify2FAAndLogin(Number(userId), token);
      return res.status(200).json(result);
    } catch (error: any) {
      if (error.message === '2FA not enabled' || error.message === 'Invalid verification code') {
        log.warn({ userId }, '2FA verification failed');
        return res.status(401).json({ code: 'INVALID_2FA_TOKEN', message: 'Invalid or expired 2FA token' });
      }
      throw error;
    }
  }
};

export default async function handler(req: VercelRequest, res: VercelResponse) {
  const { route } = req.query;
  const routePath = Array.isArray(route) ? route.join('/') : (route || '');
  
  const requestId = Math.random().toString(36).substring(7);
  const log = logger.child({ requestId, endpoint: `/api/auth/${routePath}` });

  log.info({ method: req.method }, 'Auth request received');

  const handler = handlers[routePath];

  if (!handler) {
    return res.status(404).json({ error: 'Not found' });
  }

  try {
    await handler(req, res, log);
  } catch (error: any) {
    if (error instanceof ZodError) {
      log.warn({ errors: error.errors }, 'Validation error');
      return res.status(400).json({ error: error.errors[0].message });
    }
    
    log.error({ error: error.message, stack: error.stack }, 'Handler failed');
    const status = error.message?.includes('Unauthorized') || error.message?.includes('invalid') ? 401 : 400;
    return res.status(status).json({ error: error.message || 'Internal server error' });
  }
}
