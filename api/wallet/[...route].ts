import type { VercelRequest, VercelResponse } from '@vercel/node';
import { WalletService } from '../src/lib/wallet-service.js';
import { rateLimit } from '../src/lib/rate-limit.js';
import { verifyToken } from '../src/lib/auth-middleware.js';
import { ZodError } from 'zod';
import logger from '../src/lib/logger.js';

async function checkRateLimit(req: VercelRequest, res: VercelResponse, log: any) {
  const ip = (req.headers['x-forwarded-for'] as string) || '127.0.0.1';
  const isAllowed = await rateLimit(ip, 10, 60000);
  if (!isAllowed) {
    log.warn({ ip }, 'Rate limit exceeded');
    res.status(429).json({ error: 'Too many requests' });
    return false;
  }
  return true;
}

const handlers: Record<string, (req: VercelRequest, res: VercelResponse, log: any) => Promise<any>> = {
  'create': async (req, res, log) => {
    if (req.method !== 'POST') return res.status(405).json({ error: 'Method not allowed' });
    if (!(await checkRateLimit(req, res, log))) return;

    const user = verifyToken(req);
    if (!user) return res.status(401).json({ error: 'Unauthorized' });

    const { request_id, user_id } = req.body;
    if (!request_id || !user_id) {
      return res.status(400).json({ error: 'Missing required fields: request_id, user_id' });
    }

    const result = await WalletService.createWallet(Number(user_id), request_id);
    return res.status(200).json(result);
  },

  'status': async (req, res) => {
    if (req.method !== 'GET') return res.status(405).json({ error: 'Method not allowed' });
    const user = verifyToken(req);
    if (!user) return res.status(401).json({ error: 'Unauthorized' });

    const userId = req.query.user_id;
    if (!userId) return res.status(400).json({ error: 'user_id is required' });

    const result = await WalletService.getWalletStatus(Number(userId));
    return res.status(200).json(result);
  },

  'deposit-address': async (req, res) => {
    if (req.method !== 'GET') return res.status(405).json({ error: 'Method not allowed' });
    const user = verifyToken(req);
    if (!user) return res.status(401).json({ error: 'Unauthorized' });

    const userId = req.query.user_id;
    if (!userId) return res.status(400).json({ error: 'user_id is required' });

    const chain = req.query.chain as string | undefined;
    const result = await WalletService.getDepositAddresses(Number(userId), chain);
    return res.status(200).json(result);
  }
};

export default async function handler(req: VercelRequest, res: VercelResponse) {
  const { route } = req.query;
  const routePath = Array.isArray(route) ? route.join('/') : (route || '');

  const requestId = Math.random().toString(36).substring(7);
  const log = logger.child({ requestId, endpoint: `/api/wallet/${routePath}` });

  log.info({ method: req.method, query: req.query }, 'Wallet request received');

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

    if (error.message?.includes('No wallet account found')) {
      return res.status(404).json({ error: error.message });
    }

    const status = error.message?.includes('Unauthorized') ? 401 : 500;
    return res.status(status).json({ error: error.message || 'Internal server error' });
  }
}
