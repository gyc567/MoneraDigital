import type { VercelRequest, VercelResponse } from '@vercel/node';
import { verifyToken } from '../../src/lib/auth-middleware.js';
import logger from '../../src/lib/logger.js';

/**
 * /api/addresses
 *
 * Proxy to Go backend for address management.
 * Handles:
 * - GET /api/addresses (List addresses)
 * - POST /api/addresses (Create address)
 */
export default async function handler(req: VercelRequest, res: VercelResponse) {
  const backendUrl = process.env.BACKEND_URL;

  if (!backendUrl) {
    return res.status(500).json({
      error: 'Server configuration error',
      message: 'Backend URL not configured',
    });
  }

  // Verify authentication token
  const user = verifyToken(req);
  if (!user) {
    return res.status(401).json({
      code: 'MISSING_TOKEN',
      message: 'Authentication required',
    });
  }

  // Allowed methods
  if (req.method !== 'GET' && req.method !== 'POST') {
    return res.status(405).json({ error: 'Method Not Allowed' });
  }

  try {
    const url = `${backendUrl}/api/addresses`;
    
    const options: RequestInit = {
      method: req.method,
      headers: {
        'Content-Type': 'application/json',
        // Forward Authorization header
        ...(req.headers.authorization ? { 'Authorization': req.headers.authorization } : {}),
      },
    };

    if (req.method === 'POST') {
      options.body = JSON.stringify(req.body);
    }

    const response = await fetch(url, options);
    
    // Attempt to parse JSON, default to empty object if fails (e.g. 204 No Content)
    const data = await response.json().catch(() => ({}));

    return res.status(response.status).json(data);
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    logger.error({ error: errorMessage }, 'Addresses proxy error');
    return res.status(500).json({
      error: 'Internal Server Error',
      message: 'Failed to connect to backend service',
    });
  }
}
